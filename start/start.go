package start

import (
	"context"
	"sv/bot"
	start "sv/start/cmd"
)

func Start(Token string) {
	bot := bot.NewBot(Token)

	go start.RunCMD(bot)

	go bot.Fetch()

	ctx, _ := context.WithCancel(context.Background())
	go bot.DeleteOldMessages(ctx)
}
