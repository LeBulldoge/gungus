package gungus

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"

	"github.com/LeBulldoge/gungus/internal/database"
	"github.com/LeBulldoge/gungus/internal/discord"
)

// flags
var (
	configDir = flag.String("config", "", "Config directory")
	botToken  = flag.String("token", "", "Bot token")
)

func Run() {
	flag.Parse()

	slog.Info("starting bot...")

	storage := database.New(*configDir)
	err := storage.Open(context.TODO())
	if err != nil {
		slog.Error("error while opening database", "err", err)
	}

	bot, err := discord.NewBot(*botToken, storage)
	if err != nil {
		slog.Error("error while creating session: %v", err)
	}

	err = bot.OpenConnection()
	if err != nil {
		slog.Error("error while opening session: %v", err)
		return
	}

	err = bot.CreateCommands()
	if err != nil {
		slog.Error("error while creating commands: %v", err)
		bot.Shutdown()
		return
	}

	defer bot.Shutdown()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	slog.Info("Press Ctrl+C to exit")
	<-stop
}
