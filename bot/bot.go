package bot

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sv/bot/client"
	"sv/storage"
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

	bot.Logger.Println(bot.ID, bot.UserName)
	return bot
}

func (b *Bot) DoCMDCommand(command string, param types.InputStruct) error {
	var MethodsList = map[string]func(){
		"sendmessage": func() { b.SendStruct(command, param) },
		"sendsticker": func() { b.SendStruct(command, param) },

		"deletemessage": func() {
			param, ok := param.(*types.DeleteMessage)
			if !ok {
				b.Logger.Println("Can't delete message, type of param should be types.DeleteMessage")
				return
			}

			b.DeleteMessage(param)
		},

		"deletelastmessage": func() {
			param, ok := param.(*types.DeleteMessage)
			if !ok {
				b.Logger.Println("Can't delete message, type of param should be types.DeleteMessage")
				return
			}

			b.DeleteLastBotsMessage(param.Chat_ID)
		},
	}

	meth := MethodsList[command]
	meth()

	return nil
}

func (b *Bot) Fetch() {
	var offset int64

	for {
		updates, err := b.Client.GetUpdate(offset)
		if err != nil {
			b.Logger.Println(fmt.Errorf("An error occurred during b.Client.GetUpdate: %w", err))
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
				b.Logger.Println(err)
				continue
			}

			command, ok := b.IsCommand(&u.Message)
			if ok {
				b.Logger.Println("Message is a command: ", u.Message.Text)
				command()
				continue
			}

			IsTrigger, err := b.Storage.IsTrigger(&u.Message)
			if err != nil {
				b.Logger.Println(err)
				b.SendText(&u.Message, "Sorry, an error occured")
				continue
			} else if IsTrigger {
				TriggerResp, IsSricker, err := b.Storage.GetTriggerResp(&u.Message)
				if err != nil {
					b.Logger.Println(err)
					b.SendText(&u.Message, "Sorry, an error occured")
					continue
				} else if IsSricker {
					b.SendSticker(&u.Message, TriggerResp)
					continue
				}
				b.SendText(&u.Message, TriggerResp)
				b.Logger.Println("Bot responded to the trigger")
				continue
			}

			expected, err := b.Storage.IsExpected(&u.Message)
			if err != nil {
				b.Logger.Println(err)
				b.SendText(&u.Message, "Sorry, an error occured")
				continue
			}

			if expected {
				IsTrigger, IsTriggerResponse, err := b.Storage.GetExpectedMessageStatus(&u.Message)
				if err != nil {
					b.Logger.Println(err)
					b.SendText(&u.Message, "Sorry, an error occured")
					continue
				}

				if fmt.Sprint(u.Message.Text+u.Message.Sticker.FileID) == "" { //!!!!!!!!!!!!!!!!!!!!
					b.Logger.Println("User send invalid message type, trigger cannot be saves")
					err := b.Storage.DeleteTrigger(&u.Message)
					if err != nil {
						b.Logger.Println(err)
						b.SendText(&u.Message, "Sorry, an error occured")
					}

					b.SendText(&u.Message, "Trigger must be a text or sticker, if you still want to create a trigger use command again")
					continue
				}

				if IsTrigger {
					err := b.Storage.InsertTrigger(&u.Message)
					if err != nil {
						if err.Error() == "Such trigger already exists" {
							b.SendText(&u.Message, err.Error())
							b.Logger.Println(err)
							continue
						}
						b.Logger.Println(err)
						b.SendText(&u.Message, "Sorry, an error occured")
						continue
					}

					b.SendText(&u.Message, "Now send a reponce to the trigger")
					continue
				}

				if IsTriggerResponse {
					err := b.Storage.DeleteExpectedMessage(&u.Message)
					if err != nil {
						b.Logger.Println(err)
						b.SendText(&u.Message, "Sorry, an error occured")
					}
					err = b.Storage.AddTriggerResponse(&u.Message)
					if err != nil {
						b.Logger.Println(err)
						b.SendText(&u.Message, "Sorry, an error occured")
					}

					b.SendText(&u.Message, "Trigger saved successfully")
					continue
				}
			}

			continue

		}
	}
}

func (b *Bot) IsCommand(Message *types.Message) (commad func(), ok bool) {
	var CommandList = map[string](func()){
		"addtrigger": func() { b.AddExpectedTrigger(Message) },
	}

	Message.Text = strings.TrimPrefix(strings.TrimSpace(strings.ToLower(Message.Text)), "/")
	c, ok := CommandList[Message.Text]
	if !ok {
		return nil, ok
	}

	return c, ok
}

func (b *Bot) SendText(message *types.Message, text string) {
	txt := &types.SendText{
		Chat_ID: message.Chat.ID,
		Text:    text,
	}
	mes, err := b.Client.Send("sendMessage", txt)
	if err != nil {
		b.Logger.Println("Can't send message:\t", err)
		return
	}

	err = b.Storage.InsertMessage(mes)
	if err != nil {
		b.Logger.Println("Can't save message:\t", err)
		return
	}
}

func (b *Bot) SendSticker(message *types.Message, StickerID string) {
	stic := &types.SendSticker{
		Chat_ID: message.Chat.ID,
		Sticker: StickerID,
	}

	mes, err := b.Client.Send("sendMessage", stic)
	if err != nil {
		b.Logger.Println("Can't send sticker:\t", err)
		return
	}

	err = b.Storage.InsertMessage(mes)
	if err != nil {
		b.Logger.Println("Can't save sticker:\t", err)
		return
	}
}

func (b *Bot) SendStruct(command string, param types.InputStruct) {
	mes, err := b.Client.Send(command, param)
	if err != nil {
		b.Logger.Println("Can't send message:\t", err)
		return
	}

	err = b.Storage.InsertMessage(mes)
	if err != nil {
		b.Logger.Println("Can't save message:\t", err)
		return
	}
}

func (b *Bot) DeleteMessage(param *types.DeleteMessage) {
	err := b.Client.DeleteMessage(param)
	if err != nil {
		b.Logger.Println("Can't delete message:\t", err)
		return
	}

	err = b.Storage.UpdateMessageStatus(param)
	if err != nil {
		b.Logger.Println(err)
		return
	}
}

func (b *Bot) DeleteLastBotsMessage(chatID int64) {
	del, err := b.Storage.SelectLastMessage(chatID, b.ID)
	if err != nil {
		b.Logger.Println("Can't delete last message:\t", err)
		return
	}

	err = b.Client.DeleteMessage(del)
	if err != nil {
		b.Logger.Println("Can't delete message:\t", err)
		return
	}

	err = b.Storage.UpdateMessageStatus(del)
	if err != nil {
		b.Logger.Println(err)
		return
	}
}

func (b *Bot) AddExpectedTrigger(message *types.Message) {
	b.SendText(message, "Type in trigger phrase")

	err := b.Storage.InsertExpectedMessage(message, true, false)
	if err != nil {
		b.Logger.Println("Can't add expected trigger: ", err)
		return
	}
}
