package poll

import (
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/LeBulldoge/gungus/internal/discord/embed"
	"github.com/LeBulldoge/gungus/internal/discord/format"
	"github.com/LeBulldoge/gungus/internal/poll"
	"github.com/bwmarrin/discordgo"
)

func (c *Command) handlePoll(session *discordgo.Session, intr *discordgo.InteractionCreate) {
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
			format.DisplayInteractionError(session, intr, fmt.Sprintf("Incorrect formatting for option %d. <emoji> ; <description>", i))
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

	pollEmbed := buildEmbedFromPoll(p)

	err := session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{pollEmbed},
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
		format.DisplayInteractionError(session, intr, "Error saving poll in storage.")
		return
	}

	p.ID = msg.ID
	err = c.GetStorage().AddPoll(p)
	if err != nil {
		logger.Error("failed storing poll", "err", err)
		format.DisplayInteractionError(session, intr, "Error saving poll in storage.")
	}
}

func (c *Command) handleVote(session *discordgo.Session, intr *discordgo.InteractionCreate) {
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
		format.DisplayInteractionError(session, intr, "Error casting vote.")
		return
	}

	p, err := c.GetStorage().GetPoll(intr.Message.ID)
	if err != nil {
		logger.Error("error getting poll", "err", err)
		format.DisplayInteractionError(session, intr, "Error getting poll from storage.")
		return
	}

	pollEmbed := buildEmbedFromPoll(p)
	_, err = session.ChannelMessageEditEmbed(intr.ChannelID, intr.Message.ID, pollEmbed)
	if err != nil {
		logger.Error("error editing message", "err", err)
		format.DisplayInteractionError(session, intr, "Error editing message.")
	}
}

const (
	empty = "⬛"
	full  = "🔲"
)

func buildEmbedFromPoll(p poll.Poll) *discordgo.MessageEmbed {
	e := embed.NewEmbed().
		SetTitle(p.Title)

	votes := p.CountVotes()

	total := 0
	options := make([]string, 0, len(votes))
	for option, count := range votes {
		total += count
		options = append(options, option)
	}

	sort.Strings(options)

	var sb strings.Builder
	for _, option := range options {
		count := votes[option]

		res := (float64(count) / float64(total)) * 10
		for i := 0; i < 10; i++ {
			if i < int(res) {
				sb.WriteString(full)
			} else {
				sb.WriteString(empty)
			}
		}

		sb.WriteRune(' ')
		sb.WriteString(strconv.Itoa(count))
		sb.WriteRune('/')
		sb.WriteString(strconv.Itoa(total))

		emoji := strings.Split(option, "_")[2]
		e.AddField(emoji, sb.String())

		sb.Reset()
	}

	sb.WriteString("Total votes: ")
	sb.WriteString(strconv.Itoa(total))

	e.SetFooter(sb.String(), "")
	e.SetTimestamp(time.Now().Format(time.RFC3339))

	return e.MessageEmbed
}
