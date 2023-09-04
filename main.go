package main

import (
	"flag"
	"fmt"
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

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "poll",
			Description: "Interact with polls",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "start",
					Description: "Start a poll",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options:     buildPollCreateArgs(),
				},
			},
		},
	}
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"poll": func(s *discordgo.Session, intr *discordgo.InteractionCreate) {
			opt := intr.ApplicationCommandData().Options[0]

			var pollAnsText []string
			for _, v := range opt.Options[1:] {
				if v.StringValue() != "" {
					pollAnsText = append(pollAnsText, v.StringValue())
				}
			}

			poll := Poll{
				Options: make(map[string][]string),
			}
			pollButtons := []discordgo.MessageComponent{}
			for i := 0; i < len(pollAnsText); i++ {
				spl := strings.Split(pollAnsText[i], ";")
				emojiStr, labelStr := spl[0], spl[1]
				if len(spl) < 2 {
					log.Printf("Formatting of option #%d failed: %s", i, pollAnsText[i])
					return
				}

				emoji := emojiComponentFromString(emojiStr)

				btn := discordgo.Button{
					CustomID: "option_" + emoji.Name + "_" + labelStr,
					Label:    labelStr,
					Emoji:    emoji,
					Style:    discordgo.SecondaryButton,
				}

				poll.Options[btn.CustomID] = []string{}

				pollButtons = append(pollButtons, btn)
			}
			err := s.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: opt.Options[0].StringValue(),
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: pollButtons,
						},
					},
				},
			})
			if err != nil {
				log.Printf("Attemped to respond and failed: %v", err)
				return
			}

			msg, err := s.InteractionResponse(intr.Interaction)
			if err != nil {
				log.Printf("Attemped to get message and failed: %v", err)
				return
			}
			polls[msg.ID] = poll
		},
	}
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

			var pollStr string
			voteCustomID := intr.MessageComponentData().CustomID
			if poll, ok := polls[intr.Message.ID]; ok {
				err = poll.castVote(intr.Member.User.ID, voteCustomID)
				if err != nil {
					log.Printf("Error casting vote for poll %s: %v", intr.Message.ID, err)
				}

				pollStr = fmt.Sprintf("%+v", poll)
			}

			_, err := s.ChannelMessageEdit(intr.ChannelID, intr.Message.ID, pollStr)
			if err != nil {
				log.Printf("Attemped to respond and failed: %v", err)
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
