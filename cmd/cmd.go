package gungus

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"

	"github.com/LeBulldoge/gungus/internal/database"
	"github.com/LeBulldoge/gungus/internal/discord"
	gos "github.com/LeBulldoge/gungus/internal/os"
)

// flags
var (
	configDir = flag.String("config", "", "Config directory")
	botToken  = flag.String("token", "", "Bot token")
)

func Run(version string, build string) {
	flag.Parse()

	slog.Info("starting gungus", "version", version, "build", build)

	if len(*botToken) == 0 {
		slog.Error("Please provide a discord bot authentication token.\nMore info at https://discord.com/developers/docs/getting-started")
		return
	}

	if len(*configDir) > 0 {
		gos.SetCustomConfigDir(*configDir)
	}

	storage := database.New(gos.ConfigPath())
	err := storage.Open(context.TODO())
	if err != nil {
		slog.Error("error while opening database", "err", err)
		return
	}

	bot, err := discord.StartBot(*botToken, storage)
	if err != nil {
		slog.Error("error while starting bot", "err", err)
		return
	}
	defer bot.Shutdown()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	slog.Info("Press Ctrl+C to exit")
	<-stop
}
