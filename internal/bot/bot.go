package bot

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/AndriiPerehinets/TgBOT/internal/storage"
	"github.com/AndriiPerehinets/TgBOT/internal/telegram"
	"github.com/AndriiPerehinets/TgBOT/internal/telegram/client"
	"github.com/AndriiPerehinets/TgBOT/internal/utils"
)

type Bot struct {
	ID       int64
	UserName string
	Client   *client.Client
	Logger   *log.Logger
	Storage  *storage.Storage
}

func NewBot(botToken, dbPassword string) *Bot {
	bot := &Bot{
		Client: client.NewClient(botToken),
		Logger: log.New(os.Stdout, "Bot log:\t", log.Lshortfile|log.LstdFlags),
		Storage: func() *storage.Storage {
			storage := storage.NewStorage(storage.SetUpStorage(dbPassword))
			return storage
		}(),
	}

	U, err := bot.Client.GetMe()
	if err != nil {
		bot.Logger.Fatal("Can't execute GetMe to get info about bot: ", err)
	}

	bot.ID = U.UserID
	bot.UserName = U.Username

	err = bot.Client.SetCommands()
	if err != nil {
		bot.Logger.Println(err)
	}

	bot.Logger.Println(bot.ID, bot.UserName)
	return bot
}

func (b *Bot) Fetch(ctx context.Context, maxGoroutines int) {
	var offset int64
	pool := NewPool(b, maxGoroutines, 20)
	pool.CreateWorkerPool(ctx, maxGoroutines)

	for {
		select {
		case <-ctx.Done():
			for _, ch := range pool.taskChan {
				close(ch)
			}
			pool.wg.Wait()
			b.Logger.Println("All workers successfully stopped")
			return

		default:
			updates, err := b.Client.GetUpdate(offset)
			if err != nil {
				b.Logger.Println("An error occurred during b.Client.GetUpdate: ", err)
				continue
			}

			for _, u := range updates {
				if u.Message.MessageID == 0 {
					continue
				} else if u.UpdateID >= offset {
					offset = u.UpdateID + 1
				}

				chanIndex := u.Message.Chat.ID % int64(maxGoroutines)
				if chanIndex < 0 {
					chanIndex = -chanIndex
				}
				pool.taskChan[chanIndex] <- u
			}
		}
	}
}

func (b *Bot) DeleteOldMessages(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := b.Storage.DeleteOldMessages()
			if err != nil {
				b.Logger.Println(errors.Join(err, utils.ErrUserNotified))
			}
		case <-ctx.Done():
			b.Logger.Println("DeleteOldMessage was stopped by context")
			return
		}
	}
}

func (b *Bot) DoCMDCommand(command string, param telegram.InputStruct) error {
	var MethodsList = map[string]func() error{
		"sendmessage": func() error { return b.sendStruct(command, param) },
		"sendsticker": func() error { return b.sendStruct(command, param) },

		"deletemessage": func() error {
			param, ok := param.(*telegram.DeleteMessage)
			if !ok {
				return fmt.Errorf("Can't delete message, type of param should be types.DeleteMessage")
			}

			return b.deleteMessage(param)
		},

		"deletelastmessage": func() error {
			param, ok := param.(*telegram.DeleteMessage)
			if !ok {
				return fmt.Errorf("Can't delete message, type of param should be types.DeleteMessage")
			}

			return b.deleteLastBotsMessage(param.Chat_ID)
		},
	}

	meth, ok := MethodsList[command]
	if !ok {
		return fmt.Errorf("Method %s didn't exist inside bot.DoCMDCommand", command)
	}
	err := meth()
	if err != nil {
		return err
	}

	return nil
}

func (b *Bot) fetchMessage(message *telegram.Message) error {
	err := b.Storage.InsertMessage(message)
	if err != nil {
		return fmt.Errorf("Can't insert message: %w", err)
	}

	ok, err := b.expectedHandle(message)
	if err != nil {
		return err
	} else if ok {
		return nil
	}

	command, ok := b.isCommand(message)
	if ok {
		b.Logger.Println("Message is a command: ", message.Text)
		err := command(b, message)
		if err != nil {
			return fmt.Errorf("Can't execute user command: %w", err)
		}
		return nil
	}

	ok, err = b.triggerHandle(message)
	if err != nil {
		return err
	} else if ok {
		return nil
	}

	return nil
}

func (b *Bot) triggerHandle(message *telegram.Message) (bool, error) {
	message.Text = strings.ToLower(strings.TrimSpace(message.Text))
	IsTrigger, err := b.Storage.IsTrigger(message)
	if err != nil {
		return false, fmt.Errorf("Error during message handling %w", err)
	}
	if IsTrigger {
		b.Logger.Printf("Message is trigger\n")

		TriggerResp, RespType, err := b.Storage.GetTriggerResp(message)
		if err != nil {
			return false, fmt.Errorf("Error during Trigger execution: %w", err)
		}

		if RespType == "Sticker" {
			err := b.sendSticker(message, TriggerResp)
			if err != nil {
				return false, fmt.Errorf("Can't send response to the trigger: %w", err)
			}
			b.Logger.Println("Bot responded to the trigger")
			return true, nil
		}

		err = b.sendText(message, TriggerResp)
		if err != nil {
			return false, fmt.Errorf("Can't send responce to the trigger: %w", err)
		}
		b.Logger.Println("Bot responded to the trigger")
		return true, nil
	}
	return false, nil
}

func (b *Bot) expectedHandle(message *telegram.Message) (done bool, err error) {
	expected, err := b.Storage.IsExpected(message)
	if err != nil {
		return false, fmt.Errorf("Error during message handling: %w", err)
	}

	if expected {
		b.Logger.Println("Message is expected")
		State, err := b.Storage.GetExpectedMessageState(message)
		if err != nil {
			err = errors.Join(err, b.Storage.DeleteExpectedMessage(message))
			return false, fmt.Errorf("Error during message handling: %w", err)
		}

		switch State {
		case "Trigger":
			err = b.addTrigger(message)
			if err != nil {
				return false, err
			}
			return true, nil

		case "TriggerResp":
			err = b.addTriggerResp(message)
			if err != nil {
				return false, err
			}
			return true, nil

		case "TriggerName":
			err = b.deleteTrigger(message)
			if err != nil {
				return false, err
			}
			return true, nil
		}
	}
	return false, nil
}
