package quote

import (
	"log/slog"

	"github.com/LeBulldoge/gungus/internal/database"
	"github.com/LeBulldoge/gungus/internal/discord/bot"
	"github.com/bwmarrin/discordgo"
)

type QuoteCommand struct {
	database.WithStorage

	logger *slog.Logger
}

func NewCommand() *QuoteCommand {
	return &QuoteCommand{}
}

func (c *QuoteCommand) GetSignature() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "quote",
			Description: "Interact with polls",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Description: "Save a quote",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "by_user",
							Description: "User attribution",
							Type:        discordgo.ApplicationCommandOptionUser,
							Required:    true,
						},
						{
							Name:        "text",
							Description: "Quote text",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
				{
					Name:        "random",
					Description: "Get a random quote by a particular user",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "by_user",
							Description: "User to get a quote from",
							Type:        discordgo.ApplicationCommandOptionUser,
						},
					},
				},
			},
		},
	}
}

func (c *QuoteCommand) AddLogger(logger *slog.Logger) {
	if logger == nil {
		return
	}

	c.logger = logger
}

func (c *QuoteCommand) Setup(bot *bot.Bot) error {
	bot.Session.AddHandler(func(sesh *discordgo.Session, intr *discordgo.InteractionCreate) {
		switch intr.Type {
		case discordgo.InteractionApplicationCommand:
			data := intr.ApplicationCommandData()
			if data.Name != "quote" {
				return
			}
			subData := intr.ApplicationCommandData().Options[0]
			switch subData.Name {
			case "add":
				c.addQuote(sesh, intr)
			case "random":
				c.randomQuote(sesh, intr)
			}
		}
	})

	return nil
}

func (c *QuoteCommand) Cleanup(bot *bot.Bot) error {
	return nil
}
