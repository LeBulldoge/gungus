package discord

import (
	"log/slog"

	"github.com/LeBulldoge/gungus/internal/database"
	"github.com/LeBulldoge/gungus/internal/discord/bot"
	"github.com/LeBulldoge/gungus/internal/discord/commands"
)

func StartBot(token string, storage *database.Storage) (*bot.Bot, error) {
	bot, err := bot.NewBot(token, storage)
	if err != nil {
		slog.Error("error while creating session", "err", err)
		return bot, err
	}

	err = bot.OpenConnection()
	if err != nil {
		slog.Error("error while opening session", "err", err)
		return bot, err
	}

	err = commands.SetupCommands(bot)
	if err != nil {
		slog.Error("error while creating commands", "err", err)
		bot.Shutdown()
		return bot, err
	}

	return bot, err
}
