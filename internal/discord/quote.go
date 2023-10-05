package discord

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/LeBulldoge/gungus/internal/quote"
	"github.com/bwmarrin/discordgo"
)

func handleQuote(bot *Bot, intr *discordgo.InteractionCreate) {
	opt := intr.ApplicationCommandData().Options[0]

	switch opt.Name {
	case "add":
		addQuote(bot, intr)
	case "random":
		randomQuote(bot, intr)
	}
}

func addQuote(bot *Bot, intr *discordgo.InteractionCreate) {
	opt := intr.ApplicationCommandData().Options[0]

	byUser := opt.Options[0].UserValue(bot.session)
	quoteText := opt.Options[1].StringValue()

	err := quote.AddQuote(context.TODO(), bot.storage, byUser.ID, quoteText, time.Now())
	if err != nil {
		displayInteractionError(bot.session, intr.Interaction, fmt.Sprintf("failed saving a quote: %s", err))
		return
	}

	err = bot.session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Quote by user %s saved.", byUser.Mention()),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		slog.Error("error responding to interaction", intr.ID, err)
		return
	}
}

func randomQuote(bot *Bot, intr *discordgo.InteractionCreate) {
	opt := intr.ApplicationCommandData().Options[0]

	var byUser *discordgo.User
	var quotes []quote.Quote
	var err error
	if len(opt.Options) > 0 {
		byUser = opt.Options[0].UserValue(bot.session)
		quotes, err = quote.GetQuotesByUser(context.TODO(), bot.storage, byUser.ID)
	} else {
		quotes, err = quote.GetQuotes(context.TODO(), bot.storage)
	}

	if err != nil {
		displayInteractionError(bot.session, intr.Interaction, fmt.Sprintf("failed getting quotes: %s", err))
		return
	}

	if len(quotes) == 0 {
		displayInteractionError(bot.session, intr.Interaction, fmt.Sprintf("no quotes found for %s", byUser.Mention()))
		return
	}

	ind := 0
	if len(quotes) > 1 {
		ind = rand.Intn(len(quotes)) - 1
	}

	q := quotes[ind]
	if byUser == nil {
		byUser, err = bot.session.User(q.User)
		if err != nil {
			displayInteractionError(bot.session, intr.Interaction, fmt.Sprintf("failure aquiring user data for user id: %s", q.User))
			return
		}
	}

	err = bot.session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Here is a random quote!\n\n%s: %s\n> %s\n\n", byUser.Mention(), quotes[ind].Date.UTC().Format(time.RFC822), quotes[ind].Text),
		},
	})
	if err != nil {
		slog.Error("error responding to interaction", intr.ID, err)
		return
	}
}
