package main

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
	"time"

	"github.com/benoitmasson/plotters/piechart"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
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

func plotPieChart(name string, values map[string]int) (string, error) {
	p := plot.New()
	p.HideAxes()

	pval := plotter.Values{}
	for _, value := range values {
		pval = append(pval, float64(value))
	}

	pie, err := piechart.NewPieChart(pval)
	if err != nil {
		panic(err)
	}

	for label := range values {
		pie.Labels.Nominal = append(pie.Labels.Nominal, label)
	}
	pie.Labels.Values.Show = true
	pie.Labels.Values.Percentage = true

	pie.Color = color.RGBA{255, 0, 0, 255}

	p.Add(pie)

	filename := fmt.Sprintf("%s_%d.png", name, time.Now().Unix())

	return filename, p.Save(4*vg.Inch, 4*vg.Inch, filename)
}
