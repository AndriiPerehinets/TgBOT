package start

import (
	"sv/bot"
	start "sv/start/cmd"
)

func Start(Token string) {
	bot := bot.NewBot(Token)

	go start.RunCMD(bot)

	go bot.Fetch()
}
