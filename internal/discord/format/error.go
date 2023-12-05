package format

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func CheckDiscordErrCode(err error, code int) bool {
	var restErr *discordgo.RESTError
	return errors.As(err, &restErr) && restErr.Message != nil && restErr.Message.Code == code
}

func DisplayInteractionWithError(s *discordgo.Session, intr *discordgo.InteractionCreate, content string, cause error) {
	errStr := cause.Error()

	var sb strings.Builder
	sb.Grow(len(content) + len(errStr) + 6)
	sb.WriteString(content)
	sb.WriteRune('\n')
	sb.WriteRune('\n')
	sb.WriteRune('`')
	sb.WriteString(errStr)
	sb.WriteRune('`')
	content = sb.String()

	DisplayInteractionError(s, intr, content)
}

func DisplayInteractionError(s *discordgo.Session, intr *discordgo.InteractionCreate, content string) {
	err := s.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		if CheckDiscordErrCode(err, discordgo.ErrCodeInteractionHasAlreadyBeenAcknowledged) {
			_, err = s.FollowupMessageCreate(intr.Interaction, false, &discordgo.WebhookParams{
				Content: content,
				Flags:   discordgo.MessageFlagsEphemeral,
			})
		}
		if err != nil {
			slog.Error("failed displaying error", "err", err)
		}
	}
}
