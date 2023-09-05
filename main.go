package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// flags
var (
	BotToken = flag.String("token", "", "Bot token")
)

var polls = make(map[string]Poll)

func main() {
	flag.Parse()

	sesh, err := discordgo.New("Bot " + *BotToken)
	if err != nil {
		log.Fatalf("error while creating session: %v", err)
	}

	sesh.AddHandler(func(s *discordgo.Session, intr *discordgo.InteractionCreate) {
		switch intr.Type {
		case discordgo.InteractionApplicationCommand:
			if h, ok := commandHandlers[intr.ApplicationCommandData().Name]; ok {
				h(s, intr)
			}
		case discordgo.InteractionMessageComponent:
			err = s.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredMessageUpdate,
			})

			var votes map[string]int
			voteCustomID := intr.MessageComponentData().CustomID
			if poll, ok := polls[intr.Message.ID]; ok {
				err = poll.castVote(intr.Member.User.ID, voteCustomID)
				if err != nil {
					log.Printf("error casting vote for poll %s: %v", intr.Message.ID, err)
				}

				votes = poll.getVotes()
			} else {
				log.Printf("error getting poll for message ID %s", intr.Message.ID)
				return
			}

			chartStr := plotBarChart("Plot", votes)
			msg := discordgo.NewMessageEdit(intr.ChannelID, intr.Message.ID)
			msg.Content = &chartStr

			_, err = s.ChannelMessageEditComplex(msg)
			if err != nil {
				log.Printf("error editing message %s: %v", intr.Message.ID, err)
			}
		}
	})

	err = sesh.Open()
	if err != nil {
		log.Fatalf("error while opening session: %v", err)
	}

	for _, v := range commands {
		_, err := sesh.ApplicationCommandCreate(sesh.State.User.ID, "", v)
		if err != nil {
			log.Fatalf("error while creating command %s: %v", v.Name, err)
		}

		log.Printf("created command '%s'", v.Name)
	}

	defer sesh.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Press Ctrl+C to exit")
	<-stop

	log.Println("Removing commands...")
	registeredCommands, err := sesh.ApplicationCommands(sesh.State.User.ID, "")
	if err != nil {
		log.Fatalf("Could not fetch registered commands: %v", err)
	}

	for _, v := range registeredCommands {
		err := sesh.ApplicationCommandDelete(sesh.State.User.ID, "", v.ID)
		if err != nil {
			log.Panicf("Cannot delete '%s' command: %v", v.Name, err)
		}
	}

	log.Println("Gracefully shutting down.")
}

func isCustomEmoji(s string) bool {
	return s[0] == '<'
}

func emojiComponentFromString(s string) discordgo.ComponentEmoji {
	emoji := discordgo.ComponentEmoji{}
	if isCustomEmoji(s) {
		s = s[1 : len(s)-2]
		parts := strings.Split(s, ":")

		emoji.Animated = parts[0] == "a"
		emoji.Name = parts[1]
		emoji.ID = parts[2]
	} else {
		emoji.Name = strings.Trim(s, " ")
	}

	return emoji
}
