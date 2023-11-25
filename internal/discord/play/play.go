package play

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"log/slog"
	"net/url"

	"github.com/LeBulldoge/gungus/internal/discord/bot"
	"github.com/LeBulldoge/gungus/internal/discord/format"
	"github.com/LeBulldoge/gungus/internal/discord/play/playback"
	"github.com/LeBulldoge/gungus/internal/youtube"
	"github.com/bwmarrin/discordgo"
)

type PlayCommand struct {
	playbackManager playback.PlaybackServiceManager

	logger *slog.Logger
}

func NewCommand() *PlayCommand {
	return &PlayCommand{
		playbackManager: playback.NewManager(),
	}
}

func (c *PlayCommand) GetSignature() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "play",
			Description: "Play a youtube video",
			Type:        discordgo.ChatApplicationCommand,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "link",
					Description: "Link to the video",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
				},
			},
		},
		{
			Name:        "stop",
			Description: "Stop audio playback",
			Type:        discordgo.ChatApplicationCommand,
		},
	}
}

func (c *PlayCommand) Setup(bot *bot.Bot) error {
	bot.Session.AddHandler(func(sesh *discordgo.Session, intr *discordgo.InteractionCreate) {
		if intr.Type != discordgo.InteractionApplicationCommand {
			return
		}
		opt := intr.ApplicationCommandData()
		switch opt.Name {
		case "play":
			c.HandlePlay(sesh, intr)
		}
	})

	return nil
}

func (c *PlayCommand) Cleanup(bot *bot.Bot) error {
	return nil
}

func (c *PlayCommand) AddLogger(logger *slog.Logger) {
	if logger == nil {
		return
	}

	c.logger = logger
}

func (c *PlayCommand) HandlePlay(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	opt := intr.ApplicationCommandData()
	queryString := opt.Options[0].StringValue()

	var videoUrl string
	if url, err := url.ParseRequestURI(queryString); err != nil {
		c.logger.Error("error parsing url", "url", queryString, "err", err)
		format.DisplayInteractionError(session, intr.Interaction, "Error parsing url!")
		return
	} else {
		if url.Host != "www.youtube.com" {
			format.DisplayInteractionError(session, intr.Interaction, "error parsing url: domain must be `www.youtube.com`, not "+url.Host)
			return
		}
		videoUrl = url.String()
	}

	c.logger.Info("requesting video data", "url", videoUrl)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ytDataChan := make(chan youtube.YoutubeDataResult)
	if err := youtube.GetYoutubeData(ctx, videoUrl, ytDataChan); err != nil {
		format.DisplayInteractionError(session, intr.Interaction, fmt.Sprintf("error getting youtube data: %s", err))
		return
	}

	c.logger.Info("requesting video data started", "url", videoUrl)

	err := session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		c.logger.Error("failure responding to interaction", "err", err)
		return
	}

	var playbackService *playback.PlaybackService
	if ps := c.playbackManager.Get(intr.GuildID); ps != nil {
		c.logger.Info("get stored playbackService", "url", videoUrl)
		playbackService = ps
	} else {
		c.logger.Info("creating new playbackService", "url", videoUrl)

		channelId, err := c.getUserChannelId(session, intr.Member.User.ID, intr.GuildID)
		if err != nil {
			format.DisplayInteractionError(session, intr.Interaction, fmt.Sprintf("failure getting channel id: %s", err))
			return
		}

		voice, err := session.ChannelVoiceJoin(intr.GuildID, channelId, false, true)
		if err != nil {
			format.DisplayInteractionError(session, intr.Interaction, fmt.Sprintf("failure joining channel: %s", err))
			return
		}

		playbackService = playback.NewPlaybackService(voice)
		if err := c.playbackManager.Add(intr.GuildID, playbackService); err != nil {
			c.logger.Error("error adding a new playback service", "guildId", intr.GuildID, "err", err)
		}
	}

	if !playbackService.IsRunning() {
		var wg sync.WaitGroup
		go func(guildId string) {
			pCtx, pCancel := context.WithCancel(context.Background())
			defer pCancel()

			stopHandlerCancel := createStopHandler(session, pCancel, guildId)
			go func(channelId string, botUserId string) {
				tick := time.NewTicker(time.Minute)
				defer tick.Stop()
				for {
					select {
					case <-pCtx.Done():
						return
					case <-tick.C:
						c.logger.Info("PlaybackService: checking if bot is last in server...")
						if ok, err := c.isBotLastInChannel(session, botUserId, guildId, channelId); err != nil {
							c.logger.Error("PlaybackService: timeout ticker error", "err", err)
							return
						} else if ok {
							c.logger.Info("PlaybackService: bot is last in server, cancelling playback")
							pCancel()
							return
						} else {
							c.logger.Info("PlaybackService: bot is not last in server, continuing playback")
						}
					}
				}
			}(playbackService.ChannelId(), playbackService.UserId())

			wg.Add(1)
			err := playbackService.Run(pCtx, &wg)
			if err != nil {
				c.logger.Error("playback error has occured", "err", err)
			}

			stopHandlerCancel()

			if err := playbackService.Cleanup(); err != nil {
				c.logger.Error("failure to close playbackService", "err", err)
			}

			c.logger.Info("deleting playbackService", "guildId", guildId)
			if err := c.playbackManager.Delete(guildId); err != nil {
				c.logger.Error("error deleting playbackService", "guildId", guildId, "err", err)
			}
		}(intr.GuildID)
		wg.Wait()
	}

	for ytData := range ytDataChan {
		if ytData.Error != nil {
			c.logger.Error("failure adding url to queue", "err", err)
			continue
		}
		video := ytData.Data

		err := playbackService.EnqueueVideo(video)
		if err != nil {
			c.logger.Info("failed to queue a video", "err", err)
			return
		}

		c.logger.Info("added video to playbackService", "video", video.Title)

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
			c.logger.Error("failure creating followup message to interaction", "err", err)
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
			format.DisplayInteractionError(s, i.Interaction, "failure responding to interaction")
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

func (c *PlayCommand) isBotLastInChannel(sesh *discordgo.Session, botUserId string, guildId string, channelId string) (bool, error) {
	g, err := sesh.State.Guild(guildId)
	if err != nil {
		return false, fmt.Errorf("failure getting guild: %w", err)
	}

	for _, vs := range g.VoiceStates {
		if vs.UserID != botUserId && vs.ChannelID == channelId {
			return false, nil
		}
	}

	return true, nil
}
