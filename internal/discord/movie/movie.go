package movie

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/LeBulldoge/gungus/internal/discord/embed"
	"github.com/LeBulldoge/gungus/internal/discord/format"
	movienight "github.com/LeBulldoge/gungus/internal/movie_night"
	"github.com/bwmarrin/discordgo"
)

func (c *Command) addMovie(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	opt := intr.ApplicationCommandData().Options[0]

	switch intr.Type {
	case discordgo.InteractionApplicationCommand:
		movieID := opt.Options[0].StringValue()
		log := c.logger.With(
			slog.Group(
				"add",
				"movieId", movieID,
			),
		)

		err := movienight.AddMovie(
			context.TODO(),
			c.GetStorage(),
			movieID,
			intr.Member.User.ID,
			time.Now(),
		)
		if err != nil {
			log.Error("error adding a movie", "err", err)
			format.DisplayInteractionWithError(session, intr, "Error adding a movie.", err)
			return
		}

		response, err := c.buildResponseWithMovieEmbed(session, intr, movieID)
		if err != nil {
			log.Error("error displaying added movie", "err", err)
			format.DisplayInteractionWithError(session, intr, "Error displaying added movie.", err)
			return
		}
		response.Data.Content = "New movie added!"

		err = session.InteractionRespond(intr.Interaction, response)
		if err != nil {
			c.logger.Error("error responding to request", "err", err)
			return
		}

		log.Info("movie added")
	case discordgo.InteractionApplicationCommandAutocomplete:
		opt := intr.ApplicationCommandData().Options[0]
		movieID := opt.Options[0].StringValue()
		log := c.logger.With(
			slog.Group(
				"add/autocomplete",
				"movieId", movieID,
			),
		)

		movies, err := movienight.SearchMovies(movieID)
		if err != nil {
			log.Error("error searching movies", "err", err)
			format.DisplayInteractionWithError(session, intr, "Error searching movies.", err)
			return
		}

		choices := []*discordgo.ApplicationCommandOptionChoice{}
		for _, movie := range movies {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  movie.Title,
				Value: movie.ID,
			})
		}

		err = session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionApplicationCommandAutocompleteResult,
			Data: &discordgo.InteractionResponseData{
				Choices: choices,
			},
		})
		if err != nil {
			log.Error("error responding to request", "err", err)
		}
	}
}

func embedFromMovie(session *discordgo.Session, guildID string, movie movienight.Movie) (*discordgo.MessageEmbed, error) {
	user, err := session.GuildMember(guildID, movie.AddedBy)
	if err != nil {
		return nil, err
	}

	embed := embed.NewEmbed()

	if len(movie.Cast) > 0 {
		embed.AddField("--- Cast ---", "")
		for _, castMember := range movie.Cast {
			user, err := session.GuildMember(guildID, castMember.UserID)
			if err != nil {
				return nil, err
			}
			embed.AddInlineField(castMember.Character, "by "+format.GetMemberDisplayName(user))
		}
	}
	embed.AddField("\u200b", "\u200b")
	if len(movie.Ratings) > 0 {
		embed.AddField("--- Ratings ---", "")
		for _, rating := range movie.Ratings {
			user, err := session.GuildMember(guildID, rating.UserID)
			if err != nil {
				return nil, err
			}
			embed.AddInlineField(
				strconv.FormatFloat(rating.Rating, 'f', 2, 64),
				"by "+format.GetMemberDisplayName(user),
			)
		}
	}

	res := embed.
		SetTitle(movie.Title).
		SetUrl(movie.GetURL()).
		SetDescription(movie.Description).
		SetImage(movie.Image).
		SetFooter("Added by "+format.GetMemberDisplayName(user), "").
		SetTimestamp(movie.WatchedOn.Format(time.RFC3339)).
		MessageEmbed

	return res, nil
}

func (c *Command) movieList(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	movies, err := movienight.GetMovies(context.TODO(), c.GetStorage())
	log := c.logger.With(
		slog.Group(
			"list",
			"moviesCount", len(movies),
		),
	)

	if err != nil {
		log.Error("error getting movies", "err", err)
		format.DisplayInteractionWithError(session, intr, "Error getting movies.", err)
		return
	}

	if len(movies) == 0 {
		log.Error("movie list is empty")
		format.DisplayInteractionError(session, intr, "Movie list is empty! You can add movies via the `/movie add` command.")
		return
	}

	movie := movies[0]
	embed, err := embedFromMovie(session, intr.GuildID, movie)
	if err != nil {
		log.Error("failure constructing embed from movie", "err", err)
		format.DisplayInteractionWithError(session, intr, "Error displaying movie.", err)
		return
	}

	embed.Author = &discordgo.MessageEmbedAuthor{
		Name: "0",
	}

	err = session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
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
		log.Error("error responding to request", "err", err)
		return
	}
}

func (c *Command) movieListPaginate(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	log := c.logger.WithGroup("list/paginate").With(
		"messageId", intr.Message.ID,
	)

	err := session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		log.Error("error creating deferred response", "err", err)
		return
	}

	movies, err := movienight.GetMovies(context.TODO(), c.GetStorage())
	if err != nil {
		log.Error("error getting a movie", "err", err)
		format.DisplayInteractionWithError(session, intr, "Error getting movie.", err)
		return
	}

	log = log.With("movieCount", len(movies))

	intrMessage, err := session.ChannelMessage(intr.ChannelID, intr.Message.ID)
	if err != nil {
		log.Error("error getting interaction message", "err", err)
		format.DisplayInteractionError(session, intr, fmt.Sprintf("error retrieving interaction message: %v", err))
		return
	}
	if len(intrMessage.Embeds) < 1 {
		format.DisplayInteractionError(session, intr, fmt.Sprintf("error retrieving last embed: no embeds in message: %s", intrMessage.ID))
		return
	}

	lastIndex, err := strconv.Atoi(intrMessage.Embeds[0].Author.Name)
	if err != nil {
		format.DisplayInteractionError(session, intr, fmt.Sprintf("error retrieving last embed: %v", err))
		return
	}

	log = log.With("lastMovieIndex", lastIndex)

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
		format.DisplayInteractionError(session, intr, fmt.Sprintf("error parsing movie list direction: %s", dir))
		return
	}

	movie := movies[index]
	log = log.With("nextMovieIndex", index, "nextMovieTitle", movie.Title)

	embed, err := embedFromMovie(session, intr.GuildID, movie)
	if err != nil {
		log.Error("error creating embed for movie", "err", err)
		format.DisplayInteractionWithError(session, intr, "Error displaying movie.", err)
		return
	}
	embed.Author = &discordgo.MessageEmbedAuthor{
		Name: strconv.Itoa(index),
	}

	_, err = session.ChannelMessageEditEmbed(intr.ChannelID, intr.Message.ID, embed)
	if err != nil {
		log.Error("error editing message", "err", err)
		return
	}
}

func (c *Command) buildResponseWithMovieEmbed(session *discordgo.Session, intr *discordgo.InteractionCreate, movieID string) (*discordgo.InteractionResponse, error) {
	movie, err := movienight.GetMovie(context.TODO(), c.GetStorage(), movieID)
	if err != nil {
		return nil, fmt.Errorf("failure getting a movie: %w", err)
	}

	embed, err := embedFromMovie(session, intr.GuildID, movie)
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

func (c *Command) rateMovie(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	opt := intr.ApplicationCommandData().Options[0]

	switch intr.Type {
	case discordgo.InteractionApplicationCommand:
		movieID := opt.Options[0].StringValue()
		rating := opt.Options[1].FloatValue()

		log := c.logger.WithGroup("rate").With("movieId", movieID, "rating", rating)

		if math.Abs(rating) > 10.0 {
			log.Error("incorrect rating value")
			format.DisplayInteractionError(session, intr, "Incorrect rating value, must be within -10 to 10.")
			return
		}

		err := movienight.RateMovie(context.TODO(), c.GetStorage(), movieID, intr.Member.User.ID, rating)
		if err != nil {
			log.Error("failure rating a movie", "err", err)
			format.DisplayInteractionWithError(session, intr, "Error rating movie.", err)
			return
		}

		response, err := c.buildResponseWithMovieEmbed(session, intr, movieID)
		if err != nil {
			log.Error("failure displaying a movie", "err", err)
			format.DisplayInteractionError(session, intr, "Failure displaying movie.")
			return
		}
		response.Data.Flags = discordgo.MessageFlagsEphemeral
		response.Data.Content = "Movie rated!"

		err = session.InteractionRespond(intr.Interaction, response)
		if err != nil {
			log.Error("error responding to request", "err", err)
		}
	case discordgo.InteractionApplicationCommandAutocomplete:
		err := c.moveListAutocomplete(session, intr)
		if err != nil {
			c.logger.Error("failure providing autocompletion for movie/rate", "err", err)
		}
	}
}

func (c *Command) movieDelete(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	opt := intr.ApplicationCommandData().Options[0]

	switch intr.Type {
	case discordgo.InteractionApplicationCommand:
		movieID := opt.Options[0].StringValue()
		log := c.logger.WithGroup("delete").With("movieId", movieID)

		err := movienight.DeleteMovie(context.TODO(), c.GetStorage(), movieID)
		if err != nil {
			log.Error("failure deleting movie", "err", err)
			format.DisplayInteractionWithError(session, intr, "Error deleting movie.", err)
			return
		}

		err = session.InteractionRespond(intr.Interaction,
			&discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("Movie `%s` successfully deleted!", movieID),
				},
			})
		if err != nil {
			log.Error("error responding to request", "err", err)
		}
	case discordgo.InteractionApplicationCommandAutocomplete:
		err := c.moveListAutocomplete(session, intr)
		if err != nil {
			c.logger.Error("failure providing autocompletion for movie/rate", "err", err)
		}
	}
}

func (c *Command) moveListAutocomplete(session *discordgo.Session, intr *discordgo.InteractionCreate) error {
	opt := intr.ApplicationCommandData().Options[0]
	title := opt.Options[0].StringValue()
	movies, err := movienight.GetMoviesByTitle(context.TODO(), c.GetStorage(), title)
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

	err = session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
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

func (c *Command) movieCastAutocomplete(session *discordgo.Session, intr *discordgo.InteractionCreate) error {
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

	err = session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
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

func (c *Command) addUserAsCastMember(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	opt := intr.ApplicationCommandData().Options[0]

	switch intr.Type {
	case discordgo.InteractionApplicationCommand:
		movieID := opt.Options[0].StringValue()
		character := opt.Options[1].StringValue()

		log := c.logger.WithGroup("cast").With("movieId", movieID, "character", character)

		err := movienight.AddUserAsCastMember(context.TODO(), c.GetStorage(), movieID, intr.Member.User.ID, character)
		if err != nil {
			log.Error("failure adding cast member for movie", "err", err)
			format.DisplayInteractionWithError(session, intr, "Error adding you as a cast member.", err)
			return
		}

		err = session.InteractionRespond(intr.Interaction,
			&discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Movie cast member added!",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		if err != nil {
			log.Error("error responding to request", "err", err)
		}
	case discordgo.InteractionApplicationCommandAutocomplete:
		if opt.Options[0].Focused {
			err := c.moveListAutocomplete(session, intr)
			if err != nil {
				c.logger.Error("failure providing autocompletion for movie/cast/title", "err", err)
			}
		} else {
			err := c.movieCastAutocomplete(session, intr)
			if err != nil {
				c.logger.Error("failure providing autocompletion for movie/cast/character", "err", err)
			}
		}
	}
}
