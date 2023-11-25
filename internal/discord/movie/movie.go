package movie

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/LeBulldoge/gungus/internal/discord/format"
	movienight "github.com/LeBulldoge/gungus/internal/movie_night"
	"github.com/bwmarrin/discordgo"
)

func (c *MovieCommand) addMovie(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	opt := intr.ApplicationCommandData().Options[0]

	switch intr.Type {
	case discordgo.InteractionApplicationCommand:
		movieID := opt.Options[0].StringValue()
		err := movienight.AddMovie(
			context.TODO(),
			c.GetStorage(),
			movieID,
			intr.Member.User.ID,
			time.Now(),
		)

		if err != nil {
			format.DisplayInteractionError(session, intr.Interaction, fmt.Sprintf("error adding a movie: %v", err))
			return
		}

		response, err := c.buildResponseWithMovieEmbed(session, intr, movieID)
		if err != nil {
			format.DisplayInteractionError(session, intr.Interaction, fmt.Sprintf("failure displaying movie: %v", err))
		}
		response.Data.Content = "New movie added!"

		err = session.InteractionRespond(intr.Interaction, response)
		if err != nil {
			c.logger.Error("error responding to request", "err", err)
		}
	case discordgo.InteractionApplicationCommandAutocomplete:
		opt := intr.ApplicationCommandData().Options[0]

		movies, err := movienight.SearchMovies(opt.Options[0].StringValue())
		if err != nil {
			format.DisplayInteractionError(session, intr.Interaction, fmt.Sprintf("error searching movies: %v", err))
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

func embedFromMovie(session *discordgo.Session, guildId string, movie movienight.Movie) (*discordgo.MessageEmbed, error) {
	user, err := session.GuildMember(guildId, movie.AddedBy)
	if err != nil {
		return nil, err
	}

	fields := []*discordgo.MessageEmbedField{}
	if len(movie.Cast) > 0 {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name: "--- Cast ---",
		})
		for _, castMember := range movie.Cast {
			user, err := session.GuildMember(guildId, castMember.UserID)
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
			user, err := session.GuildMember(guildId, rating.UserID)
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

func (c *MovieCommand) movieList(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	movies, err := movienight.GetMovies(context.TODO(), c.GetStorage())
	if err != nil {
		format.DisplayInteractionError(session, intr.Interaction, fmt.Sprintf("error getting movies: %v", err))
		return
	}

	if len(movies) == 0 {
		format.DisplayInteractionError(session, intr.Interaction, "Movie list is empty! You can add movies via the `/movie add` command.")
		return
	}

	movie := movies[0]
	embed, err := embedFromMovie(session, intr.GuildID, movie)
	if err != nil {
		format.DisplayInteractionError(session, intr.Interaction, fmt.Sprintf("error creating embed for movie: %v", err))
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
		c.logger.Error("error responding to request", "err", err)
		return
	}
}

func (c *MovieCommand) movieListPaginate(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	err := session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		c.logger.Error("error creating deferred response", "err", err)
		return
	}

	movies, err := movienight.GetMovies(context.TODO(), c.GetStorage())
	if err != nil {
		format.DisplayInteractionError(session, intr.Interaction, fmt.Sprintf("error adding a movie: %v", err))
		return
	}

	intrMessage, err := session.ChannelMessage(intr.ChannelID, intr.Message.ID)
	if err != nil {
		format.DisplayInteractionError(session, intr.Interaction, fmt.Sprintf("error retrieving interaction message: %v", err))
		return
	}
	if len(intrMessage.Embeds) < 1 {
		format.DisplayInteractionError(session, intr.Interaction, fmt.Sprintf("error retrieving last embed: no embeds in message: %s", intrMessage.ID))
		return
	}

	lastIndex, err := strconv.Atoi(intrMessage.Embeds[0].Author.Name)
	if err != nil {
		format.DisplayInteractionError(session, intr.Interaction, fmt.Sprintf("error retrieving last embed: %v", err))
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
		format.DisplayInteractionError(session, intr.Interaction, fmt.Sprintf("error parsing movie list direction: %s", dir))
		return
	}

	movie := movies[index]
	embed, err := embedFromMovie(session, intr.GuildID, movie)
	if err != nil {
		format.DisplayInteractionError(session, intr.Interaction, fmt.Sprintf("error creating embed for movie: %v", err))
		return
	}
	embed.Author = &discordgo.MessageEmbedAuthor{
		Name: strconv.Itoa(index),
	}

	_, err = session.ChannelMessageEditEmbed(intr.ChannelID, intr.Message.ID, embed)
	if err != nil {
		c.logger.Error("error editing message", "err", err)
		return
	}
}

func (c *MovieCommand) buildResponseWithMovieEmbed(session *discordgo.Session, intr *discordgo.InteractionCreate, movieID string) (*discordgo.InteractionResponse, error) {
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

func (c *MovieCommand) rateMovie(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	opt := intr.ApplicationCommandData().Options[0]

	switch intr.Type {
	case discordgo.InteractionApplicationCommand:
		movieID := opt.Options[0].StringValue()
		rating := opt.Options[1].FloatValue()
		if math.Abs(rating) > 10.0 {
			format.DisplayInteractionError(session, intr.Interaction, fmt.Sprintf("incorrect rating value: %v", rating))
			return
		}

		err := movienight.RateMovie(context.TODO(), c.GetStorage(), movieID, intr.Member.User.ID, rating)
		if err != nil {
			format.DisplayInteractionError(session, intr.Interaction, fmt.Sprintf("failure rating a movie: %v", err))
			return
		}

		response, err := c.buildResponseWithMovieEmbed(session, intr, movieID)
		if err != nil {
			format.DisplayInteractionError(session, intr.Interaction, fmt.Sprintf("failure displaying movie: %v", err))
		}
		response.Data.Flags = discordgo.MessageFlagsEphemeral
		response.Data.Content = "Movie rated!"

		err = session.InteractionRespond(intr.Interaction, response)
		if err != nil {
			c.logger.Error("error responding to request", "err", err)
		}
	case discordgo.InteractionApplicationCommandAutocomplete:
		err := c.moveListAutocomplete(session, intr)
		if err != nil {
			c.logger.Error("failure providing autocompletion for movie/rate", "err", err)
		}
	}
}

func (c *MovieCommand) movieDelete(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	opt := intr.ApplicationCommandData().Options[0]

	switch intr.Type {
	case discordgo.InteractionApplicationCommand:
		movieID := opt.Options[0].StringValue()

		err := movienight.DeleteMovie(context.TODO(), c.GetStorage(), movieID)
		if err != nil {
			format.DisplayInteractionError(session, intr.Interaction, fmt.Sprintf("failure deleting movie %s: %v", movieID, err))
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
			c.logger.Error("error responding to request", "err", err)
		}
	case discordgo.InteractionApplicationCommandAutocomplete:
		err := c.moveListAutocomplete(session, intr)
		if err != nil {
			c.logger.Error("failure providing autocompletion for movie/rate", "err", err)
		}
	}
}

func (c *MovieCommand) moveListAutocomplete(session *discordgo.Session, intr *discordgo.InteractionCreate) error {
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

func (c *MovieCommand) movieCastAutocomplete(session *discordgo.Session, intr *discordgo.InteractionCreate) error {
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

func (c *MovieCommand) addUserAsCastMember(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	opt := intr.ApplicationCommandData().Options[0]

	switch intr.Type {
	case discordgo.InteractionApplicationCommand:
		movieID := opt.Options[0].StringValue()
		character := opt.Options[1].StringValue()

		err := movienight.AddUserAsCastMember(context.TODO(), c.GetStorage(), movieID, intr.Member.User.ID, character)
		if err != nil {
			format.DisplayInteractionError(session, intr.Interaction, fmt.Sprintf("failure adding cast member for movie %s: %v", movieID, err))
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
			c.logger.Error("error responding to request", "err", err)
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
