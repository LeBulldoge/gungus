package discord

import (
	"context"
	"fmt"
	"log/slog"
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
		err := movienight.AddMovie(
			context.TODO(),
			bot.storage,
			opt.Options[0].StringValue(),
			intr.Member.User.ID,
			time.Now(),
		)

		if err != nil {
			displayInteractionError(bot.session, intr.Interaction, fmt.Sprintf("error adding a movie: %v", err))
			return
		}

		err = bot.session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Movie added!",
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
}

func embedFromMovie(user *discordgo.User, movie movienight.Movie) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       movie.Title,
		URL:         movie.GetURL(),
		Description: movie.Description,
		Image: &discordgo.MessageEmbedImage{
			URL: movie.Image,
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Added by " + user.String(),
		},
		Timestamp: movie.WatchedOn.Format(time.RFC3339),
	}
}

func movieList(bot *Bot, intr *discordgo.InteractionCreate) {
	movies, err := movienight.GetMovies(context.TODO(), bot.storage)
	if err != nil {
		displayInteractionError(bot.session, intr.Interaction, fmt.Sprintf("error adding a movie: %v", err))
		return
	}

	movie := movies[0]
	user, err := bot.session.User(movie.AddedBy)
	if err != nil {
		displayInteractionError(bot.session, intr.Interaction, fmt.Sprintf("error retrieving user: %v", err))
		return
	}

	embed := embedFromMovie(user, movie)
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
	user, err := bot.session.User(movie.AddedBy)
	if err != nil {
		displayInteractionError(bot.session, intr.Interaction, fmt.Sprintf("error retrieving user: %v", err))
		return
	}

	embed := embedFromMovie(user, movie)
	embed.Author = &discordgo.MessageEmbedAuthor{
		Name: strconv.Itoa(index),
	}

	_, err = bot.session.ChannelMessageEditEmbed(intr.ChannelID, intr.Message.ID, embed)
	if err != nil {
		slog.Error("error editing message", "err", err)
		return
	}
}

func handleMovie(bot *Bot, intr *discordgo.InteractionCreate) {
	data := intr.ApplicationCommandData().Options[0]
	switch data.Name {
	case "add":
		addMovie(bot, intr)
	case "list":
		movieList(bot, intr)
	}
}
