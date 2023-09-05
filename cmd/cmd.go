package gungus

import (
	"flag"
	"log/slog"
	"os"
	"os/signal"

	"github.com/LeBulldoge/gungus/internal/discord"
)

// flags
var (
	botToken = flag.String("token", "", "Bot token")
)

func Run() {
	flag.Parse()

	bot, err := discord.NewBot(*botToken)
	if err != nil {
		slog.Error("error while creating session: %v", err)
	}

	err = discord.OpenConnection(bot)
	if err != nil {
		slog.Error("error while opening session: %v", err)
		return
	}

	err = discord.CreateCommands(bot)
	if err != nil {
		slog.Error("error while creating commands: %v", err)
		discord.Shutdown(bot)
		return
	}

	defer discord.Shutdown(bot)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	slog.Info("Press Ctrl+C to exit")
	<-stop
}
