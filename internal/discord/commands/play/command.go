package play

import (
	"context"
	"log/slog"

	"github.com/LeBulldoge/gungus/internal/database"
	"github.com/LeBulldoge/gungus/internal/discord/bot"
	"github.com/LeBulldoge/gungus/internal/discord/commands/play/playback"
	"github.com/bwmarrin/discordgo"
)

type Command struct {
	playerStorage playback.PlayerStorage

	autocompleteCancelMap map[string]context.CancelFunc

	logger *slog.Logger
}

func NewCommand() *Command {
	return &Command{
		playerStorage:         playback.NewManager(),
		autocompleteCancelMap: map[string]context.CancelFunc{},
	}
}

var (
	skipMinValue        = 1.0
	queueAmountMinValue = 1.0
)

func (c *Command) GetSignature() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "play",
			Description: "Play a youtube video",
			Type:        discordgo.ChatApplicationCommand,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:         "search",
					Description:  "Youtube link or search query",
					Type:         discordgo.ApplicationCommandOptionString,
					Required:     true,
					Autocomplete: true,
				},
			},
		},
		{
			Name:        "stop",
			Description: "Stop audio playback",
			Type:        discordgo.ChatApplicationCommand,
		},
		{
			Name:        "skip",
			Description: "Skip current song",
			Type:        discordgo.ChatApplicationCommand,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "amount",
					Description: "Amount of songs to skip",
					Type:        discordgo.ApplicationCommandOptionInteger,
					MinValue:    &skipMinValue,
				},
			},
		},
		{
			Name:        "queue",
			Description: "View the current song queue",
			Type:        discordgo.ChatApplicationCommand,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "amount",
					Description: "Amount of fields to show. Each field can contain up to 10 songs.",
					Type:        discordgo.ApplicationCommandOptionInteger,
					MinValue:    &queueAmountMinValue,
				},
			},
		},
	}
}

func (c *Command) Setup(bot *bot.Bot) error {
	bot.Session.AddHandler(func(sesh *discordgo.Session, intr *discordgo.InteractionCreate) {
		if intr.Type != discordgo.InteractionApplicationCommand && intr.Type != discordgo.InteractionApplicationCommandAutocomplete {
			return
		}
		opt := intr.ApplicationCommandData()
		switch opt.Name {
		case "play":
			c.handlePlay(sesh, intr)
		case "skip":
			c.handleSkip(sesh, intr)
		case "queue":
			c.HandleQueue(sesh, intr)
		}
	})

	return nil
}

func (c *Command) Cleanup(bot *bot.Bot) error {
	return nil
}

func (c *Command) AddLogger(logger *slog.Logger) {
	if logger == nil {
		return
	}

	c.logger = logger
}

func (c *Command) SetStorageConnection(*database.Storage) {}
