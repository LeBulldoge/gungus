package play

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"net/url"

	"github.com/LeBulldoge/gungus/internal/discord/embed"
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
	log := c.logger.With(slog.Group("play/autocomplete", "query", queryString))

	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0, 5)
	defer func() {
		if err := session.InteractionRespond(intr.Interaction, autocompleteResponse(choices)); err != nil {
			log.Error("failed to respond", "err", err)
		}
		log.Info("choices collected", "count", len(choices))
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
	ctx, cancel := context.WithTimeout(context.Background(), 2900*time.Millisecond)
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

	if err := c.isUserAndBotInSameChannel(session, intr.GuildID, intr.Member.User.ID); err != nil {
		switch {
		case errors.Is(err, errUserNotInAnyChannel):
			fallthrough
		case errors.Is(err, errUserNotInBotsChannel):
			format.DisplayInteractionError(session, intr, "You must be in the same voice channel as the bot to use this command.")
			return
		}
	}

	var playbackService *playback.PlaybackService
	if ps := c.playbackManager.Get(intr.GuildID); ps != nil {
		log.Info("get stored playbackService")
		playbackService = ps
	} else {
		log.Info("creating new playbackService")

		channelId, err := c.getUserChannelId(session, intr.GuildID, intr.Member.User.ID)
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

		var wg sync.WaitGroup
		wg.Add(1)
		playbackService = c.setupPlaybackService(session, intr, voice, log, &wg)
		if playbackService == nil {
			format.DisplayInteractionError(session, intr, "Error starting playback.")
			return
		}
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

		embed := embed.NewEmbed().
			SetAuthor("Added to queue").
			SetTitle(video.Title).
			SetUrl(video.Url).
			SetThumbnail(video.Thumbnail).
			SetDescription(video.Length).
			SetFooter(fmt.Sprintf("Queue length: %d", playbackService.Count()), "").
			MessageEmbed

		_, err = session.FollowupMessageCreate(intr.Interaction, false, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{embed},
		})
		if err != nil {
			log.Error("failure creating followup message to interaction", "err", err)
			return
		}
	}
}

func (c *PlayCommand) setupPlaybackService(session *discordgo.Session, intr *discordgo.InteractionCreate, voice *discordgo.VoiceConnection, log *slog.Logger, wg *sync.WaitGroup) *playback.PlaybackService {
	playbackService := playback.NewPlaybackService(voice)
	if err := c.playbackManager.Add(intr.GuildID, playbackService); err != nil {
		log.Error("error adding a new playback service", "guildId", intr.GuildID, "err", err)
		return nil
	}

	// Run the service
	go func(guildId string) {
		playbackContext, playbackCancel := context.WithCancelCause(context.Background())
		stopHandlerCancel := createStopHandler(session, playbackCancel, guildId)

		// Setup service timeout ticker, in case bot is left alone in a channel
		go func(channelId string) {
			tick := time.NewTicker(time.Minute)
			defer tick.Stop()
			for {
				select {
				case <-playbackContext.Done():
					return
				case <-tick.C:
					log.Info("PlaybackService: checking if bot is last in server...")
					if ok, err := c.isBotLastInVoiceChannel(session, guildId, channelId); err != nil {
						log.Error("PlaybackService: timeout ticker error", "err", err)
						return
					} else if ok {
						log.Info("PlaybackService: bot is last in server, cancelling playback")
						playbackCancel(playback.ErrCauseStop)
						return
					} else {
						log.Info("PlaybackService: bot is not last in server, continuing playback")
					}
				}
			}
		}(playbackService.ChannelId())

		err := playbackService.Run(playbackContext, wg)
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

	return playbackService
}

func createStopHandler(sesh *discordgo.Session, cancel context.CancelCauseFunc, guildId string) func() {
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

		cancel(playback.ErrCauseStop)
	})
}

func (c *PlayCommand) HandleSkip(sesh *discordgo.Session, intr *discordgo.InteractionCreate) {
	guildId := intr.GuildID
	userId := intr.Member.User.ID

	if err := c.isUserAndBotInSameChannel(sesh, guildId, userId); err != nil {
		switch {
		case errors.Is(err, errUserNotInAnyChannel):
			fallthrough
		case errors.Is(err, errUserNotInBotsChannel):
			format.DisplayInteractionError(sesh, intr, "You must be in the same voice channel as the bot to use this command.")
			return
		case errors.Is(err, errBotIsNotInAnyChannel):
			format.DisplayInteractionError(sesh, intr, "Nothing to skip.")
			return
		}
	}

	if ps := c.playbackManager.Get(guildId); ps != nil {
		err := ps.Skip()
		if errors.Is(err, playback.ErrSkipUnavailable) {
			format.DisplayInteractionError(sesh, intr, "Nothing to skip yet.")
			return
		}
	} else {
		format.DisplayInteractionError(sesh, intr, "Nothing to skip.")
		return
	}

	err := sesh.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Skipped current song.",
		},
	})
	if err != nil {
		slog.Error("failure responding to interaction", "err", err)
		format.DisplayInteractionError(sesh, intr, "Failure responding to interaction. See the log for details.")
	}
}

var (
	errBotIsNotInAnyChannel = errors.New("bot isn't in any channels")
	errUserNotInAnyChannel  = errors.New("you must be in a voice channel")
	errUserNotInBotsChannel = errors.New("you must be in the same channel as the bot")
)

func (c *PlayCommand) isUserAndBotInSameChannel(sesh *discordgo.Session, guildId string, userId string) error {
	botUserId := sesh.State.User.ID

	botChannelId, err := c.getUserChannelId(sesh, guildId, botUserId)
	if err != nil {
		return errBotIsNotInAnyChannel
	}

	channelId, err := c.getUserChannelId(sesh, guildId, userId)
	if err != nil {
		return errUserNotInAnyChannel
	}

	if channelId != botChannelId {
		return errUserNotInBotsChannel
	}

	return nil
}

func (c *PlayCommand) getUserChannelId(sesh *discordgo.Session, guildId string, userId string) (string, error) {
	var channelId string

	g, err := sesh.State.Guild(guildId)
	if err != nil {
		if !errors.Is(err, discordgo.ErrStateNotFound) {
			return channelId, fmt.Errorf("failure getting guild: %w", err)
		}

		g, err = sesh.Guild(guildId)
		if err != nil {
			return channelId, fmt.Errorf("failure getting guild: %w", err)
		}
	}

	c.logger.Info("guild acquired", "guildId", g.ID, "name", g.Name)

	for _, vs := range g.VoiceStates {
		if vs.UserID == userId {
			c.logger.Info("user found in channel", "usr", vs.UserID, "chn", vs.ChannelID)
			channelId = vs.ChannelID
			break
		}
	}
	if len(channelId) == 0 {
		return channelId, errors.New("user is not in any voice channels")
	}

	return channelId, nil
}

func (c *PlayCommand) isBotLastInVoiceChannel(sesh *discordgo.Session, guildId string, channelId string) (bool, error) {
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
