package bot

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sv/bot/client"
	"sv/bot/storage"
	"sv/bot/utils"
	"sv/types"
)

type Bot struct {
	ID       int64
	UserName string
	Client   *client.Client
	Logger   *log.Logger
	Storage  *storage.Storage
}

func NewBot(Token string) *Bot {
	bot := &Bot{
		Client: client.NewClient(Token),
		Logger: log.New(os.Stdout, "Bot log:\t", log.Lshortfile|log.LstdFlags),
		Storage: func() *storage.Storage {
			db := storage.SetUpStorage()
			storage := storage.NewStorage(db)
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

func (b *Bot) DoCMDCommand(command string, param types.InputStruct) error {
	var MethodsList = map[string]func() error{
		"sendmessage": func() error { return b.SendStruct(command, param) },
		"sendsticker": func() error { return b.SendStruct(command, param) },

		"deletemessage": func() error {
			param, ok := param.(*types.DeleteMessage)
			if !ok {
				return fmt.Errorf("Can't delete message, type of param should be types.DeleteMessage")
			}

			return b.DeleteMessage(param)
		},

		"deletelastmessage": func() error {
			param, ok := param.(*types.DeleteMessage)
			if !ok {
				return fmt.Errorf("Can't delete message, type of param should be types.DeleteMessage")
			}

			return b.DeleteLastBotsMessage(param.Chat_ID)
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

func (b *Bot) Fetch() {
	var offset int64

	for {
		updates, err := b.Client.GetUpdate(offset)
		if err != nil {
			b.Logger.Println("An error occurred during b.Client.GetUpdate: ", err)
			continue
		}

		// ch := make(chan types.Update, 1000)

		for _, u := range updates {
			if u.UpdateID >= offset {
				offset = u.UpdateID + 1
			}

			// go process.CreateWorkerPool(50, ch)

			err = b.Storage.InsertMessage(&u.Message)
			if err != nil {
				b.Logger.Println(fmt.Errorf("Can't insert message %s: %w", utils.LogMessage(&u.Message), err))
				continue
			}

			command, ok := b.IsCommand(&u.Message)
			if ok {
				b.Logger.Println("Message is a command: ", u.Message.Text)
				err := command()
				if err != nil {
					b.Logger.Println("Can't execute user command: ", err)
					continue
				}
				continue
			}

			IsTrigger, err := b.Storage.IsTrigger(&u.Message)
			if err != nil {
				b.Logger.Println(err)
				erro := b.SendText(&u.Message, "Sorry, an error occured")
				if erro != nil {
					b.Logger.Println(err, erro)
					continue
				}
				continue
			} else if IsTrigger {
				TriggerResp, IsSricker, err := b.Storage.GetTriggerResp(&u.Message)
				if err != nil {
					b.Logger.Println("Error during Trigger execution:", err)
					err := b.SendText(&u.Message, "Sorry, an error occured")
					if err != nil {
						b.Logger.Println(err)
						continue
					}
					continue
				} else if IsSricker {
					err := b.SendSticker(&u.Message, TriggerResp)
					if err != nil {
						b.Logger.Println("Can't send responce to the trigger: ", err)
						continue
					}
					continue
				}
				err = b.SendText(&u.Message, TriggerResp)
				if err != nil {
					b.Logger.Println("Can't send responce to the trigger: ", err)
					continue
				}
				b.Logger.Println("Bot responded to the trigger")
				continue
			}

			expected, err := b.Storage.IsExpected(&u.Message)
			if err != nil {
				b.Logger.Println(err)
				err := b.SendText(&u.Message, "Sorry, an error occured")
				if err != nil {
					b.Logger.Println(fmt.Errorf("Error during message %s handling: %w", utils.LogMessage(&u.Message), err))
					continue
				}
				continue
			}

			if expected {
				State, err := b.Storage.GetExpectedMessageState(&u.Message)
				if err != nil {
					err = errors.Join(err, utils.ExecuteRollBack(
						func() error { return b.Storage.DeleteExpectedMessage(&u.Message) },
						func() error { return b.SendText(&u.Message, "Sorry, an error occured") },
					))
					b.Logger.Println(fmt.Errorf("Error during message %s handling: %w", utils.LogMessage(&u.Message), err))
					continue
				}

				switch State {
				case "Trigger":
					err := b.VerifyType(&u.Message)
					if err != nil {
						b.Logger.Println(fmt.Errorf("Can't add trigger. Can't VerifyType of message %s: %w", utils.LogMessage(&u.Message), err))
						continue
					}
					err = b.Storage.InsertTrigger(&u.Message)
					if err != nil {
						if err.Error() == "Such trigger already exists" {
							err = errors.Join(err, b.SendText(&u.Message, err.Error()))
							b.Logger.Println(fmt.Errorf("Can't add trigger. Error during message %s handling: %w", utils.LogMessage(&u.Message), err))
							continue
						}

						err = errors.Join(err, b.SendText(&u.Message, "Sorry, an error occured"))

						b.Logger.Println(fmt.Errorf("Can't add trigger. Error during message %s handling: %w", utils.LogMessage(&u.Message), err))
						continue
					}

					err = b.SendText(&u.Message, "Now send a response to the trigger")
					if err != nil {
						err = errors.Join(err, utils.ExecuteRollBack(
							func() error { return b.Storage.DeleteTrigger(&u.Message, false) },
							func() error { return b.Storage.DeleteExpectedMessage(&u.Message) },
							func() error { return b.SendText(&u.Message, "Sorry, an error occured") },
						))
						b.Logger.Println(fmt.Errorf("Can't add trigger. Error during message %s handling: %w", utils.LogMessage(&u.Message), err))
						continue
					}
					continue

				case "TriggerResp":
					err := b.VerifyType(&u.Message)
					if err != nil {
						err = errors.Join(err, b.Storage.DeleteTrigger(&u.Message, false))
						b.Logger.Println(fmt.Errorf("Can't set trigger response. Can't VerifyType of message %s,: %w", utils.LogMessage(&u.Message), err))
						continue
					}
					err = b.Storage.AddTriggerResponse(&u.Message)
					if err != nil {
						err = errors.Join(err, utils.ExecuteRollBack(
							func() error { return b.Storage.DeleteTrigger(&u.Message, false) },
							func() error { return b.SendText(&u.Message, "Sorry, an error occured") },
						))
						b.Logger.Println(fmt.Errorf("Can't set trigger response. Error during message %s handling: %w", utils.LogMessage(&u.Message), err))
						continue
					}

					err = b.SendText(&u.Message, "Trigger saved successfully")
					if err != nil {
						b.Logger.Println(fmt.Errorf("Can't send a submission message during message %s handling: %w", utils.LogMessage(&u.Message), err))
						continue
					}
					continue

					// case "triggername":
					// 	err := b.VerifyType(&u.Message)
					// 	if err != nil {
					// 		b.Logger.Println(fmt.Errorf("Can't VerifyType of message %#v that user send as triggername for deletion: %w", u.Message, err))
					// 		continue
					// 	}
					// 	b.DeleteTrigger(&u.Message)
				}
			}

			continue
		}
	}
}

func (b *Bot) VerifyType(message *types.Message) error {
	if fmt.Sprint(message.Text+message.Sticker.FileID) == "" {
		err := b.SendText(message, "Trigger must be a text or sticker, if you still want to create a trigger use command again")
		if err != nil {
			return fmt.Errorf("Can't execute VerifyType %w", err)
		}
		err = b.Storage.DeleteExpectedMessage(message)
		if err != nil {
			err = errors.Join(err, b.SendText(message, "Sorry, an error occured"))
			return fmt.Errorf("User send message of invalid type, error during deletion message from EXPECTED_MESSAGE table: %w", err)
		}
		return fmt.Errorf("User send message of invalid type")
	}

	return nil
}

func (b *Bot) IsCommand(Message *types.Message) (commad func() error, ok bool) {
	var CommandList = map[string]func() error{
		"addtrigger": func() error { return b.AddExpectedTrigger(Message) },
		// "deletetrigger": func() error { return b.DeleteTrigger(Message) },
	}

	txt := strings.Split(strings.TrimPrefix(strings.TrimSpace(strings.ToLower(Message.Text)), "/"), "@")[0]
	c, ok := CommandList[txt]
	if !ok {
		return nil, ok
	}

	return c, ok
}

func (b *Bot) SendText(message *types.Message, text string) error {
	txt := &types.SendText{
		Chat_ID: message.Chat.ID,
		Text:    text,
	}
	mes, err := b.Client.Send("sendMessage", txt)
	if err != nil {
		return fmt.Errorf("Can't send message: %#v, %w", txt, err)
	}

	err = b.Storage.InsertMessage(mes)
	if err != nil {
		return fmt.Errorf("Can't save message: %#v %w", txt, err)
	}

	return nil
}

func (b *Bot) SendSticker(message *types.Message, StickerID string) error {
	stic := &types.SendSticker{
		Chat_ID: message.Chat.ID,
		Sticker: StickerID,
	}

	mes, err := b.Client.Send("sendSticker", stic)
	if err != nil {
		return fmt.Errorf("Can't send sticker: %#v, %w", stic, err)
	}

	err = b.Storage.InsertMessage(mes)
	if err != nil {
		return fmt.Errorf("Can't send sticker: %#v, %w", stic, err)
	}

	return nil
}

func (b *Bot) SendStruct(command string, param types.InputStruct) error {
	mes, err := b.Client.Send(command, param)
	if err != nil {
		return fmt.Errorf("Can't send struct: %#v, %w", param, err)
	}

	err = b.Storage.InsertMessage(mes)
	if err != nil {
		return fmt.Errorf("Can't send struct: %#v, %w", param, err)
	}

	return nil
}

func (b *Bot) DeleteMessage(param *types.DeleteMessage) error {
	err := b.Client.DeleteMessage(param)
	if err != nil {
		return fmt.Errorf("Can't delete message: %#v, %w", param, err)
	}

	err = b.Storage.UpdateMessageStatus(param)
	if err != nil {
		return fmt.Errorf("Can't delete message: %#v, %w", param, err)
	}

	return nil
}

func (b *Bot) DeleteLastBotsMessage(chatID int64) error {
	del, err := b.Storage.SelectLastMessage(chatID, b.ID)
	if err != nil {
		return fmt.Errorf("Can't delete last message: ChatID:%d, %w", chatID, err)
	}

	err = utils.ExecuteRollBack(
		func() error { return b.Client.DeleteMessage(del) },
		func() error { return b.Storage.UpdateMessageStatus(del) },
	)
	if err != nil {
		return fmt.Errorf("Can't delete last message: ChatID:%d, %w", chatID, err)
	}

	return nil
}

func (b *Bot) AddExpectedTrigger(message *types.Message) error {
	err := b.SendText(message, "Type in trigger phrase")
	if err != nil {
		return fmt.Errorf("Can't execute AddExpectedTrigger %w", err)
	}

	err = b.Storage.InsertExpectedMessage(message, "Trigger")
	if err != nil {
		err = errors.Join(err, b.SendText(message, "Sorry, an error occured"))

		return fmt.Errorf("Can't execute AddExpectedTrigger: %w", err)
	}

	return nil
}

// func (b *Bot) DeleteTrigger(message *types.Message) error {
// 	err := b.SendText(message, "Type in the trigger that you want to delete")
// 	if err != nil {
// 		return fmt.Errorf("Can't execute DeleteTrigger, %w", err)
// 	}
// 	// b.Storage.InsertExpectedMessage(message, "TriggerName")
// 	// IsAdmin, err := b.Client.IsAdministrator(message.Chat.ID, message.From.UserID)
// 	// if err != nil {
// 	// 	erro := b.SendText(message, "Sorry, an error occured")
// 	// if erro != nil {
// 	// 	return fmt.Errorf("Can't DeleteTrigger: %w, %w", err, erro)
// 	// }
// 	//	return fmt.Errorf("Can't DeleteTrigger: %w", err)
// 	// }

// 	// err = b.Storage.DeleteTrigger(message, IsAdmin)
// 	// if err != nil {
// 	// 	erro := b.SendText(message, "Sorry, an error occured")
// 	// 	if erro != nil {
// 	// 		return fmt.Errorf("Can't DeleteTrigger: %w, %w", err, erro)
// 	// 	}
// 	// 	return fmt.Errorf("Can't DeleteTrigger: %w", err)
// 	// }
// }
