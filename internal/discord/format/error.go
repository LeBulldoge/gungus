package format

import (
	"errors"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/exp/slog"
)

func CheckDiscordErrCode(err error, code int) bool {
	var restErr *discordgo.RESTError
	return errors.As(err, &restErr) && restErr.Message != nil && restErr.Message.Code == code
}

func DisplayInteractionError(s *discordgo.Session, intr *discordgo.Interaction, content string) {
	slog.Error(content)
	err := s.InteractionRespond(intr, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		if CheckDiscordErrCode(err, discordgo.ErrCodeInteractionHasAlreadyBeenAcknowledged) {
			_, err = s.FollowupMessageCreate(intr, false, &discordgo.WebhookParams{
				Content: content,
				Flags:   discordgo.MessageFlagsEphemeral,
			})
		}
		if err != nil {
			slog.Error("failed displaying error", "err", err)
		}
	}
}
