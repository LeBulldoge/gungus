package discord

import (
	"log/slog"
	"strings"

	"github.com/LeBulldoge/gungus/internal/database"
	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	session *discordgo.Session
	storage *database.Storage
}

func NewBot(token string, storage *database.Storage) (*Bot, error) {
	s, err := discordgo.New("Bot " + token)
	return &Bot{session: s, storage: storage}, err
}

func (bot *Bot) addHandlers() {
	bot.session.AddHandler(func(_ *discordgo.Session, intr *discordgo.InteractionCreate) {
		switch intr.Type {
		case discordgo.InteractionApplicationCommand:
			fallthrough
		case discordgo.InteractionApplicationCommandAutocomplete:
			if h, ok := commandHandlers[intr.ApplicationCommandData().Name]; ok {
				h(bot, intr)
			}
		case discordgo.InteractionMessageComponent:
			customID := intr.MessageComponentData().CustomID
			if strings.HasPrefix(customID, "option") {
				handleVote(bot, intr)
			} else if strings.HasPrefix(customID, "movielist") {
				movieListPaginate(bot, intr)
			} else {
				slog.Error("unsupported component type", "id", intr.MessageComponentData().CustomID)
			}
		default:
			slog.Error("unsupported interaction", "intr", intr)
		}
	})
}

func (bot *Bot) OpenConnection() error {
	bot.addHandlers()

	return bot.session.Open()
}

func (bot *Bot) CreateCommands() error {
	for _, v := range commands {
		_, err := bot.session.ApplicationCommandCreate(bot.session.State.User.ID, "", v)
		if err != nil {
			slog.Error("error while creating command", "cmd", v.Name, "err", err)
			return err
		}

		slog.Info("created command", "cmd", v.Name)
	}

	return nil
}

func (bot *Bot) Shutdown() {
	err := bot.storage.Close()
	if err != nil {
		slog.Error("failure closing database connection", "err", err)
	}

	slog.Info("Removing commands...")
	registeredCommands, err := bot.session.ApplicationCommands(bot.session.State.User.ID, "")
	if err != nil {
		slog.Error("could not fetch registered commands", "err", err)
	}

	for _, v := range registeredCommands {
		err := bot.session.ApplicationCommandDelete(bot.session.State.User.ID, "", v.ID)
		if err != nil {
			slog.Error("cannot delete command", "cmd", v.Name, "err", err)
		}
	}

	slog.Info("gracefully shutting down.")
}
