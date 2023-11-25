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

func (c *PlayCommand) SetStorageConnection(*database.Storage) {}
