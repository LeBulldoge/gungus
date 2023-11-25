package poll

import (
	"fmt"
	"strings"

	"log/slog"

	"github.com/LeBulldoge/gungus/internal/discord/bot"
	"github.com/bwmarrin/discordgo"
)

type PollCommand struct {
	logger *slog.Logger
}

func NewCommand() *PollCommand {
	return &PollCommand{}
}

func (c *PollCommand) GetSignature() []*discordgo.ApplicationCommand {
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

func (c *PollCommand) Setup(bot *bot.Bot) error {
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
				c.handlePoll(bot, intr)
			}
		case discordgo.InteractionMessageComponent:
			customID := intr.MessageComponentData().CustomID
			if strings.HasPrefix(customID, "option") {
				c.handleVote(bot, intr)
			}
		}
	})

	return nil
}

func (c *PollCommand) Cleanup(bot *bot.Bot) error {
	return nil
}

func (c *PollCommand) AddLogger(logger *slog.Logger) {
	if logger == nil {
		return
	}

	c.logger = logger
}
