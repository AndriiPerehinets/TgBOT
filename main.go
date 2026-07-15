package main

import (
	"log"
	"os"
	"os/signal"
	"sv/start"
	"syscall"

	"github.com/joho/godotenv"
)

func main() {
	Token := getToken()

	start.Start(Token)

	stop := make(chan os.Signal, 1)

	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop
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
