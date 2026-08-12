package utils

import (
	"errors"
)

var ErrTriggerExists = errors.New("Such trigger already exists")
var ErrUserNotified = errors.New("User already nitified")
var ErrTriggerDontExists = errors.New("There is no such trigger in this chat")

func TrancateText(text string, maxLen int) string {
	if len(text) > maxLen {
		runes := []rune(text)
		return string(runes[:maxLen-3]) + "..."
	}
	return text
}
