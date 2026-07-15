package start

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sv/bot"
	"sv/types"
)

func RunCMD(bot *bot.Bot) {

	buf := bufio.NewReader(os.Stdin)

	logger := log.New(os.Stdout, "RunCMD func Log:\t", log.LstdFlags|log.Llongfile)

	for {
		command, err := buf.ReadString('\n')
		if err != nil {
			logger.Println(fmt.Errorf("Can't read the input from Stdin %w", err))
		}

		command = strings.TrimSpace(strings.ToLower(command))
		if command == "end" {
			logger.Fatal("Program ended due to end command execution")
		}

		com, exist := CommandBuilder[command]
		if exist == false {
			logger.Println("Invalid command")
			continue
		}

		param := com()

		err = bot.DoCMDCommand(command, param)
		if err != nil {
			logger.Println(fmt.Errorf("During command execution occurred an error:%w", err))
		}
	}
}

var CommandBuilder = map[string]func() types.InputStruct{
	"sendmessage": func() types.InputStruct {
		return &types.SendText{
			Chat_ID: readInputInt64("Type in ChatID"),
			Text:    readInput("Type in Text"),
		}
	},
	"sendsticker": func() types.InputStruct {
		return &types.SendSticker{
			Chat_ID: readInputInt64("Type in ChatID"),
			Sticker: readInput("Type in Sticker"),
		}
	},
	"deletemessage": func() types.InputStruct {
		return &types.DeleteMessage{
			Chat_ID:   readInputInt64("Type in ChatID"),
			MessageID: readInputInt64("Type in MessageID (if you are using DeleteLastMessage method just type in anything or press Enter)"),
		}
	},
	"deletelastmessage": func() types.InputStruct {
		return &types.DeleteMessage{
			Chat_ID:   readInputInt64("Type in ChatID"),
			MessageID: readInputInt64("Type in MessageID (if you are using DeleteLastMessage method just type in anything or press Enter)"),
		}
	},
}

func readInput(prompt string) string {
	log.Println(prompt)

	reader := bufio.NewReader(os.Stdin)

	input, _ := reader.ReadString('\n')

	input = strings.TrimSpace(input)

	return input
}

func readInputInt64(prompt string) int64 {
	for {
		res := readInput(prompt)

		number, err := strconv.ParseInt(res, 10, 64)
		if err != nil {
			log.Println("type in a number")
			continue
		}

		return number
	}
}
