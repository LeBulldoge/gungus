package main

import (
	"strconv"
	"strings"
)

var (
	empty = "⬛"
	full  = "🔲"
)

func plotBarChart(title string, values map[string]int) string {
	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteRune('\n')

	total := 0
	for _, v := range values {
		total += v
	}

	for k, v := range values {
		sb.WriteString(k + " ")
		res := (float64(v) / float64(total)) * 10
		for i := 0; i < 10; i++ {
			if i < int(res) {
				sb.WriteString(full)
			} else {
				sb.WriteString(empty)
			}
		}

		sb.WriteRune(' ')
		sb.WriteString(strconv.Itoa(v))
		sb.WriteRune('/')
		sb.WriteString(strconv.Itoa(total))
		sb.WriteRune('\n')
	}

	return sb.String()
}
