package discord

import (
	"strconv"
	"strings"
	"time"
)

func timeToTimestamp(t time.Time) string {
	var sb strings.Builder
	sb.WriteString("<t:")
	sb.WriteString(strconv.FormatInt(t.Unix(), 10))
	sb.WriteRune('>')

	return sb.String()
}
