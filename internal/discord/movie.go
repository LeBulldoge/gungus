package discord

import (
	"context"
	"fmt"
	"log/slog"
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

	embed := &discordgo.MessageEmbed{
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

	err = bot.session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	if err != nil {
		slog.Error("error responding to request", "err", err)
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
