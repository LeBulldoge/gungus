package discord

import (
	"fmt"
	"log/slog"

	movienight "github.com/LeBulldoge/gungus/internal/movie_night"
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
			},
		},
	}
	commandHandlers = map[string]func(bot *Bot, i *discordgo.InteractionCreate){
		"movie": func(bot *Bot, intr *discordgo.InteractionCreate) {
			opt := intr.ApplicationCommandData().Options[0]

			switch intr.Type {
			case discordgo.InteractionApplicationCommand:
				err := bot.session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "You selected " + opt.Options[0].StringValue(),
					},
				})

				if err != nil {
					slog.Error("error responding to request", "err", err)
				}
			case discordgo.InteractionApplicationCommandAutocomplete:
				opt := intr.ApplicationCommandData().Options[0]

				movies, err := movienight.SearchMovies(opt.Options[0].StringValue())
				if err != nil {
					displayInteractionError(bot.session, intr.Interaction, fmt.Sprintf("error searching movies: %v", err))
					return
				}

				choices := []*discordgo.ApplicationCommandOptionChoice{}
				for _, movie := range movies {
					choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
						Name:  movie.Title,
						Value: movie.ID,
					})
				}

				err = bot.session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionApplicationCommandAutocompleteResult,
					Data: &discordgo.InteractionResponseData{
						Choices: choices,
					},
				})

				if err != nil {
					slog.Error("error responding to request", "err", err)
				}
			}
		},
		"poll":  handlePoll,
		"quote": handleQuote,
	}
)
