package utils

import (
	"errors"
)

var ErrTriggerExists = errors.New("Such trigger already exists")
var ErrUserNotified = errors.New("User already nitified")
var ErrTriggerDontExists = errors.New("There is no such trigger in this chat")

func ExecuteRollBack(actions ...func() error) (RollBackErr error) {
	for _, action := range actions {
		err := action()
		RollBackErr = errors.Join(RollBackErr, err)
	}
	return RollBackErr
}
