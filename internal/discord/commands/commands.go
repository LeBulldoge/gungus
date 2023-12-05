package commands

import (
	"fmt"
	"log/slog"

	"github.com/LeBulldoge/gungus/internal/database"
	"github.com/LeBulldoge/gungus/internal/discord/bot"
	"github.com/LeBulldoge/gungus/internal/discord/commands/movie"
	"github.com/LeBulldoge/gungus/internal/discord/commands/play"
	"github.com/LeBulldoge/gungus/internal/discord/commands/poll"
	"github.com/LeBulldoge/gungus/internal/discord/commands/quote"
	"github.com/bwmarrin/discordgo"
)

type Command interface {
	Setup(*bot.Bot) error
	Cleanup(*bot.Bot) error
	SetStorageConnection(*database.Storage)

	GetSignature() []*discordgo.ApplicationCommand

	AddLogger(*slog.Logger)
}

var commands = map[string]Command{
	"play":  play.NewCommand(),
	"movie": movie.NewCommand(),
	"poll":  poll.NewCommand(),
	"quote": quote.NewCommand(),
}

func SetupCommands(bot *bot.Bot) error {
	botUserId := bot.Session.State.User.ID
	for name, cmd := range commands {
		sigs := cmd.GetSignature()
		for _, sig := range sigs {
			regCmd, err := bot.Session.ApplicationCommandCreate(
				botUserId, "", sig,
			)
			if err != nil {
				return fmt.Errorf("failed to register %s: %w", sig.Name, err)
			}
			slog.Info("command registered", "command.name", regCmd.Name)
		}
		logger := slog.Default().With(
			slog.Group(
				"command",
				slog.String("name", name),
			),
		)
		cmd.AddLogger(logger)
		cmd.SetStorageConnection(bot.Storage)
		if err := cmd.Setup(bot); err != nil {
			return fmt.Errorf("failed to setup command %s: %w", name, err)
		}
	}
	return nil
}
