package discord

import (
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func timeToTimestamp(t time.Time) string {
	var sb strings.Builder
	sb.WriteString("<t:")
	sb.WriteString(strconv.FormatInt(t.Unix(), 10))
	sb.WriteRune('>')

	return sb.String()
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
