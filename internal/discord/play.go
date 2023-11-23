package discord

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"sync"
	"time"

	"github.com/LeBulldoge/gungus/internal/discord/playback"
	"github.com/LeBulldoge/gungus/internal/youtube"
	"github.com/bwmarrin/discordgo"
)

func handlePlay(bot *Bot, intr *discordgo.InteractionCreate) {
	opt := intr.ApplicationCommandData()
	queryString := opt.Options[0].StringValue()

	var videoUrl string
	if url, err := url.ParseRequestURI(queryString); err != nil {
		displayInteractionError(bot.session, intr.Interaction, fmt.Sprintf("error parsing url: %s", err))
		return
	} else {
		if url.Host != "www.youtube.com" {
			displayInteractionError(bot.session, intr.Interaction, "error parsing url: domain must be `www.youtube.com`, not "+url.Host)
			return
		}
		videoUrl = url.String()
	}

	slog.Info("requesting video data", "url", videoUrl)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ytDataChan := make(chan youtube.YoutubeDataResult)
	if err := youtube.GetYoutubeData(ctx, videoUrl, ytDataChan); err != nil {
		displayInteractionError(bot.session, intr.Interaction, fmt.Sprintf("error getting youtube data: %s", err))
		return
	}

	slog.Info("requesting video data started", "url", videoUrl)

	err := bot.session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		slog.Error("failure responding to interaction", "err", err)
		return
	}

	var playbackService *playback.PlaybackService
	if ps := bot.playbackManager.Get(intr.GuildID); ps != nil {
		slog.Info("get stored playbackService", "url", videoUrl)
		playbackService = ps
	} else {
		slog.Info("creating new playbackService", "url", videoUrl)

		channelId, err := getUserChannelId(bot.session, intr.Member.User.ID, intr.GuildID)
		if err != nil {
			displayInteractionError(bot.session, intr.Interaction, fmt.Sprintf("failure getting channel id: %s", err))
			return
		}

		voice, err := bot.session.ChannelVoiceJoin(intr.GuildID, channelId, false, true)
		if err != nil {
			displayInteractionError(bot.session, intr.Interaction, fmt.Sprintf("failure joining channel: %s", err))
			return
		}

		playbackService = playback.NewPlaybackService(voice)
		if err := bot.playbackManager.Add(intr.GuildID, playbackService); err != nil {
			slog.Error("error adding a new playback service", "guildId", intr.GuildID, "err", err)
		}
	}

	if !playbackService.IsRunning() {
		var wg sync.WaitGroup
		go func(guildId string) {
			pCtx, pCancel := context.WithCancel(context.Background())
			defer pCancel()

			stopHandlerCancel := createStopHandler(bot.session, pCancel, guildId)
			go func(channelId string, botUserId string) {
				tick := time.NewTicker(time.Minute)
				for {
					select {
					case <-pCtx.Done():
					case <-tick.C:
						slog.Info("PlaybackService: checking if bot is last in server...")
						if ok, err := isBotLastInChannel(bot.session, botUserId, guildId, channelId); err != nil {
							slog.Error("PlaybackService: timeout ticker error", "err", err)
						} else if ok {
							slog.Info("PlaybackService: bot is last in server, cancelling playback")
							pCancel()
						} else {
							slog.Info("PlaybackService: bot is not last in server, continuing playback")
						}
					}
				}
			}(playbackService.ChannelId(), playbackService.UserId())

			wg.Add(1)
			err := playbackService.Run(pCtx, &wg)
			if err != nil {
				slog.Error("playback error has occured", "err", err)
			}

			stopHandlerCancel()

			if err := playbackService.Cleanup(); err != nil {
				slog.Error("failure to close playbackService", "err", err)
			}

			slog.Info("deleting playbackService", "guildId", guildId)
			if err := bot.playbackManager.Delete(guildId); err != nil {
				slog.Error("error deleting playbackService", "guildId", guildId, "err", err)
			}
		}(intr.GuildID)
		wg.Wait()
	}

	for ytData := range ytDataChan {
		if ytData.Error != nil {
			slog.Error("failure adding url to queue", "err", err)
			continue
		}
		video := ytData.Data

		err := playbackService.EnqueueVideo(video)
		if err != nil {
			slog.Info("failed to queue a video", "err", err)
			return
		}

		slog.Info("added video to playbackService", "video", video.Title)

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

		_, err = bot.session.FollowupMessageCreate(intr.Interaction, false, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{embed},
		})
		if err != nil {
			slog.Error("failure creating followup message to interaction", "err", err)
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
			slog.Error("failure responding to interaction", "err", err)
		}

		cancel()
	})
}

func getUserChannelId(sesh *discordgo.Session, userId string, guildId string) (string, error) {
	var channelId string

	g, err := sesh.State.Guild(guildId)
	if err != nil {
		return channelId, fmt.Errorf("failure getting guild: %w", err)
	}

	slog.Info("guild acquired", "guildId", g.ID, "name", g.Name)

	for _, vs := range g.VoiceStates {
		slog.Info("user in channel", "usr", vs.UserID, "chn", vs.ChannelID)
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

func isBotLastInChannel(sesh *discordgo.Session, botUserId string, guildId string, channelId string) (bool, error) {
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
