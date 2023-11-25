package poll

import (
	"fmt"
	"strings"

	"github.com/LeBulldoge/gungus/internal/discord/bot"
	"github.com/LeBulldoge/gungus/internal/discord/format"
	"github.com/LeBulldoge/gungus/internal/poll"
	"github.com/bwmarrin/discordgo"
)

func (c *PollCommand) handlePoll(bot *bot.Bot, intr *discordgo.InteractionCreate) {
	opt := intr.ApplicationCommandData().Options[0]

	var pollAnsText []string
	for _, v := range opt.Options[1:] {
		if v.StringValue() != "" {
			pollAnsText = append(pollAnsText, v.StringValue())
		}
	}

	p := poll.New(opt.Options[0].StringValue())

	pollButtons := []discordgo.MessageComponent{}
	for i := 0; i < len(pollAnsText); i++ {
		spl := strings.Split(pollAnsText[i], ";")
		if len(spl) < 2 {
			format.DisplayInteractionError(bot.Session, intr.Interaction, "Incorrect formatting for option %d. <emoji> ; <description>")
			return
		}

		emojiStr, labelStr := spl[0], spl[1]

		emoji := format.EmojiComponentFromString(emojiStr)
		customID := fmt.Sprintf("option_%d_%s", i, strings.Trim(emojiStr, " "))

		btn := discordgo.Button{
			CustomID: customID,
			Label:    labelStr,
			Emoji:    emoji,
			Style:    discordgo.SecondaryButton,
		}

		p.Options[btn.CustomID] = []string{}

		pollButtons = append(pollButtons, btn)
	}

	err := bot.Session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: poll.PlotBarChart(p.Title, p.CountVotes()),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: pollButtons,
				},
			},
		},
	})
	if err != nil {
		c.logger.Error("error responding to interaction", intr.ID, err)
		return
	}

	msg, err := bot.Session.InteractionResponse(intr.Interaction)
	if err != nil {
		c.logger.Error("error collecting response for interaction", intr.ID, err)
		return
	}

	p.ID = msg.ID
	err = bot.Storage.AddPoll(p)
	if err != nil {
		c.logger.Error("failed storing poll", "err", err)
	}
}

func (c *PollCommand) handleVote(bot *bot.Bot, intr *discordgo.InteractionCreate) {
	err := bot.Session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		c.logger.Error("error creating deferred response", "err", err)
		return
	}

	voteCustomID := intr.MessageComponentData().CustomID
	err = bot.Storage.CastVote(intr.Message.ID, voteCustomID, intr.Member.User.ID)
	if err != nil {
		c.logger.Error("error casting vote", "id", intr.Message.ID, "err", err)
		return
	}

	p, err := bot.Storage.GetPoll(intr.Message.ID)
	if err != nil {
		c.logger.Error("error getting poll", "id", intr.Message.ID, "err", err)
	}

	chartStr := poll.PlotBarChart(p.Title, p.CountVotes())
	msg := discordgo.NewMessageEdit(intr.ChannelID, intr.Message.ID)
	msg.Content = &chartStr

	_, err = bot.Session.ChannelMessageEditComplex(msg)
	if err != nil {
		c.logger.Error("error editing message", "id", intr.Message.ID, "err", err)
	}
}
