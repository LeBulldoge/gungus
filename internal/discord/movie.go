package discord

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"

	movienight "github.com/LeBulldoge/gungus/internal/movie_night"
	"github.com/bwmarrin/discordgo"
)

func addMovie(bot *Bot, intr *discordgo.InteractionCreate) {
	opt := intr.ApplicationCommandData().Options[0]

	switch intr.Type {
	case discordgo.InteractionApplicationCommand:
		movieID := opt.Options[0].StringValue()
		err := movienight.AddMovie(
			context.TODO(),
			bot.storage,
			movieID,
			intr.Member.User.ID,
			time.Now(),
		)

		if err != nil {
			displayInteractionError(bot.session, intr.Interaction, fmt.Sprintf("error adding a movie: %v", err))
			return
		}

		response, err := buildMovieEmbedResponse(bot, intr, movieID)
		if err != nil {
			displayInteractionError(bot.session, intr.Interaction, fmt.Sprintf("failure displaying movie: %v", err))
		}
		response.Data.Content = "New movie added!"

		err = bot.session.InteractionRespond(intr.Interaction, response)
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

func embedFromMovie(bot *Bot, guildId string, movie movienight.Movie) (*discordgo.MessageEmbed, error) {
	user, err := bot.session.GuildMember(guildId, movie.AddedBy)
	if err != nil {
		return nil, err
	}

	fields := []*discordgo.MessageEmbedField{}
	for _, rating := range movie.Ratings {
		user, err := bot.session.GuildMember(guildId, rating.UserID)
		if err != nil {
			return nil, err
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Rating by " + getMemberDisplayName(user),
			Value:  strconv.FormatFloat(rating.Rating, 'f', 2, 64),
			Inline: true,
		})
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

func movieList(bot *Bot, intr *discordgo.InteractionCreate) {
	movies, err := movienight.GetMovies(context.TODO(), bot.storage)
	if err != nil {
		displayInteractionError(bot.session, intr.Interaction, fmt.Sprintf("error adding a movie: %v", err))
		return
	}
	if len(movies) == 0 {
		displayInteractionError(bot.session, intr.Interaction, "Movie list is empty! You can add movies via the `/movie add` command.")
		return
	}

	movie := movies[0]
	embed, err := embedFromMovie(bot, intr.GuildID, movie)
	if err != nil {
		displayInteractionError(bot.session, intr.Interaction, fmt.Sprintf("error creating embed for movie: %v", err))
		return
	}

	embed.Author = &discordgo.MessageEmbedAuthor{
		Name: "0",
	}

	err = bot.session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
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
					},
				},
			},
		},
	})

	if err != nil {
		slog.Error("error responding to request", "err", err)
		return
	}
}

func movieListPaginate(bot *Bot, intr *discordgo.InteractionCreate) {
	err := bot.session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		slog.Error("error creating deferred response", "err", err)
		return
	}

	movies, err := movienight.GetMovies(context.TODO(), bot.storage)
	if err != nil {
		displayInteractionError(bot.session, intr.Interaction, fmt.Sprintf("error adding a movie: %v", err))
		return
	}

	intrMessage, err := bot.session.ChannelMessage(intr.ChannelID, intr.Message.ID)
	if err != nil {
		displayInteractionError(bot.session, intr.Interaction, fmt.Sprintf("error retrieving interaction message: %v", err))
		return
	}
	if len(intrMessage.Embeds) < 1 {
		displayInteractionError(bot.session, intr.Interaction, fmt.Sprintf("error retrieving last embed: no embeds in message: %s", intrMessage.ID))
		return
	}

	lastIndex, err := strconv.Atoi(intrMessage.Embeds[0].Author.Name)
	if err != nil {
		displayInteractionError(bot.session, intr.Interaction, fmt.Sprintf("error retrieving last embed: %v", err))
		return
	}

	index := lastIndex

	customIDSplit := strings.Split(intr.MessageComponentData().CustomID, "_")
	dir := customIDSplit[len(customIDSplit)-1]

	switch dir {
	case "forward":
		if len(movies) > lastIndex+1 {
			index = lastIndex + 1
		}
	case "back":
		if 0 <= lastIndex-1 {
			index = lastIndex - 1
		}
	default:
		displayInteractionError(bot.session, intr.Interaction, fmt.Sprintf("error parsing movie list direction: %s", dir))
		return
	}

	movie := movies[index]
	embed, err := embedFromMovie(bot, intr.GuildID, movie)
	if err != nil {
		displayInteractionError(bot.session, intr.Interaction, fmt.Sprintf("error creating embed for movie: %v", err))
		return
	}
	embed.Author = &discordgo.MessageEmbedAuthor{
		Name: strconv.Itoa(index),
	}

	_, err = bot.session.ChannelMessageEditEmbed(intr.ChannelID, intr.Message.ID, embed)
	if err != nil {
		slog.Error("error editing message", "err", err)
		return
	}
}

func buildMovieEmbedResponse(bot *Bot, intr *discordgo.InteractionCreate, movieID string) (*discordgo.InteractionResponse, error) {
	movie, err := movienight.GetMovie(context.TODO(), bot.storage, movieID)
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

func rateMovie(bot *Bot, intr *discordgo.InteractionCreate) {
	opt := intr.ApplicationCommandData().Options[0]

	switch intr.Type {
	case discordgo.InteractionApplicationCommand:
		movieID := opt.Options[0].StringValue()
		rating := opt.Options[1].FloatValue()
		if math.Abs(rating) > 10.0 {
			displayInteractionError(bot.session, intr.Interaction, fmt.Sprintf("incorrect rating value: %v", rating))
			return
		}

		err := movienight.RateMovie(context.TODO(), bot.storage, movieID, intr.Member.User.ID, rating)
		if err != nil {
			displayInteractionError(bot.session, intr.Interaction, fmt.Sprintf("failure rating a movie: %v", err))
			return
		}

		response, err := buildMovieEmbedResponse(bot, intr, movieID)
		if err != nil {
			displayInteractionError(bot.session, intr.Interaction, fmt.Sprintf("failure displaying movie: %v", err))
		}
		response.Data.Flags = discordgo.MessageFlagsEphemeral
		response.Data.Content = "Movie rated!"

		err = bot.session.InteractionRespond(intr.Interaction, response)
		if err != nil {
			slog.Error("error responding to request", "err", err)
		}
	case discordgo.InteractionApplicationCommandAutocomplete:
		title := opt.Options[0].StringValue()
		movies, err := movienight.GetMoviesByTitle(context.TODO(), bot.storage, title)
		if err != nil {
			displayInteractionError(bot.session, intr.Interaction, fmt.Sprintf("error getting movies: %v", err))
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
}

func handleMovie(bot *Bot, intr *discordgo.InteractionCreate) {
	data := intr.ApplicationCommandData().Options[0]
	switch data.Name {
	case "add":
		addMovie(bot, intr)
	case "list":
		movieList(bot, intr)
	case "rate":
		rateMovie(bot, intr)
	}
}
