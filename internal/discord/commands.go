package discord

import (
	"fmt"
	"log"
	"log/slog"
	"strings"

	"github.com/LeBulldoge/gungus/internal/poll"
	"github.com/bwmarrin/discordgo"
)

func buildPollCreateArgs() []*discordgo.ApplicationCommandOption {
	res := []*discordgo.ApplicationCommandOption{
		{
			Name:        "title",
			Description: "Title of the poll",
			Type:        discordgo.ApplicationCommandOptionString,
			Required:    true,
		},
	}
	for i := 0; i < 6; i++ {
		res = append(res, &discordgo.ApplicationCommandOption{
			Name:        fmt.Sprintf("option_%d", i),
			Description: fmt.Sprintf("Option number %d. Format: <emoji>;<description>", i),
			Type:        discordgo.ApplicationCommandOptionString,
			Required:    i < 2,
		})
	}

	return res
}

func interactonError(s *discordgo.Session, intr *discordgo.Interaction, content string) error {
	slog.Error(content)
	return s.InteractionRespond(intr, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "poll",
			Description: "Interact with polls",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "start",
					Description: "Start a poll",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options:     buildPollCreateArgs(),
				},
			},
		},
	}
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"poll": func(s *discordgo.Session, intr *discordgo.InteractionCreate) {
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
					err := interactonError(s, intr.Interaction, "Incorrect formatting for option %d. <emoji> ; <description>")
					if err != nil {
						slog.Error("error responding to interaction", "err", err)
					}
					return
				}

				emojiStr, labelStr := spl[0], spl[1]

				emoji := emojiComponentFromString(emojiStr)
				customID := fmt.Sprintf("%d_%s", i, strings.Trim(emojiStr, " "))

				btn := discordgo.Button{
					CustomID: customID,
					Label:    labelStr,
					Emoji:    emoji,
					Style:    discordgo.SecondaryButton,
				}

				p.Options[btn.CustomID] = []string{}

				pollButtons = append(pollButtons, btn)
			}

			err := s.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
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
				log.Printf("error responding to interaction ID %s: %v", intr.ID, err)
				return
			}

			msg, err := s.InteractionResponse(intr.Interaction)
			if err != nil {
				log.Printf("error collecting response for interaction %s: %v", intr.ID, err)
				return
			}

			p.ID = msg.ID
			err = storage.AddPoll(p)
			if err != nil {
				fmt.Printf("failed storing poll: %v", err)
			}
		},
	}
)
