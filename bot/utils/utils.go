package utils

import (
	"errors"
	"fmt"
	"sv/types"
)

func ExecuteRollBack(actions ...func() error) (RollBackErr error) {
	for _, action := range actions {
		err := action()
		RollBackErr = errors.Join(RollBackErr, err)
	}
	return RollBackErr
}

func LogMessage(message *types.Message) string {
	return fmt.Sprintf("ChatID: %d, UserID; %d, Username: %s, Text: %s, Sticker: %s ", message.Chat.ID, message.From.UserID,
		message.From.Username, message.Text, message.Sticker.FileID)
}
