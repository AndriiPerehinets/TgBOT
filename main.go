package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"sv/start"
	"syscall"
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
	token := flag.String("bot-token", "", "Tlegram bot API token")

	flag.Parse()

	if *token == "" {
		logger.Fatal("Token wasn't passed")
	}

	logger.Print("Token accepted")

	logger.Println("Passed token is", *token)

	return *token
}
