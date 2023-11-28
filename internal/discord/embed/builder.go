package embed

import (
	"github.com/bwmarrin/discordgo"
)

type embedBuilder struct {
	*discordgo.MessageEmbed
}

func NewEmbed() *embedBuilder {
	return &embedBuilder{
		&discordgo.MessageEmbed{},
	}
}

func (b *embedBuilder) SetTitle(title string) *embedBuilder {
	b.Title = title

	return b
}

func (b *embedBuilder) SetUrl(url string) *embedBuilder {
	b.URL = url

	return b
}

func (b *embedBuilder) SetDescription(desc string) *embedBuilder {
	b.Description = desc

	return b
}

func (b *embedBuilder) AddField(name string, value string) *embedBuilder {
	b.Fields = append(b.Fields, &discordgo.MessageEmbedField{
		Name:   name,
		Value:  value,
		Inline: false,
	})

	return b
}

func (b *embedBuilder) AddInlineField(name string, value string) *embedBuilder {
	b.Fields = append(b.Fields, &discordgo.MessageEmbedField{
		Name:   name,
		Value:  value,
		Inline: true,
	})

	return b
}

func (b *embedBuilder) SetFooter(text string, iconURL string) *embedBuilder {
	b.Footer = &discordgo.MessageEmbedFooter{
		Text:         text,
		IconURL:      iconURL,
		ProxyIconURL: iconURL,
	}

	return b
}

func (b *embedBuilder) SetImage(url string) *embedBuilder {
	b.Image = &discordgo.MessageEmbedImage{
		URL: url,
	}

	return b
}

func (b *embedBuilder) SetThumbnail(url string) *embedBuilder {
	b.Thumbnail = &discordgo.MessageEmbedThumbnail{
		URL: url,
	}

	return b
}

func (b *embedBuilder) SetAuthor(name string) *embedBuilder {
	b.Author = &discordgo.MessageEmbedAuthor{
		Name: name,
	}

	return b
}

func (b *embedBuilder) SetTimestamp(timestamp string) *embedBuilder {
	b.Timestamp = timestamp

	return b
}
