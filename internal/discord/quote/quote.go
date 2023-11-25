package quote

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/LeBulldoge/gungus/internal/discord/bot"
	"github.com/LeBulldoge/gungus/internal/discord/format"
	"github.com/LeBulldoge/gungus/internal/quote"
	"github.com/bwmarrin/discordgo"
)

type QuoteCommand struct {
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
				c.addQuote(bot, intr)
			case "random":
				c.randomQuote(bot, intr)
			}
		}
	})

	return nil
}

func (c *QuoteCommand) Cleanup(bot *bot.Bot) error {
	return nil
}

func (c *QuoteCommand) addQuote(bot *bot.Bot, intr *discordgo.InteractionCreate) {
	opt := intr.ApplicationCommandData().Options[0]

	byUser := opt.Options[0].UserValue(bot.Session)
	quoteText := opt.Options[1].StringValue()

	err := quote.AddQuote(context.TODO(), bot.Storage, byUser.ID, quoteText, time.Now())
	if err != nil {
		format.DisplayInteractionError(bot.Session, intr.Interaction, fmt.Sprintf("failed saving a quote: %s", err))
		return
	}

	err = bot.Session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Quote by user %s saved.", byUser.Mention()),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		c.logger.Error("error responding to interaction", intr.ID, err)
		return
	}
}

func (c *QuoteCommand) randomQuote(bot *bot.Bot, intr *discordgo.InteractionCreate) {
	opt := intr.ApplicationCommandData().Options[0]

	var byUser *discordgo.User
	var quotes []quote.Quote
	var err error
	if len(opt.Options) > 0 {
		byUser = opt.Options[0].UserValue(bot.Session)
		quotes, err = quote.GetQuotesByUser(context.TODO(), bot.Storage, byUser.ID)
	} else {
		quotes, err = quote.GetQuotes(context.TODO(), bot.Storage)
	}

	if err != nil {
		format.DisplayInteractionError(bot.Session, intr.Interaction, fmt.Sprintf("failed getting quotes: %s", err))
		return
	}

	if len(quotes) == 0 {
		format.DisplayInteractionError(bot.Session, intr.Interaction, "no quotes found")
		return
	}

	ind := 0
	if len(quotes) > 1 {
		ind = rand.Intn(len(quotes))
	}

	selectedQuote := quotes[ind]
	if byUser == nil {
		byUser, err = bot.Session.User(selectedQuote.User)
		if err != nil {
			format.DisplayInteractionError(bot.Session, intr.Interaction, fmt.Sprintf("failure aquiring user data for user id: %s", selectedQuote.User))
			return
		}
	}
	mention := byUser.Mention()
	dateStamp := format.TimeToTimestamp(selectedQuote.Date.UTC())

	const MessageFlagsSilent = 1 << 12
	err = bot.Session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Here is a random quote!\n\n%s: %s\n> %s\n\n", mention, dateStamp, selectedQuote.Text),
			Flags:   MessageFlagsSilent,
		},
	})
	if err != nil {
		c.logger.Error("error responding to interaction", intr.ID, err)
		return
	}
}
