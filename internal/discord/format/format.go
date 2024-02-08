package format

import (
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func TimeToTimestamp(t time.Time) string {
	var sb strings.Builder
	sb.WriteString("<t:")
	sb.WriteString(strconv.FormatInt(t.Unix(), 10))
	sb.WriteRune('>')

	return sb.String()
}

func IsCustomEmoji(s string) bool {
	return s[0] == '<'
}

func EmojiComponentFromString(s string) *discordgo.ComponentEmoji {
	emoji := discordgo.ComponentEmoji{}
	if IsCustomEmoji(s) {
		s = s[1 : len(s)-2]
		parts := strings.Split(s, ":")

		emoji.Animated = parts[0] == "a"
		emoji.Name = parts[1]
		emoji.ID = parts[2]
	} else {
		emoji.Name = strings.Trim(s, " ")
	}

	return &emoji
}

func GetMemberDisplayName(member *discordgo.Member) string {
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
