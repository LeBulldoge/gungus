package movie

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/LeBulldoge/gungus/internal/discord/bot"
	"github.com/LeBulldoge/gungus/internal/discord/format"
	movienight "github.com/LeBulldoge/gungus/internal/movie_night"
	"github.com/bwmarrin/discordgo"
)

type MovieCommand struct {
	logger *slog.Logger
}

func NewCommand() *MovieCommand {
	return &MovieCommand{}
}

func (c *MovieCommand) GetSignature() []*discordgo.ApplicationCommand {
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

func (c *MovieCommand) Setup(bot *bot.Bot) error {
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
				c.addMovie(bot, intr)
			case "list":
				c.movieList(bot, intr)
			case "rate":
				c.rateMovie(bot, intr)
			case "remove":
				c.movieDelete(bot, intr)
			case "cast":
				c.addUserAsCastMember(bot, intr)
			}
		case discordgo.InteractionMessageComponent:
			customID := intr.MessageComponentData().CustomID
			if strings.HasPrefix(customID, "movielist") {
				c.movieListPaginate(bot, intr)
			}
		}
	})

	return nil
}

func (c *MovieCommand) Cleanup(bot *bot.Bot) error {
	return nil
}

func (c *MovieCommand) AddLogger(logger *slog.Logger) {
	if logger == nil {
		return
	}

	c.logger = logger
}

func (c *MovieCommand) addMovie(bot *bot.Bot, intr *discordgo.InteractionCreate) {
	opt := intr.ApplicationCommandData().Options[0]

	switch intr.Type {
	case discordgo.InteractionApplicationCommand:
		movieID := opt.Options[0].StringValue()
		err := movienight.AddMovie(
			context.TODO(),
			bot.Storage,
			movieID,
			intr.Member.User.ID,
			time.Now(),
		)

		if err != nil {
			format.DisplayInteractionError(bot.Session, intr.Interaction, fmt.Sprintf("error adding a movie: %v", err))
			return
		}

		response, err := buildResponseWithMovieEmbed(bot, intr, movieID)
		if err != nil {
			format.DisplayInteractionError(bot.Session, intr.Interaction, fmt.Sprintf("failure displaying movie: %v", err))
		}
		response.Data.Content = "New movie added!"

		err = bot.Session.InteractionRespond(intr.Interaction, response)
		if err != nil {
			c.logger.Error("error responding to request", "err", err)
		}
	case discordgo.InteractionApplicationCommandAutocomplete:
		opt := intr.ApplicationCommandData().Options[0]

		movies, err := movienight.SearchMovies(opt.Options[0].StringValue())
		if err != nil {
			format.DisplayInteractionError(bot.Session, intr.Interaction, fmt.Sprintf("error searching movies: %v", err))
			return
		}

		choices := []*discordgo.ApplicationCommandOptionChoice{}
		for _, movie := range movies {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  movie.Title,
				Value: movie.ID,
			})
		}

		err = bot.Session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionApplicationCommandAutocompleteResult,
			Data: &discordgo.InteractionResponseData{
				Choices: choices,
			},
		})

		if err != nil {
			c.logger.Error("error responding to request", "err", err)
		}
	}
}

func getMemberDisplayName(member *discordgo.Member) string {
	var displayName string
	if len(member.Nick) > 0 {
		displayName = member.Nick
	} else if len(member.User.Token) > 0 {
		displayName = member.User.Token
	} else {
		displayName = member.User.Username
	}
	return displayName
}

func embedFromMovie(bot *bot.Bot, guildId string, movie movienight.Movie) (*discordgo.MessageEmbed, error) {
	user, err := bot.Session.GuildMember(guildId, movie.AddedBy)
	if err != nil {
		return nil, err
	}

	fields := []*discordgo.MessageEmbedField{}
	if len(movie.Cast) > 0 {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name: "--- Cast ---",
		})
		for _, castMember := range movie.Cast {
			user, err := bot.Session.GuildMember(guildId, castMember.UserID)
			if err != nil {
				return nil, err
			}
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   castMember.Character,
				Value:  "by " + getMemberDisplayName(user),
				Inline: true,
			})
		}
	}
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:  "\u200b",
		Value: "\u200b",
	})
	if len(movie.Ratings) > 0 {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name: "--- Ratings ---",
		})
		for _, rating := range movie.Ratings {
			user, err := bot.Session.GuildMember(guildId, rating.UserID)
			if err != nil {
				return nil, err
			}
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   strconv.FormatFloat(rating.Rating, 'f', 2, 64),
				Value:  "by " + getMemberDisplayName(user),
				Inline: true,
			})
		}
	}

	return &discordgo.MessageEmbed{
		Title:       movie.Title,
		URL:         movie.GetURL(),
		Description: movie.Description,
		Image: &discordgo.MessageEmbedImage{
			URL: movie.Image,
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Added by " + getMemberDisplayName(user),
		},
		Timestamp: movie.WatchedOn.Format(time.RFC3339),
		Fields:    fields,
	}, nil
}

func (c *MovieCommand) movieList(bot *bot.Bot, intr *discordgo.InteractionCreate) {
	movies, err := movienight.GetMovies(context.TODO(), bot.Storage)
	if err != nil {
		format.DisplayInteractionError(bot.Session, intr.Interaction, fmt.Sprintf("error getting movies: %v", err))
		return
	}

	if len(movies) == 0 {
		format.DisplayInteractionError(bot.Session, intr.Interaction, "Movie list is empty! You can add movies via the `/movie add` command.")
		return
	}

	movie := movies[0]
	embed, err := embedFromMovie(bot, intr.GuildID, movie)
	if err != nil {
		format.DisplayInteractionError(bot.Session, intr.Interaction, fmt.Sprintf("error creating embed for movie: %v", err))
		return
	}

	embed.Author = &discordgo.MessageEmbedAuthor{
		Name: "0",
	}

	err = bot.Session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							CustomID: "movielist_" + intr.ID + "_back",
							Emoji: discordgo.ComponentEmoji{
								Name: "⬅️",
							},
							Style: discordgo.SecondaryButton,
						},
						discordgo.Button{
							CustomID: "movielist_" + intr.ID + "_forward",
							Emoji: discordgo.ComponentEmoji{
								Name: "➡️",
							},
							Style: discordgo.SecondaryButton,
						},
						discordgo.Button{
							CustomID: "movielist_" + intr.ID + "_refresh",
							Emoji: discordgo.ComponentEmoji{
								Name: "🔄",
							},
							Style: discordgo.SecondaryButton,
						},
					},
				},
			},
		},
	})

	if err != nil {
		c.logger.Error("error responding to request", "err", err)
		return
	}
}

func (c *MovieCommand) movieListPaginate(bot *bot.Bot, intr *discordgo.InteractionCreate) {
	err := bot.Session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		c.logger.Error("error creating deferred response", "err", err)
		return
	}

	movies, err := movienight.GetMovies(context.TODO(), bot.Storage)
	if err != nil {
		format.DisplayInteractionError(bot.Session, intr.Interaction, fmt.Sprintf("error adding a movie: %v", err))
		return
	}

	intrMessage, err := bot.Session.ChannelMessage(intr.ChannelID, intr.Message.ID)
	if err != nil {
		format.DisplayInteractionError(bot.Session, intr.Interaction, fmt.Sprintf("error retrieving interaction message: %v", err))
		return
	}
	if len(intrMessage.Embeds) < 1 {
		format.DisplayInteractionError(bot.Session, intr.Interaction, fmt.Sprintf("error retrieving last embed: no embeds in message: %s", intrMessage.ID))
		return
	}

	lastIndex, err := strconv.Atoi(intrMessage.Embeds[0].Author.Name)
	if err != nil {
		format.DisplayInteractionError(bot.Session, intr.Interaction, fmt.Sprintf("error retrieving last embed: %v", err))
		return
	}

	var index int

	customIDSplit := strings.Split(intr.MessageComponentData().CustomID, "_")
	dir := customIDSplit[len(customIDSplit)-1]

	switch dir {
	case "forward":
		if len(movies) > lastIndex+1 {
			index = lastIndex + 1
		} else {
			index = 0
		}
	case "back":
		if 0 <= lastIndex-1 {
			index = lastIndex - 1
		} else {
			index = len(movies) - 1
		}
	case "refresh":
		index = lastIndex
	default:
		format.DisplayInteractionError(bot.Session, intr.Interaction, fmt.Sprintf("error parsing movie list direction: %s", dir))
		return
	}

	movie := movies[index]
	embed, err := embedFromMovie(bot, intr.GuildID, movie)
	if err != nil {
		format.DisplayInteractionError(bot.Session, intr.Interaction, fmt.Sprintf("error creating embed for movie: %v", err))
		return
	}
	embed.Author = &discordgo.MessageEmbedAuthor{
		Name: strconv.Itoa(index),
	}

	_, err = bot.Session.ChannelMessageEditEmbed(intr.ChannelID, intr.Message.ID, embed)
	if err != nil {
		c.logger.Error("error editing message", "err", err)
		return
	}
}

func buildResponseWithMovieEmbed(bot *bot.Bot, intr *discordgo.InteractionCreate, movieID string) (*discordgo.InteractionResponse, error) {
	movie, err := movienight.GetMovie(context.TODO(), bot.Storage, movieID)
	if err != nil {
		return nil, fmt.Errorf("failure getting a movie: %w", err)
	}

	embed, err := embedFromMovie(bot, intr.GuildID, movie)
	if err != nil {
		return nil, fmt.Errorf("failure building an embed: %w", err)
	}

	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	}, nil
}

func (c *MovieCommand) rateMovie(bot *bot.Bot, intr *discordgo.InteractionCreate) {
	opt := intr.ApplicationCommandData().Options[0]

	switch intr.Type {
	case discordgo.InteractionApplicationCommand:
		movieID := opt.Options[0].StringValue()
		rating := opt.Options[1].FloatValue()
		if math.Abs(rating) > 10.0 {
			format.DisplayInteractionError(bot.Session, intr.Interaction, fmt.Sprintf("incorrect rating value: %v", rating))
			return
		}

		err := movienight.RateMovie(context.TODO(), bot.Storage, movieID, intr.Member.User.ID, rating)
		if err != nil {
			format.DisplayInteractionError(bot.Session, intr.Interaction, fmt.Sprintf("failure rating a movie: %v", err))
			return
		}

		response, err := buildResponseWithMovieEmbed(bot, intr, movieID)
		if err != nil {
			format.DisplayInteractionError(bot.Session, intr.Interaction, fmt.Sprintf("failure displaying movie: %v", err))
		}
		response.Data.Flags = discordgo.MessageFlagsEphemeral
		response.Data.Content = "Movie rated!"

		err = bot.Session.InteractionRespond(intr.Interaction, response)
		if err != nil {
			c.logger.Error("error responding to request", "err", err)
		}
	case discordgo.InteractionApplicationCommandAutocomplete:
		err := c.moveListAutocomplete(bot, intr)
		if err != nil {
			c.logger.Error("failure providing autocompletion for movie/rate", "err", err)
		}
	}
}

func (c *MovieCommand) movieDelete(bot *bot.Bot, intr *discordgo.InteractionCreate) {
	opt := intr.ApplicationCommandData().Options[0]

	switch intr.Type {
	case discordgo.InteractionApplicationCommand:
		movieID := opt.Options[0].StringValue()

		err := movienight.DeleteMovie(context.TODO(), bot.Storage, movieID)
		if err != nil {
			format.DisplayInteractionError(bot.Session, intr.Interaction, fmt.Sprintf("failure deleting movie %s: %v", movieID, err))
			return
		}

		err = bot.Session.InteractionRespond(intr.Interaction,
			&discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("Movie `%s` successfully deleted!", movieID),
				},
			})
		if err != nil {
			c.logger.Error("error responding to request", "err", err)
		}
	case discordgo.InteractionApplicationCommandAutocomplete:
		err := c.moveListAutocomplete(bot, intr)
		if err != nil {
			c.logger.Error("failure providing autocompletion for movie/rate", "err", err)
		}
	}
}

func (c *MovieCommand) moveListAutocomplete(bot *bot.Bot, intr *discordgo.InteractionCreate) error {
	opt := intr.ApplicationCommandData().Options[0]
	title := opt.Options[0].StringValue()
	movies, err := movienight.GetMoviesByTitle(context.TODO(), bot.Storage, title)
	if err != nil {
		return fmt.Errorf("failure getting movies by title: %w", err)
	}

	choices := []*discordgo.ApplicationCommandOptionChoice{}
	for _, movie := range movies {
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  movie.Title,
			Value: movie.ID,
		})
	}

	err = bot.Session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})

	if err != nil {
		return fmt.Errorf("error responding to request: %w", err)
	}

	return nil
}

func (c *MovieCommand) movieCastAutocomplete(bot *bot.Bot, intr *discordgo.InteractionCreate) error {
	opt := intr.ApplicationCommandData().Options[0]
	cast, err := movienight.SearchCharacters(opt.Options[0].StringValue(), opt.Options[1].StringValue())
	if err != nil {
		return fmt.Errorf("failure getting movie cast: %w", err)
	}

	if len(cast) > 25 {
		cast = cast[:25]
	}

	choices := []*discordgo.ApplicationCommandOptionChoice{}
	for _, character := range cast {
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  character,
			Value: character,
		})
	}

	err = bot.Session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})

	if err != nil {
		return fmt.Errorf("error responding to request: %w", err)
	}

	return nil
}

func (c *MovieCommand) addUserAsCastMember(bot *bot.Bot, intr *discordgo.InteractionCreate) {
	opt := intr.ApplicationCommandData().Options[0]

	switch intr.Type {
	case discordgo.InteractionApplicationCommand:
		movieID := opt.Options[0].StringValue()
		character := opt.Options[1].StringValue()

		err := movienight.AddUserAsCastMember(context.TODO(), bot.Storage, movieID, intr.Member.User.ID, character)
		if err != nil {
			format.DisplayInteractionError(bot.Session, intr.Interaction, fmt.Sprintf("failure adding cast member for movie %s: %v", movieID, err))
			return
		}

		err = bot.Session.InteractionRespond(intr.Interaction,
			&discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Movie cast member added!",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		if err != nil {
			c.logger.Error("error responding to request", "err", err)
		}
	case discordgo.InteractionApplicationCommandAutocomplete:
		if opt.Options[0].Focused {
			err := c.moveListAutocomplete(bot, intr)
			if err != nil {
				c.logger.Error("failure providing autocompletion for movie/cast/title", "err", err)
			}
		} else {
			err := c.movieCastAutocomplete(bot, intr)
			if err != nil {
				c.logger.Error("failure providing autocompletion for movie/cast/character", "err", err)
			}
		}
	}
}
