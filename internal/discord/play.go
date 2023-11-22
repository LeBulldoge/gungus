package discord

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"sync"

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

		g, err := bot.session.State.Guild(intr.GuildID)
		if err != nil {
			displayInteractionError(bot.session, intr.Interaction, fmt.Sprintf("failure getting guild: %s", err))
			return
		}

		slog.Info("guild acquired", "guildId", g.ID, "name", g.Name)

		var channelId string
		for _, vs := range g.VoiceStates {
			slog.Info("user in channel", "usr", vs.UserID, "chn", vs.ChannelID)
			if vs.UserID == intr.Member.User.ID {
				channelId = vs.ChannelID
				break
			}
		}
		if len(channelId) == 0 {
			displayInteractionError(bot.session, intr.Interaction, "failure joining channel: user is not in any channels")
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
