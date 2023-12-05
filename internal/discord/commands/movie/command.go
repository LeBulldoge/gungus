package movie

import (
	"log/slog"
	"strings"

	"github.com/LeBulldoge/gungus/internal/database"
	"github.com/LeBulldoge/gungus/internal/discord/bot"
	"github.com/bwmarrin/discordgo"
)

type Command struct {
	database.WithStorage

	logger *slog.Logger
}

func NewCommand() *Command {
	return &Command{}
}

func (c *Command) GetSignature() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
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
					Name:        "cast",
					Description: "Tag yourself in the movie",
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
							Name:         "character",
							Description:  "Name of the character",
							Type:         discordgo.ApplicationCommandOptionString,
							Required:     true,
							Autocomplete: true,
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
}

func (c *Command) Setup(bot *bot.Bot) error {
	bot.Session.AddHandler(func(sesh *discordgo.Session, intr *discordgo.InteractionCreate) {
		switch intr.Type {
		case discordgo.InteractionApplicationCommand:
			fallthrough
		case discordgo.InteractionApplicationCommandAutocomplete:
			data := intr.ApplicationCommandData()
			if data.Name != "movie" {
				return
			}
			subData := intr.ApplicationCommandData().Options[0]
			switch subData.Name {
			case "add":
				c.addMovie(sesh, intr)
			case "list":
				c.movieList(sesh, intr)
			case "rate":
				c.rateMovie(sesh, intr)
			case "remove":
				c.movieDelete(sesh, intr)
			case "cast":
				c.addUserAsCastMember(sesh, intr)
			}
		case discordgo.InteractionMessageComponent:
			customID := intr.MessageComponentData().CustomID
			if strings.HasPrefix(customID, "movielist") {
				c.movieListPaginate(sesh, intr)
			}
		}
	})

	return nil
}

func (c *Command) Cleanup(bot *bot.Bot) error {
	return nil
}

func (c *Command) AddLogger(logger *slog.Logger) {
	if logger == nil {
		return
	}

	c.logger = logger
}
