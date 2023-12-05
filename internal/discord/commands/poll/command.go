package poll

import (
	"fmt"
	"strings"

	"log/slog"

	"github.com/LeBulldoge/gungus/internal/database"
	"github.com/LeBulldoge/gungus/internal/discord/bot"
	"github.com/bwmarrin/discordgo"
)

type Command struct {
	database.WithStorage

	logger *slog.Logger
}

func NewCommand() *Command {
	return &Command{}
}

func (c *Command) GetSignature() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "poll",
			Description: "Interact with polls",
			Type:        discordgo.ChatApplicationCommand,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "start",
					Description: "Start a poll",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options:     buildPollCreateArgs(),
				},
			},
		},
	}
}

func buildPollCreateArgs() []*discordgo.ApplicationCommandOption {
	res := []*discordgo.ApplicationCommandOption{
		{
			Name:        "title",
			Description: "Title of the poll",
			Type:        discordgo.ApplicationCommandOptionString,
			Required:    true,
		},
	}
	for i := 0; i < 6; i++ {
		res = append(res, &discordgo.ApplicationCommandOption{
			Name:        fmt.Sprintf("option_%d", i),
			Description: fmt.Sprintf("Option number %d. Format: <emoji>;<description>", i),
			Type:        discordgo.ApplicationCommandOptionString,
			Required:    i < 2,
		})
	}

	return res
}

func (c *Command) Setup(bot *bot.Bot) error {
	bot.Session.AddHandler(func(sesh *discordgo.Session, intr *discordgo.InteractionCreate) {
		switch intr.Type {
		case discordgo.InteractionApplicationCommand:
			data := intr.ApplicationCommandData()
			if data.Name != "poll" {
				return
			}
			subData := intr.ApplicationCommandData().Options[0]
			switch subData.Name {
			case "start":
				c.handlePoll(sesh, intr)
			}
		case discordgo.InteractionMessageComponent:
			customID := intr.MessageComponentData().CustomID
			if strings.HasPrefix(customID, "option") {
				c.handleVote(sesh, intr)
			}
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
