package play

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"sync"
	"time"

	"github.com/LeBulldoge/gungus/internal/discord/commands/play/playback"
	"github.com/LeBulldoge/gungus/internal/discord/embed"
	"github.com/LeBulldoge/gungus/internal/discord/format"
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

func (c *Command) handlePlayAutocomplete(session *discordgo.Session, intr *discordgo.InteractionCreate) {
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

	cancelKey := intr.Member.User.ID
	if cancel, ok := c.autocompleteCancelMap[cancelKey]; ok {
		log.Info("cancelling ongoing autocomplete process")
		cancel()
		delete(c.autocompleteCancelMap, cancelKey)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2900*time.Millisecond)
	defer cancel()

	c.autocompleteCancelMap[cancelKey] = cancel
	defer delete(c.autocompleteCancelMap, intr.Member.User.ID)

	ytDataChan := make(chan youtube.SearchResult, 5)
	if err := youtube.SearchYoutube(ctx, queryString, ytDataChan); err != nil {
		log.Error("error getting youtube data", "err", err)
		return
	}

	for ytData := range ytDataChan {
		if ytData.Error != nil {
			log.Error("failure getting result from SearchYoutube", "err", ytData.Error)
			continue
		}
		video := ytData.Video

		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  video.Title,
			Value: video.URL,
		})
	}
}

var allowedHosts = []string{
	"www.youtube.com",
	"youtube.com",
	"youtu.be",
}

func isHostAllowed(host string) bool {
	for _, h := range allowedHosts {
		if host == h {
			return true
		}
	}
	return false
}

func (c *Command) handlePlay(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	if intr.Type == discordgo.InteractionApplicationCommandAutocomplete {
		c.handlePlayAutocomplete(session, intr)
		return
	}
	opt := intr.ApplicationCommandData()
	queryString := opt.Options[0].StringValue()
	index := -1
	if len(opt.Options) > 1 {
		index = int(opt.Options[1].IntValue())
	}

	log := c.logger.With("query", queryString)

	url, err := url.ParseRequestURI(queryString)
	if err != nil {
		log.Error("error parsing url", "err", err)
		format.DisplayInteractionError(session, intr, "Error parsing url!")
		return
	}
	if !isHostAllowed(url.Host) {
		log.Error("error parsing url: incorrect domain")
		format.DisplayInteractionError(session, intr, "Domain must be `youtube.com`, `youtu.be` and etc.")
		return
	}

	videoURL := url.String()

	log.Info("requesting video data", "url", videoURL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ytDataChan := make(chan youtube.SearchResult)
	if err := youtube.GetYoutubeData(ctx, videoURL, ytDataChan); err != nil {
		log.Error("error getting youtube data", "err", err)
		format.DisplayInteractionError(session, intr, "Error getting video data from youtube. See the log for details.")
		return
	}

	err = session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
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

	var player *playback.Player
	if ps := c.playerStorage.Get(intr.GuildID); ps != nil {
		log.Info("get stored player")
		player = ps
	} else {
		log.Info("creating new player")

		channelID, err := c.getUserChannelID(session, intr.GuildID, intr.Member.User.ID)
		if err != nil {
			log.Error("failure getting channel id", "err", err)
			format.DisplayInteractionError(session, intr, "You must be in a voice channel to use this command.")
			return
		}

		voice, err := session.ChannelVoiceJoin(intr.GuildID, channelID, false, true)
		if err != nil {
			if voice != nil {
				voice.Close()
			}
			log.Error("failure joining voice channel", "channelId", channelID, "err", err)
			format.DisplayInteractionError(session, intr, "Error joining voice channel.")
			return
		}

		var wg sync.WaitGroup
		wg.Add(1)
		player = c.setupPlayer(session, intr, voice, log, &wg)
		if player == nil {
			if voice != nil {
				voice.Close()
			}
			format.DisplayInteractionError(session, intr, "Error starting playback.")
			return
		}
		wg.Wait()
	}

	for ytData := range ytDataChan {
		if ytData.Error != nil {
			log.Error("failure getting url from GetYoutubeData", "err", ytData.Error)
			format.DisplayInteractionError(session, intr, "Error getting song data.")
			continue
		}
		video := ytData.Video

		if index < 1 {
			if err := player.Add(video); err != nil {
				log.Error("failed to add video to playback service", "err", err)
				return
			}
		} else {
			if err := player.Insert(video, index); err != nil {
				log.Error("failed to insert video to playback service", "err", err)
				return
			}
			index++
		}

		log.Info("added video to player", "video", video.Title)

		embed := embed.NewEmbed().
			SetAuthor("Added to queue").
			SetTitle(video.Title).
			SetUrl(video.GetShortURL()).
			SetThumbnail(video.Thumbnail).
			SetDescription(video.Length).
			SetFooter(fmt.Sprintf("Queue length: %d", player.Count()), "").
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

func (c *Command) setupPlayer(session *discordgo.Session, intr *discordgo.InteractionCreate, voice *discordgo.VoiceConnection, log *slog.Logger, wg *sync.WaitGroup) *playback.Player {
	player := playback.NewPlayer(voice)
	if err := c.playerStorage.Add(intr.GuildID, player); err != nil {
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
					if last, err := c.isBotLastInVoiceChannel(session, guildId, channelId); err != nil {
						log.Error("timeout ticker error", "err", err)
						return
					} else if last {
						playbackCancel(playback.ErrCauseTimeout)
						return
					}
				}
			}
		}(player.ChannelID())

		err := player.Run(playbackContext, wg)
		if err != nil {
			log.Error("playback error has occured", "err", err)
		}

		stopHandlerCancel()

		if err := player.Cleanup(); err != nil {
			log.Error("failure to close player", "err", err)
		}

		log.Info("deleting player", "guildId", guildId)
		if err := c.playerStorage.Delete(guildId); err != nil {
			log.Error("error deleting player", "guildId", guildId, "err", err)
		}
	}(intr.GuildID)

	return player
}

func createStopHandler(sesh *discordgo.Session, cancel context.CancelCauseFunc, guildID string) func() {
	return sesh.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.GuildID != guildID {
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

func (c *Command) handleSkip(sesh *discordgo.Session, intr *discordgo.InteractionCreate) {
	guildID := intr.GuildID
	userID := intr.Member.User.ID

	if err := c.isUserAndBotInSameChannel(sesh, guildID, userID); err != nil {
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

	opt := intr.ApplicationCommandData().Options

	skipAmount := int64(skipMinValue)
	if len(opt) > 0 {
		skipAmount = intr.ApplicationCommandData().Options[0].IntValue()
	}

	if ps := c.playerStorage.Get(guildID); ps != nil {
		err := ps.Skip(int(skipAmount))
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

func (c *Command) isUserAndBotInSameChannel(sesh *discordgo.Session, guildID string, userID string) error {
	botUserID := sesh.State.User.ID

	botChannelID, err := c.getUserChannelID(sesh, guildID, botUserID)
	if err != nil {
		return errBotIsNotInAnyChannel
	}

	channelID, err := c.getUserChannelID(sesh, guildID, userID)
	if err != nil {
		return errUserNotInAnyChannel
	}

	if channelID != botChannelID {
		return errUserNotInBotsChannel
	}

	return nil
}

func (c *Command) getUserChannelID(sesh *discordgo.Session, guildID string, userID string) (string, error) {
	var channelID string

	g, err := sesh.State.Guild(guildID)
	if err != nil {
		if !errors.Is(err, discordgo.ErrStateNotFound) {
			return channelID, fmt.Errorf("failure getting guild: %w", err)
		}

		g, err = sesh.Guild(guildID)
		if err != nil {
			return channelID, fmt.Errorf("failure getting guild: %w", err)
		}
	}

	c.logger.Info("guild acquired", "guildId", g.ID, "name", g.Name)

	for _, vs := range g.VoiceStates {
		if vs.UserID == userID {
			c.logger.Info("user found in channel", "usr", vs.UserID, "chn", vs.ChannelID)
			channelID = vs.ChannelID
			break
		}
	}
	if len(channelID) == 0 {
		return channelID, errors.New("user is not in any voice channels")
	}

	return channelID, nil
}

func (c *Command) isBotLastInVoiceChannel(sesh *discordgo.Session, guildID string, channelID string) (bool, error) {
	g, err := sesh.State.Guild(guildID)
	if err != nil {
		return false, fmt.Errorf("failure getting guild: %w", err)
	}

	for _, vs := range g.VoiceStates {
		if vs.UserID != sesh.State.User.ID && vs.ChannelID == channelID {
			return false, nil
		}
	}

	return true, nil
}
