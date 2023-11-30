package play

import (
	"log/slog"

	"github.com/LeBulldoge/gungus/internal/database"
	"github.com/LeBulldoge/gungus/internal/discord/bot"
	"github.com/LeBulldoge/gungus/internal/discord/play/playback"
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

var (
	skipMinValue = 1.0
)

func (c *PlayCommand) GetSignature() []*discordgo.ApplicationCommand {
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
	}
}

func (c *PlayCommand) Setup(bot *bot.Bot) error {
	bot.Session.AddHandler(func(sesh *discordgo.Session, intr *discordgo.InteractionCreate) {
		if intr.Type != discordgo.InteractionApplicationCommand && intr.Type != discordgo.InteractionApplicationCommandAutocomplete {
			return
		}
		opt := intr.ApplicationCommandData()
		switch opt.Name {
		case "play":
			c.HandlePlay(sesh, intr)
		case "skip":
			c.HandleSkip(sesh, intr)
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

func (c *PlayCommand) SetStorageConnection(*database.Storage) {}
