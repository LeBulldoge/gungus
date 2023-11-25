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

//func (bot *Bot) AddHandlers(commandHandlers map[string]func(*Bot, *discordgo.InteractionCreate)) {
//	bot.Session.AddHandler(func(_ *discordgo.Session, intr *discordgo.InteractionCreate) {
//		switch intr.Type {
//		case discordgo.InteractionApplicationCommand:
//			fallthrough
//		case discordgo.InteractionApplicationCommandAutocomplete:
//			if h, ok := commandHandlers[intr.ApplicationCommandData().Name]; ok {
//				h(bot, intr)
//			}
//		case discordgo.InteractionMessageComponent:
//			customID := intr.MessageComponentData().CustomID
//			if strings.HasPrefix(customID, "option") {
//				handleVote(bot, intr)
//			} else if strings.HasPrefix(customID, "movielist") {
//				movieListPaginate(bot, intr)
//			} else {
//				slog.Error("unsupported component type", "id", intr.MessageComponentData().CustomID)
//			}
//		default:
//			slog.Error("unsupported interaction", "intr", intr)
//		}
//	})
//}

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
