package discord

import (
	"fmt"
	"log/slog"

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

func displayInteractionError(s *discordgo.Session, intr *discordgo.Interaction, content string) {
	slog.Error(content)
	err := s.InteractionRespond(intr, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		slog.Error("failed displaying error", "err", err)
	}
}

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "poll",
			Description: "Interact with polls",
			Type:        discordgo.ChatApplicationCommand,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "start",
					Description: "Start a poll",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options:     buildPollCreateArgs(),
				},
			},
		},
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
		{
			Name:        "movie",
			Description: "Interact with movies",
			Type:        discordgo.ChatApplicationCommand,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Description: "Add a movie to the list",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:         "title",
							Description:  "Title of the movie",
							Type:         discordgo.ApplicationCommandOptionString,
							Required:     true,
							Autocomplete: true,
						},
					},
				},
				{
					Name:        "list",
					Description: "Browse the movie list",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "rate",
					Description: "Rate movie in the list",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:         "title",
							Description:  "Title of the movie",
							Type:         discordgo.ApplicationCommandOptionString,
							Required:     true,
							Autocomplete: true,
						},
						{
							Name:        "rating",
							Description: "Rating of the movie from -10.0 to 10.0",
							Type:        discordgo.ApplicationCommandOptionNumber,
							Required:    true,
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove a movie from the list",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:         "title",
							Description:  "Title of the movie",
							Type:         discordgo.ApplicationCommandOptionString,
							Required:     true,
							Autocomplete: true,
						},
					},
				},
			},
		},
	}
	commandHandlers = map[string]func(bot *Bot, i *discordgo.InteractionCreate){
		"movie": handleMovie,
		"poll":  handlePoll,
		"quote": handleQuote,
	}
)
