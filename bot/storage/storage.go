package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sv/bot/utils"
	"sv/types"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var logger = log.New(os.Stdout, "Storage log:\t", log.Lshortfile|log.LstdFlags)

type Storage struct {
	db     *sql.DB
	logger *log.Logger
}

func NewStorage(db *sql.DB) *Storage {
	return &Storage{
		db:     db,
		logger: logger,
	}
}

func SetUpStorage() *sql.DB {
	password := os.Getenv("POSTGRES_PASSWORD")
	if password == "" {
		logger.Fatal("POSTGRES_PASSWORD is uninitialized inside .env file")
	}

	dsn := "postgres://postgres:" + password + "@localhost:5432/postgres"

	postgres_db, err := sql.Open("pgx", dsn)
	if err != nil {
		logger.Fatal("Can't open connection to the postgres data base ", err) //gracefull shut
	}

	err = postgres_db.Ping()
	if err != nil {
		logger.Fatal("Error during connection to the postgres data base ", err)
	}

	logger.Println("Connection to postgres data base was successfully established")

	_, err = postgres_db.Exec("SELECT 1 FROM pg_database WHERE datname = 'telegram_bot'")
	if err == nil {
		logger.Println("Data dase Telegram_Bot already exists")
	} else {
		_, err = postgres_db.Exec("CREATE DATABASE Telegram_Bot")
		if err != nil {
			logger.Println("Can't create database ", err)
		} else {
			logger.Println("Data base Telegram_Bot was successfully created")
		}
	}

	postgres_db.Close()

	newDSN := "postgres://postgres:" + password + "@localhost:5432/telegram_bot"

	db, err := sql.Open("pgx", newDSN)
	if err != nil {
		logger.Fatal("Can't open connection to the Telegram_Bot database ", err)
	}

	err = db.Ping()
	if err != nil {
		logger.Fatal("Error during connection to the Telegram_Bot data base ", err)
	}

	logger.Println("Connection to Telegram_Bot data base was successfully established")

	query := `
		CREATE TABLE IF NOT EXISTS CHATS (
		ChatID BIGINT NOT NULL, 
		USERNAME VARCHAR(50) NOT NULL, 
		TYPE VARCHAR(50) NOT NULL, 
		CONSTRAINT pk_chat PRIMARY KEY(ChatID))
		`

	_, err = db.Exec(query)
	if err != nil {
		logger.Fatal("Error dutring creation of CHATS table: ", err)
	} else {
		logger.Println("Table CHATS table successfully created")
	}

	query = `
		CREATE TABLE IF NOT EXISTS STICKERS (
			FileUniqueID VARCHAR(100) NOT NULL,
			FileID VARCHAR(100) NOT NULL,
			Emoji VARCHAR (20) NOT NULL, 
			SetName VARCHAR(100) NOT NULL,
			CONSTRAINT pk_uniqueid PRIMARY KEY(FileUniqueID)
		)
	`

	_, err = db.Exec(query)
	if err != nil {
		logger.Fatal("Error dutring creation of STICKERS table: ", err)
	} else {
		logger.Println("Table STICKER table successfully created")
	}

	query = `
		CREATE TABLE IF NOT EXISTS MESSAGES (
		MessageID BIGINT NOT NULL, 
		UserID BIGINT NOT NULL, 
		UserName VARCHAR(100) NOT NULL,
		ChatID BIGINT NOT NULL, 
		Text VARCHAR(4096), 
		StickerID VARCHAR(100), 
		CreatedON TIMESTAMPTZ NOT NULL, 
		Deleted BOOL NOT NULL,
		CONSTRAINT unique_message UNIQUE(MessageID, ChatID),
		CONSTRAINT fk_chat FOREIGN KEY(ChatID) REFERENCES CHATS(ChatID) ON DELETE CASCADE,
		CONSTRAINT fk_stickerid FOREIGN KEY(StickerID) REFERENCES STICKERS(FileUniqueID) ON DELETE CASCADE)
		`

	_, err = db.Exec(query)
	if err != nil {
		logger.Fatal("Error dutring creation of MESSAGES table: ", err)
	} else {
		logger.Println("Table MESSAGES table successfully created")
	}

	query = `
		CREATE TABLE IF NOT EXISTS EXPECTED_MESSAGES (
		ChatID BIGINT NOT NULL,
		UserID BIGINT NOT NULL,
		State VARCHAR(50) NOT NULL,
		CreatedON TIMESTAMPTZ NOT NULL,
		CONSTRAINT fk_chatid FOREIGN KEY(ChatID) REFERENCES CHATS(ChatID) ON DELETE CASCADE,
		CONSTRAINT unique_pair UNIQUE(ChatID, UserID))
	`
	_, err = db.Exec(query)
	if err != nil {
		logger.Fatal("Error dutring creation of EXPECTED_MESSAGES table: ", err)
	} else {
		logger.Println("EXPECTED_MESSAGES table was successfully created")
	}

	query = `
		CREATE TABLE IF NOT EXISTS TRIGGERS (
		ChatID BIGINT NOT NULL,
		UserID BIGINT NOT NULL,
		Username VARCHAR(100) NOT NULL,
		TriggerType VARCHAR(20) NOT NULL,
		Trigger VARCHAR(4096),
		TriggerResp VARCHAR(4096),
		RespType VARCHAR(20),
		CONSTRAINT unique_identifier UNIQUE(ChatID, Trigger),
		CONSTRAINT fk_chat_id FOREIGN KEY(ChatID) REFERENCES CHATS(ChatID) ON DELETE CASCADE)
	`

	_, err = db.Exec(query)
	if err != nil {
		logger.Fatal("Error dutring creation of TRIGGERS table: ", err)
	} else {
		logger.Println("TRIGGERS table was successfully created")
	}
	return db
}

func (S *Storage) InsertChat(chat *types.Chat) error {
	var exists bool
	row := S.db.QueryRow("SELECT EXISTS(SELECT 1 FROM CHATS WHERE chatid = $1)", chat.ID)
	err := row.Scan(&exists)
	if err != nil {
		return fmt.Errorf("Error during checking chat existence: %w", err)
	}

	if exists {
		return nil
	}

	S.logger.Println("New Chat was created")
	query := `
		INSERT INTO CHATS (chatid, username, type)
		VALUES($1, $2, $3)
	`

	_, err = S.db.Exec(query, chat.ID, chat.Username, chat.Type)
	if err != nil {
		return fmt.Errorf("Error during chat insertion: %w", err)
	}

	return nil
}

func (S *Storage) InsertSticker(sticker *types.Sticker) error {
	var exists bool
	err := S.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM STICKERS WHERE FileUniqueID = $1)`, sticker.FileUniqueID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("Error during checking sticker existence: %w", err)
	}

	if exists {
		return nil
	}

	query := `
		INSERT INTO STICKERS (FileUniqueID, FileID, Emoji, SetName)
		VALUES($1, $2, $3, $4)
	`

	_, err = S.db.Exec(query, sticker.FileUniqueID, sticker.FileID, sticker.Emoji, sticker.SetName)
	if err != nil {
		return fmt.Errorf("Error during inserting new sticker: %w", err)
	}

	return nil
}

func (S *Storage) InsertMessage(message *types.Message) error {
	err := S.InsertChat(&message.Chat)
	if err != nil {
		return fmt.Errorf("Can't insert Chat for message insertion: %w", err)
	}

	err = S.InsertSticker(&message.Sticker)
	if err != nil {
		return fmt.Errorf("Can't insert Sticker for message insertion: %w", err)
	}

	query := `
		INSERT INTO MESSAGES (MessageID, UserID, UserName, ChatID, TEXT, StickerID, CreatedON, Deleted) 
		VALUES($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = S.db.Exec(query, message.MessageID, message.From.UserID, message.From.Username,
		message.Chat.ID, message.Text, message.Sticker.FileUniqueID, time.Unix(message.Date, 0), false)
	if err != nil {
		return fmt.Errorf("Can't insert message: %w", err)
	}

	return nil
}

func (S *Storage) UpdateMessageStatus(param *types.DeleteMessage) error {
	query := `
		UPDATE MESSAGES
		SET Deleted = TRUE
		WHERE ChatID = $1 AND MessageID = $2
	`
	_, err := S.db.Exec(query, param.Chat_ID, param.MessageID)
	if err != nil {
		return fmt.Errorf("Can't update message status to deleted: %w", err)
	}

	return nil
}

func (S *Storage) SelectLastMessage(chatID, botID int64) (*types.DeleteMessage, error) {
	query := `
		SELECT 
			MessageID, 
			ChatID
		FROM MESSAGES 
		WHERE ChatID = $1 AND UserID = $2
		ORDER BY CreatedON DESC 
		LIMIT 1
	`

	row := S.db.QueryRow(query, chatID, botID)

	var MessageID, ChatID int64

	err := row.Scan(&MessageID, &ChatID)

	if err != nil {
		return nil, fmt.Errorf("Error during searching for last message: %w", err)
	}

	resp := &types.DeleteMessage{
		MessageID: MessageID,
		Chat_ID:   ChatID,
	}

	return resp, nil
}

func (S *Storage) InsertExpectedMessage(message *types.Message, state string) error {
	query := `
		INSERT INTO EXPECTED_MESSAGES (ChatID, UserId, State, CreatedON)
		VALUES($1, $2, $3, $4)
		ON CONFLICT(ChatID, UserID) DO NOTHING
	`

	_, err := S.db.Exec(query, message.Chat.ID, message.From.UserID, state, time.Unix(message.Date, 0))
	if err != nil {
		return fmt.Errorf("Can't insert trigger to EXPECTED_MESSAGE table: %w", err)
	}

	return nil
}

func (S *Storage) GetExpectedMessageState(message *types.Message) (State string, err error) {
	query := `
		SELECT 
			State
		FROM EXPECTED_MESSAGES
		WHERE ChatID = $1 AND UserID = $2
	`

	err = S.db.QueryRow(query, message.Chat.ID, message.From.UserID).Scan(&State)
	if err != nil {
		return "", fmt.Errorf("Error during getting status of expected message: %w", err)
	}

	return State, nil
}

func (S *Storage) IsExpected(message *types.Message) (bool, error) {
	query := `
		SELECT EXISTS (SELECT * FROM EXPECTED_MESSAGES 
		WHERE ChatID = $1 AND UserID = $2)
	`
	var exists bool
	err := S.db.QueryRow(query, message.Chat.ID, message.From.UserID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("Error during checking whether message is expected: %w", err)
	}

	return exists, nil
}

func (S *Storage) DeleteExpectedMessage(message *types.Message) error {
	query := `
		DELETE FROM EXPECTED_MESSAGES WHERE ChatID = $1 AND UserID = $2
	`

	resp, err := S.db.Exec(query, message.Chat.ID, message.From.UserID)
	if err != nil {
		return fmt.Errorf("Can't delete message from EXPECTED_MESSAGE table: %w", err)
	}

	affected, err := resp.RowsAffected()
	if affected != 1 {
		return fmt.Errorf("There is no such expected message")
	}

	if err != nil {
		return fmt.Errorf("Can'delete trigger: %w", err)
	}

	return nil
}

func (S *Storage) InsertTrigger(message *types.Message) error {
	query := `
		INSERT INTO TRIGGERS (ChatID, UserID, Username, TriggerType, Trigger)
		VALUES($1, $2, $3, $4, $5) 
		ON CONFLICT (ChatID, Trigger) DO NOTHING
		RETURNING 1
	`
	var Trigger, TriggerType string

	if !(message.Sticker.FileUniqueID == "") {
		TriggerType = "Sticker"
		err := S.InsertSticker(&message.Sticker)
		if err != nil {
			return fmt.Errorf("Can't insert trigger: %w", errors.Join(err, S.DeleteExpectedMessage(message)))
		}
		Trigger = message.Sticker.FileUniqueID
	} else {
		TriggerType = "Text"
		Trigger = strings.TrimSpace(message.Text)
	}

	var i int
	err := S.db.QueryRow(query, message.Chat.ID, message.From.UserID, message.From.Username,
		TriggerType, Trigger).Scan(&i)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.Join(err, S.DeleteExpectedMessage(message), utils.ErrTriggerExists)
		}
		return errors.Join(fmt.Errorf("Can't insert trigger: %w", err), S.DeleteTrigger(message, false))
	}

	query = `
		UPDATE EXPECTED_MESSAGES
		SET
			State = 'TriggerResp'
		WHERE ChatID = $1 AND UserID = $2
	`

	_, err = S.db.Exec(query, message.Chat.ID, message.From.UserID)
	if err != nil {
		err = errors.Join(err, utils.ExecuteRollBack(
			func() error { return S.DeleteExpectedMessage(message) },
			func() error { return S.DeleteTrigger(message, false) },
		))
		return fmt.Errorf("Can't change expected message status to IsTriggerResponse %w", err)
	}

	return nil
}

func (S *Storage) AddTriggerResponse(message *types.Message) error {
	var TriggerResp, RespType string

	if !(message.Sticker.FileUniqueID == "") {
		RespType = "Sticker"
		err := S.InsertSticker(&message.Sticker)
		if err != nil {
			return fmt.Errorf("Can't insert trigger: %w", errors.Join(err, S.DeleteExpectedMessage(message)))
		}
		TriggerResp = message.Sticker.FileUniqueID
	} else {
		RespType = "Text"
		TriggerResp = strings.TrimSpace(message.Text)
	}
	query := `
		UPDATE TRIGGERS 
		SET 
			TriggerResp = $1,
			RespType = $2
		WHERE ChatID = $3 AND UserID = $4 AND TriggerResp IS NULL
	`

	_, err := S.db.Exec(query, TriggerResp, RespType, message.Chat.ID, message.From.UserID)
	if err != nil {
		err = errors.Join(err, utils.ExecuteRollBack(
			func() error { return S.DeleteExpectedMessage(message) },
			func() error { return S.DeleteTrigger(message, false) },
		))

		return fmt.Errorf("Can't add trigger reponse %w", err)
	}

	err = S.DeleteExpectedMessage(message)
	if err != nil {
		erro := S.DeleteTrigger(message, false)
		if erro != nil {
			return fmt.Errorf("Error during adding trigger response: %w, %w", err, erro)
		}
		return fmt.Errorf("Error during adding trigger response: %w", err)
	}

	return nil
}

func (S *Storage) GetChatTriggers(message *types.Message) (string, error) {
	query := `
		SELECT 
			t.Trigger, 
			t.Username,
			s.Emoji,
			s.SetName
		FROM TRIGGERS t
		LEFT JOIN STICKERS s
			ON t.Trigger = s.FileUniqueID
		WHERE t.ChatID = $1
		ORDER BY t.Username
	`

	row, err := S.db.Query(query, message.Chat.ID)
	if err != nil {
		return "", fmt.Errorf("Can't get chat triggers: %w", err)
	}

	defer row.Close()
	var result, Trigger, Username string
	var Emoji, SetName *string
	for row.Next() {
		if err = row.Scan(&Trigger, &Username, &Emoji, &SetName); err != nil {
			return "", fmt.Errorf("Can't get chat triggers: %w", err)
		}

		if Emoji == nil {
			Trigger = utils.TrancateText(Trigger, 35)
			result += fmt.Sprintf("Trigger: %s\n   🔴 Creator: %s\n", Trigger, Username)
		} else {
			*SetName = utils.TrancateText(*SetName, 30)
			result += fmt.Sprintf("Trigger: %s [From: %s]\n   🔴 Creator: %s\n", *Emoji, *SetName, Username)
		}
	}

	if err := row.Err(); err != nil {
		return "", fmt.Errorf("Can't get chat triggers. Error during rows cycle: %w", err)
	}

	if result == "" {
		return "", sql.ErrNoRows
	}

	return result, nil
}

func (S *Storage) GetPersonTriggers(message *types.Message) (string, error) {
	query := `
		SELECT 
			t.Trigger,
			s.Emoji,
			s.SetName
		FROM TRIGGERS t
		LEFT JOIN STICKERS s
			ON t.Trigger = s.FileUniqueID
		WHERE t.ChatID = $1 AND t.UserID = $2
		ORDER BY t.Trigger
	`

	row, err := S.db.Query(query, message.Chat.ID, message.From.UserID)
	if err != nil {
		return "", fmt.Errorf("Can't get person triggers: %w", err)
	}

	defer row.Close()
	var result, Trigger string
	var Emoji, SetName *string
	for row.Next() {
		if err = row.Scan(&Trigger, &Emoji, &SetName); err != nil {
			return "", fmt.Errorf("Can't get person triggers: %w", err)
		}

		if Emoji == nil {
			utils.TrancateText(Trigger, 35)
			result += fmt.Sprintf("Trigger: %s\n", Trigger)
		} else {
			utils.TrancateText(*SetName, 30)
			result += fmt.Sprintf("Trigger: %s From[: %s]\n", *Emoji, *SetName)
		}

	}

	if err := row.Err(); err != nil {
		return "", fmt.Errorf("Can't get person triggers. Error during rows cycle: %w", err)
	}

	if result == "" {
		return "", sql.ErrNoRows
	}
	return result, nil
}

func (S *Storage) IsTrigger(message *types.Message) (IsTrigger bool, err error) {
	query := `
		SELECT EXISTS (SELECT 1 FROM TRIGGERS
		WHERE ChatID = $1 AND Trigger = $2 AND TriggerResp IS NOT NULL)  
	`

	err = S.db.QueryRow(query, message.Chat.ID, strings.TrimSpace(message.Text+message.Sticker.FileUniqueID)).Scan(&IsTrigger)
	if err != nil {
		return IsTrigger, fmt.Errorf("Can't check whether message is trigger: %w", err)
	}

	return IsTrigger, nil
}

func (S *Storage) GetTriggerResp(message *types.Message) (Resp string, RespType string, err error) {
	query := `
		SELECT TriggerResp, RespType FROM TRIGGERS
		WHERE ChatID = $1 AND Trigger = $2 AND TriggerResp IS NOT NULL 
	`

	err = S.db.QueryRow(query, message.Chat.ID,
		strings.TrimSpace(message.Text+message.Sticker.FileUniqueID)).Scan(&Resp, &RespType)

	if err != nil {
		return "", "", fmt.Errorf("Can't get trigger response: %w", err)
	}

	if RespType == "Sticker" {
		err = S.db.QueryRow(`SELECT FileID FROM STICKERS WHERE FileUniqueID = $1`, Resp).Scan(&Resp)
		if err != nil {
			return "", "", fmt.Errorf("Can't get trigger response: %w", err)
		}
	}

	return Resp, RespType, nil
}

func (S *Storage) DeleteTrigger(message *types.Message, IsAdmin bool) error {
	query := `
		DELETE FROM TRIGGERS WHERE ChatID = $1 AND Trigger = $2 AND (UserID = $3 OR $4)
	`

	resp, err := S.db.Exec(query, message.Chat.ID, strings.TrimSpace(message.Text+message.Sticker.FileUniqueID),
		message.From.UserID, IsAdmin)

	if err != nil {
		return fmt.Errorf("Can' delete trigger: %w", err)
	}

	affected, err := resp.RowsAffected()
	if affected != 1 {
		return fmt.Errorf("Can't delete trigger: %w or you don't have the permission to delete this trigger", utils.ErrTriggerDontExists)
	}

	if err != nil {
		return fmt.Errorf("Can't delete trigger: %w", err)
	}

	S.logger.Println("Trigger was successfully deleted")

	return nil
}

func (S *Storage) DeleteOldMessages() error {
	query := `
	DELETE FROM MESSAGES 
	WHERE CreatedON < NOW() - INTERVAL '3 months';

	DELETE FROM EXPECTED_MESSAGESS 
	WHERE CreatedON < NOW() - INTERVAL '1 day';
	`

	affected, err := S.db.Exec(query)
	if err != nil {
		return fmt.Errorf("Can't delete old messages: %w", err)
	}

	if i, _ := affected.RowsAffected(); i != 0 {
		S.logger.Printf("During DeleteOldMessages %d messages was deleted\n", i)
	}

	return nil
}
