package cli

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/AndriiPerehinets/TgBOT/internal/bot"
	"github.com/AndriiPerehinets/TgBOT/internal/telegram"
)

func RunCMD(ctx context.Context, bot *bot.Bot) {
	logger := log.New(os.Stdout, "RunCMD func Log:\t", log.LstdFlags|log.Llongfile)

	inputChan := make(chan string)

	go func() {
		defer close(inputChan)
		for {
			buf := bufio.NewReader(os.Stdin)

			command, err := buf.ReadString('\n')

			if err != nil {
				logger.Println(fmt.Errorf("Can't read the input from Stdin %w", err))
			}

			select {
			case inputChan <- command:
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return

		case command := <-inputChan:

			command = strings.TrimSpace(strings.ToLower(command))

			com, exist := CommandBuilder[command]
			if exist == false {
				logger.Println("Invalid command")
				continue
			}

			param := com()

			err := bot.DoCMDCommand(command, param)
			if err != nil {
				logger.Println(fmt.Errorf("During %s execution occurred an error: %w", command, err))
			}
		}
	}
}

var CommandBuilder = map[string]func() telegram.InputStruct{
	"sendmessage": func() telegram.InputStruct {
		return &telegram.SendText{
			Chat_ID: readInputInt64("Type in ChatID"),
			Text:    readInput("Type in Text"),
		}
	},
	"sendsticker": func() telegram.InputStruct {
		return &telegram.SendSticker{
			Chat_ID:       readInputInt64("Type in ChatID"),
			StickerFileID: readInput("Type in StickerFileID"),
		}
	},
	"deletemessage": func() telegram.InputStruct {
		return &telegram.DeleteMessage{
			Chat_ID:   readInputInt64("Type in ChatID"),
			MessageID: readInputInt64("Type in MessageID (if you are using DeleteLastMessage method just type in anything or press Enter)"),
		}
	},
	"deletelastmessage": func() telegram.InputStruct {
		return &telegram.DeleteMessage{
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
