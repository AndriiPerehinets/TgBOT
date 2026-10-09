package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"github.com/AndriiPerehinets/TgBOT/internal/bot"
	"github.com/AndriiPerehinets/TgBOT/internal/cli"

	"github.com/joho/godotenv"
)

func main() {
	botToken, dbPassword, maxGoroutines := getEnv()

	ctx, _ := context.WithCancel(context.Background())

	ctx, _ = signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)

	run(ctx, botToken, dbPassword, maxGoroutines)
}

func run(ctx context.Context, botToken, dbPassword string, maxGoroutines int) {
	var wg sync.WaitGroup
	bot := bot.NewBot(botToken, dbPassword)

	wg.Add(3)
	go func() {
		defer wg.Done()
		cli.RunCMD(ctx, bot)
	}()

	go func() {
		defer wg.Done()
		bot.Fetch(ctx, maxGoroutines)
	}()

	go func() {
		defer wg.Done()
		bot.DeleteOldMessages(ctx)
	}()

	wg.Wait()
}

func getEnv() (botToken, dbPassword string, maxGoroutines int) {
	logger := log.New(os.Stdout, "Main Log:\t", log.LstdFlags|log.Llongfile)

	err := godotenv.Load()
	if err != nil {
		logger.Fatalln("Can't find file .env, ", err)
	}

	botToken = os.Getenv("TG_BOT_TOKEN")
	if botToken == "" {
		logger.Fatalln("TG_BOT_TOKEN is uninitialized inside .env file")
	}

	dbPassword = os.Getenv("POSTGRES_PASSWORD")
	if dbPassword == "" {
		logger.Fatalln("POSTGRES_PASSWORD is uninitialized inside .env file")
	}

	maxG := os.Getenv("MAX_GOROUTINES")
	if maxG == "" {
		logger.Fatalln("MAX_GOROUTINES is uninitialized inside .env file")
	}
	if maxGoroutines, err = strconv.Atoi(maxG); err != nil {
		logger.Fatalln("Can't parse maxGoroutines to int type, maxGoroutines must be a number %w", err)
	}

	return botToken, dbPassword, maxGoroutines
}
