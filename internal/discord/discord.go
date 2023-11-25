package discord

import (
	"log/slog"

	"github.com/LeBulldoge/gungus/internal/database"
	"github.com/LeBulldoge/gungus/internal/discord/bot"
)

func StartBot(token string, storage *database.Storage) (*bot.Bot, error) {
	bot, err := bot.NewBot(token, storage)
	if err != nil {
		slog.Error("error while creating session: %v", err)
		return bot, err
	}

	err = bot.OpenConnection()
	if err != nil {
		slog.Error("error while opening session: %v", err)
		return bot, err
	}

	err = setupCommands(bot)
	if err != nil {
		slog.Error("error while creating commands: %v", err)
		bot.Shutdown()
		return bot, err
	}

	return bot, err
}
