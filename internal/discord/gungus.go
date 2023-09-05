package discord

import (
	"flag"
	"log/slog"
	"strings"

	"github.com/LeBulldoge/gungus/internal/poll"
	"github.com/bwmarrin/discordgo"
)

var polls = make(map[string]poll.Poll)

type Bot *discordgo.Session

func NewBot(token string) (Bot, error) {
	return discordgo.New("Bot " + token)
}

func OpenConnection(bot *discordgo.Session) error {
	flag.Parse()

	bot.AddHandler(func(s *discordgo.Session, intr *discordgo.InteractionCreate) {
		switch intr.Type {
		case discordgo.InteractionApplicationCommand:
			if h, ok := commandHandlers[intr.ApplicationCommandData().Name]; ok {
				h(s, intr)
			}
		case discordgo.InteractionMessageComponent:
			err := s.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredMessageUpdate,
			})
			if err != nil {
				slog.Error("error creating deferred response: %v", err)
				return
			}

			p, ok := polls[intr.Message.ID]
			if ok {
				voteCustomID := intr.MessageComponentData().CustomID
				err = p.CastVote(intr.Member.User.ID, voteCustomID)
				if err != nil {
					slog.Error("error casting vote for poll %s: %v", intr.Message.ID, err)
					return
				}
			} else {
				slog.Error("error getting poll for message ID %s", intr.Message.ID)
				return
			}

			chartStr := poll.PlotBarChart(p.Title, p.GetVotes())
			msg := discordgo.NewMessageEdit(intr.ChannelID, intr.Message.ID)
			msg.Content = &chartStr

			_, err = s.ChannelMessageEditComplex(msg)
			if err != nil {
				slog.Error("error editing message %s: %v", intr.Message.ID, err)
			}
		}
	})

	return bot.Open()
}

func CreateCommands(bot *discordgo.Session) error {
	for _, v := range commands {
		_, err := bot.ApplicationCommandCreate(bot.State.User.ID, "", v)
		if err != nil {
			slog.Error("error while creating command", "cmd", v.Name, "err", err)
			return err
		}

		slog.Info("created command", "cmd", v.Name)
	}

	return nil
}

func Shutdown(bot *discordgo.Session) {
	slog.Info("Removing commands...")
	registeredCommands, err := bot.ApplicationCommands(bot.State.User.ID, "")
	if err != nil {
		slog.Error("could not fetch registered commands", "err", err)
	}

	for _, v := range registeredCommands {
		err := bot.ApplicationCommandDelete(bot.State.User.ID, "", v.ID)
		if err != nil {
			slog.Error("cannot delete command", "cmd", v.Name, "err", err)
		}
	}

	slog.Info("Gracefully shutting down.")
}

func isCustomEmoji(s string) bool {
	return s[0] == '<'
}

func emojiComponentFromString(s string) discordgo.ComponentEmoji {
	emoji := discordgo.ComponentEmoji{}
	if isCustomEmoji(s) {
		s = s[1 : len(s)-2]
		parts := strings.Split(s, ":")

		emoji.Animated = parts[0] == "a"
		emoji.Name = parts[1]
		emoji.ID = parts[2]
	} else {
		emoji.Name = strings.Trim(s, " ")
	}

	return emoji
}
