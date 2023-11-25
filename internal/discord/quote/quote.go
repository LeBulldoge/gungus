package quote

import (
	"context"
	"fmt"
	"time"

	"math/rand"

	"github.com/LeBulldoge/gungus/internal/discord/format"
	"github.com/LeBulldoge/gungus/internal/quote"
	"github.com/bwmarrin/discordgo"
)

func (c *QuoteCommand) addQuote(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	opt := intr.ApplicationCommandData().Options[0]

	byUser := opt.Options[0].UserValue(session)
	quoteText := opt.Options[1].StringValue()

	err := quote.AddQuote(context.TODO(), c.GetStorage(), byUser.ID, quoteText, time.Now())
	if err != nil {
		format.DisplayInteractionError(session, intr.Interaction, fmt.Sprintf("failed saving a quote: %s", err))
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
		c.logger.Error("error responding to interaction", intr.ID, err)
		return
	}
}

func (c *QuoteCommand) randomQuote(session *discordgo.Session, intr *discordgo.InteractionCreate) {
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

	if err != nil {
		format.DisplayInteractionError(session, intr.Interaction, fmt.Sprintf("failed getting quotes: %s", err))
		return
	}

	if len(quotes) == 0 {
		format.DisplayInteractionError(session, intr.Interaction, "no quotes found")
		return
	}

	ind := 0
	if len(quotes) > 1 {
		ind = rand.Intn(len(quotes))
	}

	selectedQuote := quotes[ind]
	if byUser == nil {
		byUser, err = session.User(selectedQuote.User)
		if err != nil {
			format.DisplayInteractionError(session, intr.Interaction, fmt.Sprintf("failure aquiring user data for user id: %s", selectedQuote.User))
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
		c.logger.Error("error responding to interaction", intr.ID, err)
		return
	}
}
