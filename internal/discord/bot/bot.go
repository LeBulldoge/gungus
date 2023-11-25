package bot

import (
	"log/slog"

	"github.com/LeBulldoge/gungus/internal/database"
	"github.com/LeBulldoge/gungus/internal/discord/play/playback"
	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	Session         *discordgo.Session
	Storage         *database.Storage
	PlaybackManager playback.PlaybackServiceManager
}

func NewBot(token string, storage *database.Storage) (*Bot, error) {
	s, err := discordgo.New("Bot " + token)
	return &Bot{Session: s, Storage: storage, PlaybackManager: playback.NewManager()}, err
}

func (bot *Bot) OpenConnection() error {
	return bot.Session.Open()
}

func (bot *Bot) CreateCommands(commands []*discordgo.ApplicationCommand) error {
	for _, v := range commands {
		_, err := bot.Session.ApplicationCommandCreate(bot.Session.State.User.ID, "", v)
		if err != nil {
			slog.Error("error while creating command", "cmd", v.Name, "err", err)
			return err
		}

		slog.Info("created command", "cmd", v.Name)
	}

	return nil
}

func (bot *Bot) Shutdown() {
	err := bot.Storage.Close()
	if err != nil {
		slog.Error("failure closing database connection", "err", err)
	}

	slog.Info("Removing commands...")
	registeredCommands, err := bot.Session.ApplicationCommands(bot.Session.State.User.ID, "")
	if err != nil {
		slog.Error("could not fetch registered commands", "err", err)
	}

	for _, v := range registeredCommands {
		err := bot.Session.ApplicationCommandDelete(bot.Session.State.User.ID, "", v.ID)
		if err != nil {
			slog.Error("cannot delete command", "cmd", v.Name, "err", err)
		}
	}

	slog.Info("gracefully shutting down.")
}
