package bot

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sv/bot/utils"
	"sv/types"
)

func (b *Bot) isCommand(Message *types.Message) (commad func() error, ok bool) {
	var CommandList = map[string]func() error{
		"add_trigger":    func() error { return b.addExpectedTrigger(Message) },
		"delete_trigger": func() error { return b.addExpectedDeleteTrigger(Message) },
		"chat_triggers":  func() error { return b.getChatTriggers(Message) },
		"my_triggers":    func() error { return b.getPersonTriggers(Message) },
		// start, getmytriggers, get all cards, get my cards, deletealltriggers
	}

	txt := strings.Split(strings.TrimPrefix(strings.TrimSpace(strings.ToLower(Message.Text)), "/"), "@")[0]
	c, ok := CommandList[txt]
	if !ok {
		return nil, ok
	}

	return c, ok
}

func (b *Bot) sendText(message *types.Message, text string) error {
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

func (b *Bot) sendSticker(message *types.Message, StickerFileID string) error {
	stic := &types.SendSticker{
		Chat_ID:       message.Chat.ID,
		StickerFileID: StickerFileID,
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

func (b *Bot) sendStruct(command string, param types.InputStruct) error {
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

func (b *Bot) deleteMessage(param *types.DeleteMessage) error {
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

func (b *Bot) deleteLastBotsMessage(chatID int64) error {
	del, err := b.Storage.SelectLastMessage(chatID, b.ID)
	if err != nil {
		return fmt.Errorf("Can't delete last message: ChatID:%d, %w", chatID, err)
	}

	err = errors.Join(b.Client.DeleteMessage(del), b.Storage.UpdateMessageStatus(del))
	if err != nil {
		return fmt.Errorf("Can't delete last message: ChatID:%d, %w", chatID, err)
	}

	return nil
}

func (b *Bot) addExpectedTrigger(message *types.Message) error {
	err := b.sendText(message, "Type in trigger phrase")
	if err != nil {
		return fmt.Errorf("Can't execute AddExpectedTrigger %w", err)
	}

	err = b.Storage.InsertExpectedMessage(message, "Trigger")
	if err != nil {
		return fmt.Errorf("Can't execute AddExpectedTrigger: %w", err)
	}

	return nil
}

func (b *Bot) addTrigger(message *types.Message) error {
	err := b.verifyType(message)
	if err != nil {
		err := errors.Join(err, b.Storage.DeleteExpectedMessage(message))
		return fmt.Errorf("Can't add trigger. Can't VerifyType of message: %w", err)
	}
	message.Text = strings.ToLower(strings.TrimSpace(message.Text))
	err = b.Storage.InsertTrigger(message)
	if err != nil {
		if errors.Is(err, utils.ErrTriggerExists) {
			err = errors.Join(err, b.sendText(message, "Such trigger already exists"), utils.ErrUserNotified)
			return fmt.Errorf("Can't add trigger. Error during message handling: %w", err)
		}

		return fmt.Errorf("Can't add trigger. Error during message handling: %w", err)
	}

	err = b.sendText(message, "Now send a response to the trigger")
	if err != nil {
		err = errors.Join(err, b.Storage.DeleteTrigger(message, false), b.Storage.DeleteExpectedMessage(message))

		return fmt.Errorf("Can't add trigger. Error during message handling: %w", err)
	}
	return nil
}

func (b *Bot) addTriggerResp(message *types.Message) error {
	err := b.verifyType(message)
	if err != nil {
		err = errors.Join(err, b.Storage.DeleteTrigger(message, false), b.Storage.DeleteExpectedMessage(message))
		return fmt.Errorf("Can't set trigger response. Can't VerifyType of message: %w", err)
	}

	err = b.Storage.AddTriggerResponse(message)
	if err != nil {
		err = errors.Join(err, b.Storage.DeleteTrigger(message, false))
		return fmt.Errorf("Can't set trigger response. Error during message handling: %w", err)
	}

	err = b.sendText(message, "Trigger saved successfully")
	if err != nil {
		return fmt.Errorf("Can't send a submission message during message handling: %w", err)
	}

	return nil
}

func (b *Bot) addExpectedDeleteTrigger(message *types.Message) error {
	err := b.sendText(message, "Type in the trigger that you want to delete")
	if err != nil {
		return fmt.Errorf("Can't execute AddExpectedDeleteTrigger, %w", err)
	}

	err = b.Storage.InsertExpectedMessage(message, "TriggerName")
	if err != nil {
		return fmt.Errorf("Can't execute AddExpectedDeleteTrigger: %w", err)
	}

	return nil
}

func (b *Bot) deleteTrigger(message *types.Message) error {
	err := b.verifyType(message)
	if err != nil {
		return fmt.Errorf("Can't delete trigger: %w", err)
	}

	err = errors.Join(err, b.Storage.DeleteExpectedMessage(message))
	IsAdmin, err := b.Client.IsAdministrator(&message.Chat, message.From.UserID)
	if err != nil {
		return fmt.Errorf("Can't Delete trigger: %w", err)
	}

	err = b.Storage.DeleteTrigger(message, IsAdmin)
	if err != nil {
		if errors.Is(err, utils.ErrTriggerDontExists) {
			err = errors.Join(err, b.sendText(message, utils.ErrTriggerDontExists.Error()+
				" or you don't have the permission to delete this trigger"), utils.ErrUserNotified)
			return fmt.Errorf("Can't DeleteTrigger: %w", err)
		}
		return fmt.Errorf("Can't DeleteTrigger: %w", err)
	}

	err = errors.Join(err, b.sendText(message, "Trigger was successfully deleted"))
	if err != nil {
		return fmt.Errorf("Can't DeleteTrigger: %w", err)
	}

	return nil
}

func (b *Bot) getChatTriggers(message *types.Message) error {
	resp, err := b.Storage.GetChatTriggers(message)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.Join(err, b.sendText(message, "There is no triggers yet"), utils.ErrUserNotified)
			return err
		}
		return err
	}

	if err = b.sendText(message, resp); err != nil {
		return fmt.Errorf("Can't get chat triggers: %w", err)
	}

	return nil
}

func (b *Bot) getPersonTriggers(message *types.Message) error {
	resp, err := b.Storage.GetPersonTriggers(message)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.Join(err, b.sendText(message, "There is no triggers yet"), utils.ErrUserNotified)
			return err
		}
		return err
	}

	if err = b.sendText(message, resp); err != nil {
		return fmt.Errorf("Can't get person triggers: %w", err)
	}

	return nil
}

func (b *Bot) verifyType(message *types.Message) error {
	if _, ok := b.isCommand(message); ok {
		err := errors.Join(b.sendText(message, "You can't use this message because it is one of the bot commands, if you still want to create a trigger use command again"), utils.ErrUserNotified)
		return fmt.Errorf("User send message of invalid type: %w", err)
	} else if fmt.Sprint(message.Text+message.Sticker.FileID) == "" {
		err := b.sendText(message, "Trigger must be a text or sticker, if you still want to create a trigger use command again")
		err = errors.Join(err, b.Storage.DeleteExpectedMessage(message), utils.ErrUserNotified)
		return fmt.Errorf("User send message of invalid type: %w", err)
	}

	return nil
}
