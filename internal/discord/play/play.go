package play

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"net/url"

	"github.com/LeBulldoge/gungus/internal/discord/format"
	"github.com/LeBulldoge/gungus/internal/discord/play/playback"
	"github.com/LeBulldoge/gungus/internal/youtube"
	"github.com/bwmarrin/discordgo"
)

func autocompleteResponse(choices []*discordgo.ApplicationCommandOptionChoice) *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	}
}

func (c *PlayCommand) handlePlayAutocomplete(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	opt := intr.ApplicationCommandData()
	queryString := opt.Options[0].StringValue()
	log := c.logger.WithGroup("play/autocomplete").With("query", queryString)

	choices := []*discordgo.ApplicationCommandOptionChoice{}
	defer func() {
		log.Info("choices collected", "count", len(choices))
		if err := session.InteractionRespond(intr.Interaction, autocompleteResponse(choices)); err != nil {
			log.Error("failed to respond", "err", err)
		}
	}()

	if len(queryString) < 3 {
		log.Info("search string is less than 3")
		return
	}

	if _, err := url.ParseRequestURI(queryString); err == nil {
		log.Info("skipping autocomplete")
		return
	}

	log.Info("searching for videos")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	ytDataChan := make(chan youtube.YoutubeDataResult, 5)
	if err := youtube.SearchYoutube(ctx, queryString, ytDataChan); err != nil {
		log.Error("error getting youtube data", "err", err)
		return
	}

	for ytData := range ytDataChan {
		if ytData.Error != nil {
			log.Error("failure getting result from SearchYoutube", "err", ytData.Error)
			continue
		}
		video := ytData.Data

		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  video.Title,
			Value: video.Url,
		})
	}
}

func (c *PlayCommand) HandlePlay(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	if intr.Type == discordgo.InteractionApplicationCommandAutocomplete {
		c.handlePlayAutocomplete(session, intr)
		return
	}
	opt := intr.ApplicationCommandData()
	queryString := opt.Options[0].StringValue()

	log := c.logger.With("query", queryString)

	var videoUrl string
	if url, err := url.ParseRequestURI(queryString); err != nil {
		log.Error("error parsing url", "err", err)
		format.DisplayInteractionError(session, intr, "Error parsing url!")
		return
	} else {
		if url.Host != "www.youtube.com" {
			log.Error("error parsing url: incorrect domain")
			format.DisplayInteractionError(session, intr, "Domain must be `www.youtube.com`, not `"+url.Host+"`")
			return
		}
		videoUrl = url.String()
	}

	log.Info("requesting video data", "url", videoUrl)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ytDataChan := make(chan youtube.YoutubeDataResult)
	if err := youtube.GetYoutubeData(ctx, videoUrl, ytDataChan); err != nil {
		log.Error("error getting youtube data", "err", err)
		format.DisplayInteractionError(session, intr, "Error getting video data from youtube. See the log for details.")
		return
	}

	err := session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		log.Error("failure responding to interaction", "err", err)
		return
	}

	var playbackService *playback.PlaybackService
	if ps := c.playbackManager.Get(intr.GuildID); ps != nil {
		log.Info("get stored playbackService")
		playbackService = ps
	} else {
		log.Info("creating new playbackService")

		channelId, err := c.getUserChannelId(session, intr.Member.User.ID, intr.GuildID)
		if err != nil {
			log.Error("failure getting channel id", "err", err)
			format.DisplayInteractionError(session, intr, "You must be in a voice channel to use this command.")
			return
		}

		voice, err := session.ChannelVoiceJoin(intr.GuildID, channelId, false, true)
		if err != nil {
			log.Error("failure joining voice channel", "channelId", channelId, "err", err)
			format.DisplayInteractionError(session, intr, "Error joining voice channel.")
			return
		}

		playbackService = playback.NewPlaybackService(voice)
		if err := c.playbackManager.Add(intr.GuildID, playbackService); err != nil {
			log.Error("error adding a new playback service", "guildId", intr.GuildID, "err", err)
			format.DisplayInteractionError(session, intr, "Error starting playback.")
			return
		}
	}

	if !playbackService.IsRunning() {
		var wg sync.WaitGroup
		go func(guildId string) {
			pCtx, pCancel := context.WithCancel(context.Background())
			defer pCancel()

			stopHandlerCancel := createStopHandler(session, pCancel, guildId)
			go func(channelId string) {
				tick := time.NewTicker(time.Minute)
				defer tick.Stop()
				for {
					select {
					case <-pCtx.Done():
						return
					case <-tick.C:
						log.Info("PlaybackService: checking if bot is last in server...")
						if ok, err := c.isBotLastInChannel(session, guildId, channelId); err != nil {
							log.Error("PlaybackService: timeout ticker error", "err", err)
							return
						} else if ok {
							log.Info("PlaybackService: bot is last in server, cancelling playback")
							pCancel()
							return
						} else {
							log.Info("PlaybackService: bot is not last in server, continuing playback")
						}
					}
				}
			}(playbackService.ChannelId())

			wg.Add(1)
			err := playbackService.Run(pCtx, &wg)
			if err != nil {
				log.Error("playback error has occured", "err", err)
			}

			stopHandlerCancel()

			if err := playbackService.Cleanup(); err != nil {
				log.Error("failure to close playbackService", "err", err)
			}

			log.Info("deleting playbackService", "guildId", guildId)
			if err := c.playbackManager.Delete(guildId); err != nil {
				log.Error("error deleting playbackService", "guildId", guildId, "err", err)
			}
		}(intr.GuildID)
		wg.Wait()
	}

	for ytData := range ytDataChan {
		if ytData.Error != nil {
			log.Error("failure getting url from GetYoutubeData", "err", ytData.Error)
			continue
		}
		video := ytData.Data

		err := playbackService.EnqueueVideo(video)
		if err != nil {
			log.Info("failed to queue a video", "err", err)
			return
		}

		log.Info("added video to playbackService", "video", video.Title)

		embed := &discordgo.MessageEmbed{
			Author: &discordgo.MessageEmbedAuthor{
				Name: "Added to queue",
			},
			Title: video.Title,
			URL:   videoUrl,
			Thumbnail: &discordgo.MessageEmbedThumbnail{
				URL: video.Thumbnail,
			},
			Description: video.Length,
			Footer: &discordgo.MessageEmbedFooter{
				Text: fmt.Sprintf("Queue length: %d", playbackService.Count()),
			},
		}

		_, err = session.FollowupMessageCreate(intr.Interaction, false, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{embed},
		})
		if err != nil {
			log.Error("failure creating followup message to interaction", "err", err)
			return
		}
	}
}

func createStopHandler(sesh *discordgo.Session, cancel context.CancelFunc, guildId string) func() {
	return sesh.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.GuildID != guildId {
			return
		}

		opt := i.ApplicationCommandData()
		if opt.Name != "stop" {
			return
		}

		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Stopping playback.",
			},
		})
		if err != nil {
			format.DisplayInteractionError(s, i, "Failure responding to interaction. See the log for details.")
		}

		cancel()
	})
}

func (c *PlayCommand) getUserChannelId(sesh *discordgo.Session, userId string, guildId string) (string, error) {
	var channelId string

	g, err := sesh.State.Guild(guildId)
	if err != nil {
		return channelId, fmt.Errorf("failure getting guild: %w", err)
	}

	c.logger.Info("guild acquired", "guildId", g.ID, "name", g.Name)

	for _, vs := range g.VoiceStates {
		c.logger.Info("user in channel", "usr", vs.UserID, "chn", vs.ChannelID)
		if vs.UserID == userId {
			channelId = vs.ChannelID
			break
		}
	}
	if len(channelId) == 0 {
		return channelId, errors.New("user is not in any voice channels")
	}

	return channelId, nil
}

func (c *PlayCommand) isBotLastInChannel(sesh *discordgo.Session, guildId string, channelId string) (bool, error) {
	g, err := sesh.State.Guild(guildId)
	if err != nil {
		return false, fmt.Errorf("failure getting guild: %w", err)
	}

	for _, vs := range g.VoiceStates {
		if vs.UserID != sesh.State.User.ID && vs.ChannelID == channelId {
			return false, nil
		}
	}

	return true, nil
}
