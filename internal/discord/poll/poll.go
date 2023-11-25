package poll

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/LeBulldoge/gungus/internal/discord/format"
	"github.com/LeBulldoge/gungus/internal/poll"
	"github.com/bwmarrin/discordgo"
)

func (c *PollCommand) handlePoll(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	opt := intr.ApplicationCommandData().Options[0]
	pollTitle := opt.Options[0].StringValue()

	var pollAnsText []string
	for _, v := range opt.Options[1:] {
		if v.StringValue() != "" {
			pollAnsText = append(pollAnsText, v.StringValue())
		}
	}

	logger := c.logger.With(
		slog.Group(
			"poll",
			"title", pollTitle,
			"answers", pollAnsText,
		),
	)

	p := poll.New(pollTitle)

	pollButtons := []discordgo.MessageComponent{}
	for i := 0; i < len(pollAnsText); i++ {
		spl := strings.Split(pollAnsText[i], ";")
		if len(spl) < 2 {
			logger.Error("incorrect formatting for poll option", "option", i)
			format.DisplayInteractionError(session, intr.Interaction, fmt.Sprintf("Incorrect formatting for option %d. <emoji> ; <description>", i))
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

	err := session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
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
		logger.Error("error responding to interaction", intr.ID, err)
		return
	}

	msg, err := session.InteractionResponse(intr.Interaction)
	if err != nil {
		logger.Error("error collecting response for interaction", intr.ID, err)
		format.DisplayInteractionError(session, intr.Interaction, "Error saving poll in storage.")
		return
	}

	p.ID = msg.ID
	err = c.GetStorage().AddPoll(p)
	if err != nil {
		logger.Error("failed storing poll", "err", err)
		format.DisplayInteractionError(session, intr.Interaction, "Error saving poll in storage.")
	}
}

func (c *PollCommand) handleVote(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	err := session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		c.logger.Error("error creating deferred response", "err", err)
		return
	}

	voteCustomID := intr.MessageComponentData().CustomID
	logger := c.logger.With(
		slog.Group(
			"vote",
			"customId", voteCustomID,
			"pollId", intr.Message.ID,
		),
	)

	err = c.GetStorage().CastVote(intr.Message.ID, voteCustomID, intr.Member.User.ID)
	if err != nil {
		logger.Error("error casting vote", "err", err)
		format.DisplayInteractionError(session, intr.Interaction, "Error casting vote.")
		return
	}

	p, err := c.GetStorage().GetPoll(intr.Message.ID)
	if err != nil {
		logger.Error("error getting poll", "err", err)
		format.DisplayInteractionError(session, intr.Interaction, "Error getting poll from storage.")
		return
	}

	chartStr := poll.PlotBarChart(p.Title, p.CountVotes())
	msg := discordgo.NewMessageEdit(intr.ChannelID, intr.Message.ID)
	msg.Content = &chartStr

	_, err = session.ChannelMessageEditComplex(msg)
	if err != nil {
		logger.Error("error editing message", "err", err)
		format.DisplayInteractionError(session, intr.Interaction, "Error editing message.")
	}
}
