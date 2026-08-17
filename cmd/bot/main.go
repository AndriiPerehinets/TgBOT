package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/AndriiPerehinets/TgBOT/internal/bot"
	"github.com/AndriiPerehinets/TgBOT/internal/cli"

	"github.com/joho/godotenv"
)

func main() {
	Token := getToken()

	run(Token)

	stop := make(chan os.Signal, 1)

	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop
}

func run(Token string) {
	bot := bot.NewBot(Token)

	go cli.RunCMD(bot)

	go bot.Fetch()

	ctx, _ := context.WithCancel(context.Background())
	go bot.DeleteOldMessages(ctx)
}

func getToken() string {
	logger := log.New(os.Stdout, "Main Log:\t", log.LstdFlags|log.Llongfile)

	err := godotenv.Load()
	if err != nil {
		logger.Fatalln("Can't find file .env, ", err)
	}

	botToken := os.Getenv("TG_BOT_TOKEN")
	if botToken == "" {
		logger.Fatalln("TG_BOT_TOKEN is uninitialized inside .env file")
	}

	return botToken
}
