package types

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
)

type InputStruct interface {
	InitByUser()
}

type Sticker struct {
	FileID string `json:"file_id"`
}

type SendText struct {
	Chat_ID int64  `json:"chat_id"`
	Text    string `json:"text"`
}

func (S *SendText) InitByUser() {
	S.Chat_ID = readInputInt64("Type in ChatID")
	S.Text = readInput("Type in Text")
}

type SendSticker struct {
	Chat_ID int64  `json:"chat_id"`
	Sticker string `json:"sticker"`
}

func (S *SendSticker) InitByUser() {
	S.Chat_ID = readInputInt64("Type in ChatID")
	S.Sticker = readInput("Type in Sticker")
}

type DeleteMessage struct {
	Chat_ID   int64 `json:"chat_id"`
	MessageID int64 `json:"message_id"`
}

func (D *DeleteMessage) InitByUser() {
	D.Chat_ID = readInputInt64("Type in ChatID")
	D.MessageID = readInputInt64("Type in MessageID (if you are using DeleteLastMessage method just type in anything or press Enter)")
}

type GetUpdate struct {
	Offset  int64 `json:"offset"`
	Timeout int64 `json:"timeout"`
}

func (G *GetUpdate) InitByUser() {}

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
