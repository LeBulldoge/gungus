package quote

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/LeBulldoge/gungus/internal/discord/format"
	"github.com/LeBulldoge/gungus/internal/quote"
	"github.com/bwmarrin/discordgo"
)

func (c *Command) addQuote(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	opt := intr.ApplicationCommandData().Options[0]

	byUser := opt.Options[0].UserValue(session)
	quoteText := opt.Options[1].StringValue()

	log := c.logger.With(
		slog.Group(
			"add",
			"byUser", byUser,
			"text", quoteText,
		),
	)

	err := quote.AddQuote(context.TODO(), c.GetStorage(), byUser.ID, quoteText, time.Now())
	if err != nil {
		log.Error("failed saving a quote", "err", err)
		format.DisplayInteractionError(session, intr, "Error saving a quote.")
		return
	}

	err = session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Quote by user %s saved.", byUser.Mention()),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Error("error responding to interaction", intr.ID, err)
		return
	}

	log.Info("quote added")
}

func (c *Command) randomQuote(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	opt := intr.ApplicationCommandData().Options[0]

	var byUser *discordgo.User
	var quotes []quote.Quote
	var err error
	if len(opt.Options) > 0 {
		byUser = opt.Options[0].UserValue(session)
		quotes, err = quote.GetQuotesByUser(context.TODO(), c.GetStorage(), byUser.ID)
	} else {
		quotes, err = quote.GetQuotes(context.TODO(), c.GetStorage())
	}

	log := c.logger.With(
		slog.Group(
			"random",
			"byUser", byUser,
		),
	)

	if err != nil {
		log.Error("failure getting quotes", "err", err)
		format.DisplayInteractionError(session, intr, "Error getting quotes.")
		return
	}

	if len(quotes) == 0 {
		log.Error("no quotes found", "err", err)
		format.DisplayInteractionError(session, intr, "No quotes found.")
		return
	}

	ind := 0
	if len(quotes) > 1 {
		ind = rand.Intn(len(quotes))
	}

	selectedQuote := quotes[ind]
	log = log.With(
		slog.Group(
			"random",
			"selectedQuote", selectedQuote,
		),
	)
	if byUser == nil {
		byUser, err = session.User(selectedQuote.User)
		if err != nil {
			log.Error("failure getting user data", "err", err)
			format.DisplayInteractionError(session, intr, "Error getting user data.")
			return
		}
	}
	mention := byUser.Mention()
	dateStamp := format.TimeToTimestamp(selectedQuote.Date.UTC())

	const MessageFlagsSilent = 1 << 12
	err = session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Here is a random quote!\n\n%s: %s\n> %s\n\n", mention, dateStamp, selectedQuote.Text),
			Flags:   MessageFlagsSilent,
		},
	})
	if err != nil {
		log.Error("error responding to interaction", "err", err)
		return
	}

	log.Info("successfully quoted")
}
