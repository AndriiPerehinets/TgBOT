package types

type SendResponse struct {
	Ok       bool    `json:"ok"`
	Response Message `json:"result"`
}

type GetUpdateResponse struct {
	Ok       bool     `json:"ok"`
	Response []Update `json:"result"`
}

type Update struct {
	UpdateID int64   `json:"update_id"`
	Message  Message `json:"message"`
}

type Message struct {
	MessageID int64   `json:"message_id"`
	From      User    `json:"from"`
	Chat      Chat    `json:"chat"`
	Date      int64   `sson:"date"`
	Text      string  `json:"text"`
	Sticker   Sticker `json:"sticker"`
	Left_Chat User    `json:"left_chat_member"` //для відсетження чи кікнули бота з чату
}

type Chat struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	Username string `json:"username"`
}

type User struct {
	UserID   int64  `json:"id"`
	Username string `json:"username"`
}
