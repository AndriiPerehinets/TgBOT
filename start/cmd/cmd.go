package start

import (
	"bufio"
	"fmt"
	"log"
	"os"
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

		param, exist := Comand_To_Type_Map[command]
		if exist == false {
			logger.Println(fmt.Errorf("Invalid command"))
			continue
		}

		param.InitByUser()

		err = bot.DoCMDCommand(command, param)
		if err != nil {
			logger.Println(fmt.Errorf("During command execution occurred an error:%w", err))
		}
	}
}

var Comand_To_Type_Map = map[string]types.InputStruct{
	"sendmessage":       &types.SendText{},
	"sendsticker":       &types.SendSticker{},
	"deletemessage":     &types.DeleteMessage{},
	"deletelastmessage": &types.DeleteMessage{},
}
