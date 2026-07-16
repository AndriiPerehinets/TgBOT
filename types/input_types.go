package types

type InputStruct interface{}

type Sticker struct {
	FileID string `json:"file_id"`
}

type SendText struct {
	Chat_ID int64  `json:"chat_id"`
	Text    string `json:"text"`
}

type SendSticker struct {
	Chat_ID int64  `json:"chat_id"`
	Sticker string `json:"sticker"`
}

type DeleteMessage struct {
	Chat_ID   int64 `json:"chat_id"`
	MessageID int64 `json:"message_id"`
}

type GetUpdate struct {
	Offset  int64 `json:"offset"`
	Timeout int64 `json:"timeout"`
}

type BotCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

type SetBotCommand struct {
	Commands []BotCommand `json:"commands"`
	Scope    Scope        `json:"scope"`
}

type Scope struct {
	Type string `json:"type"`
}
