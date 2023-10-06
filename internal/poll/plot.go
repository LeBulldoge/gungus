package poll

import (
	"sort"
	"strconv"
	"strings"
)

var (
	empty = "⬛"
	full  = "🔲"
)

func PlotBarChart(title string, values map[string]int) string {
	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteRune('\n')

	labels := make([]string, 0, len(values)/2)
	total := 0
	for k, v := range values {
		total += v
		labels = append(labels, k)
	}

	sort.Strings(labels)
	for _, s := range labels {
		label := strings.Split(s, "_")[2]
		value := values[s]

		sb.WriteString(label + " ")
		res := (float64(value) / float64(total)) * 10
		for i := 0; i < 10; i++ {
			if i < int(res) {
				sb.WriteString(full)
			} else {
				sb.WriteString(empty)
			}
		}

		sb.WriteRune(' ')
		sb.WriteString(strconv.Itoa(value))
		sb.WriteRune('/')
		sb.WriteString(strconv.Itoa(total))
		sb.WriteRune('\n')
	}

	return sb.String()
}
