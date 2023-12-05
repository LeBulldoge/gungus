package play

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/LeBulldoge/gungus/internal/discord/embed"
	"github.com/LeBulldoge/gungus/internal/discord/format"
	"github.com/LeBulldoge/gungus/internal/youtube"
	"github.com/bwmarrin/discordgo"
)

func (c *Command) HandleQueue(sesh *discordgo.Session, intr *discordgo.InteractionCreate) {
	guildID := intr.GuildID

	var queue []youtube.Video
	if ps := c.playbackManager.Get(guildID); ps != nil {
		queue = ps.Queue()
	}
	if len(queue) < 1 {
		format.DisplayInteractionError(sesh, intr, "There is nothing in the queue.")
		return
	}

	queueLength := len(queue)

	opt := intr.ApplicationCommandData().Options

	totalNumOfFields := 1
	if len(opt) > 0 {
		totalNumOfFields = int(opt[0].IntValue())
	}

	currentVideo := queue[0]
	embed := embed.NewEmbed().
		SetAuthor("Currently playing").
		SetTitle(currentVideo.Title).
		SetThumbnail(currentVideo.Thumbnail).
		SetUrl(currentVideo.GetShortURL()).
		SetDescription(currentVideo.Length).
		SetFooter("Total count: "+strconv.Itoa(queueLength), "").
		SetTimestamp(time.Now().Format(time.RFC3339))

	fieldStart := 1
	fieldEnd := 10

	if queueLength > 1 {
		embed.AddField("In queue", "")

		var sb strings.Builder
		maxLineLen := 1024 / 10
		// max length of the formatting string below
		fmtStringLen := 5 + 2 + 2 + 17 + 11 + 3 + 2 + 8 + 1
		maxTitleLen := maxLineLen - fmtStringLen

		fieldNum := 0
		for fieldNum < totalNumOfFields && fieldStart < queueLength {
			fieldNum++
			if fieldEnd > queueLength {
				fieldEnd = queueLength
			}

			for i, video := range queue[fieldStart:fieldEnd] {
				titleLen := len(video.Title)
				if titleLen > maxTitleLen {
					video.Title = video.Title[:maxTitleLen-3] + "..."
				}
				fmt.Fprintf(&sb, "%d: [%s](%s) - (%s)\n", fieldStart+i+1, video.Title, video.GetShortURL(), video.Length)
			}

			embed.AddField("", sb.String())
			fieldStart = fieldEnd
			fieldEnd += 10

			sb.Reset()
		}
	}

	err := sesh.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed.MessageEmbed},
		},
	})
	if err != nil {
		c.logger.Error("failure responding to interaction", "err", err)
		format.DisplayInteractionError(sesh, intr, "Failure responding to interaction. See the log for details.")
	}
}
