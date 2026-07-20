package types

type GetMeResponse struct {
	Ok     bool `json:"ok"`
	Result User `json:"result"`
}

type TgResponse struct {
	Ok     bool `json:"ok"`
	Result any  `json:"result"`
}

type SendResponse struct {
	Ok     bool    `json:"ok"`
	Result Message `json:"result"`
}

type SetCommandsResponse struct {
	Ok     bool `json:"ok"`
	Result bool `json:"result"`
}

type GetUpdateResponse struct {
	Ok     bool     `json:"ok"`
	Result []Update `json:"result"`
}

type GetAdministratorsResponse struct {
	Ok     bool                      `json:"ok"`
	Result []ChatMemberAdministrator `json:"result"`
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

type ChatMemberAdministrator struct {
	User User `json:"user"`
}
