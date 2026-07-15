package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
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
		CREATE TABLE IF NOT EXISTS MESSAGES (
		MessageID BIGINT NOT NULL, 
		UserID BIGINT NOT NULL, 
		UserName VARCHAR(100) NOT NULL,
		ChatID BIGINT NOT NULL, 
		Text VARCHAR(1000), 
		Sticker VARCHAR(100), 
		Time TIMESTAMPTZ NOT NULL, 
		Deleted BOOL NOT NULL,
		CONSTRAINT pk_message PRIMARY KEY(MessageID),
		CONSTRAINT fk_chat FOREIGN KEY(ChatID) REFERENCES CHATS(ChatID) ON DELETE CASCADE)
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
		IsTrigger BOOL NOT NULL,
		IsTriggerResp BOOL NOT NULL,
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
		TriggerPhrase VARCHAR(1000),
		TriggerSticker VARCHAR(100),
		TriggerResp VARCHAR(1000),
		IsRespSticker BOOL,
		CONSTRAINT unique_identifier UNIQUE(ChatID, TriggerPhrase, TriggerSticker),
		CONSTRAINT fk_chatid FOREIGN KEY(ChatID) REFERENCES CHATS(ChatID) ON DELETE CASCADE) 
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
		S.logger.Println("Chat already exitst inside CHATS table")
		return nil
	}

	query := `
		INSERT INTO CHATS (chatid, username, type)
		VALUES($1, $2, $3)
	`

	_, err = S.db.Exec(query, chat.ID, chat.Username, chat.Type)
	if err != nil {
		return fmt.Errorf("Error during chat insertion: %w", err)
	}

	S.logger.Println("Chat was successfully inserted")

	return nil
}

func (S *Storage) InsertMessage(message *types.Message) error {
	err := S.InsertChat(&message.Chat)
	if err != nil {
		return fmt.Errorf("Can't insert message: %w", err)
	}

	date := time.Unix(message.Date, 0)

	query := `
		INSERT INTO MESSAGES (MessageID, UserID, UserName, ChatID, TEXT, Sticker, Time, Deleted) 
		VALUES($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = S.db.Exec(query, message.MessageID, message.From.UserID, message.From.Username,
		message.Chat.ID, message.Text, message.Sticker.FileID, date, false)
	if err != nil {
		return fmt.Errorf("Can't insert message: %w", err)
	}

	S.logger.Println("Message was successfully inserted")

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

	S.logger.Println("Message was successfully updated")

	return nil
}

func (S *Storage) SelectLastMessage(chatID, botID int64) (*types.DeleteMessage, error) {
	query := `
		SELECT 
			MessageID, 
			ChatID
		FROM MESSAGES 
		WHERE ChatID = $1 AND UserID = $2
		ORDER BY Time DESC 
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

func (S *Storage) InsertExpectedMessage(message *types.Message, IsTrigger, IsTriggerPhrase bool) error {
	if IsTrigger && IsTriggerPhrase {
		return fmt.Errorf("Invalid method call Message can't be trigger and trigger response at the same time")
	}

	query := `
		INSERT INTO EXPECTED_MESSAGES (ChatID, UserId, IsTrigger, IsTriggerResp)
		VALUES($1, $2, $3, $4)
		ON CONFLICT(ChatID, UserID) DO NOTHING
	`

	_, err := S.db.Exec(query, message.Chat.ID, message.From.UserID, IsTrigger, IsTriggerPhrase)
	if err != nil {
		return fmt.Errorf("Can't inserted expected trigger: %w", err)
	}

	S.logger.Println("Expected message was successfully inserted")

	return nil
}

func (S *Storage) GetExpectedMessageStatus(message *types.Message) (IsTrigger, IsTriggerResp bool, err error) {
	query := `
		SELECT 
			IsTrigger,
			IsTriggerResp
		FROM EXPECTED_MESSAGES
		WHERE ChatID = $1 AND UserID = $2
	`

	err = S.db.QueryRow(query, message.Chat.ID, message.From.UserID).Scan(&IsTrigger, &IsTriggerResp)
	if err != nil {
		return false, false, fmt.Errorf("Error during geting status of expected message: %w", err)
	}

	return IsTrigger, IsTriggerResp, nil
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

	return exists, err
}

func (S *Storage) DeleteExpectedMessage(message *types.Message) error {
	query := `
		DELETE FROM EXPECTED_MESSAGES WHERE ChatID = $1 AND UserID = $2
	`

	_, err := S.db.Exec(query, message.Chat.ID, message.From.UserID)

	if err != nil {
		return fmt.Errorf("Can't delete expected message: %w", err)
	}

	S.logger.Println("Message was successfully deleted")

	return nil
}

func (S *Storage) InsertTrigger(message *types.Message) error {
	query := `
		INSERT INTO TRIGGERS (ChatID, UserID, TriggerPhrase, TriggerSticker, IsRespSticker)
		VALUES($1, $2, $3, $4, $5) 
		ON CONFLICT (ChatID, TriggerPhrase, TriggerSticker) DO NOTHING
		RETURNING 1
	`
	var IsSticker = false
	if message.Sticker.FileID != "" {
		IsSticker = true
	}

	var A int
	err := S.db.QueryRow(query, message.Chat.ID, message.From.UserID, strings.TrimSpace(message.Text), message.Sticker.FileID, IsSticker).Scan(&A)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("Such trigger already exists")
		}
		return fmt.Errorf("Can't insert trigger %w", err)
	}

	query = `
		UPDATE EXPECTED_MESSAGES
		SET
			IsTrigger = false,
			IsTriggerResp = true
		WHERE ChatID = $1 AND UserID = $2
	`

	_, err = S.db.Exec(query, message.Chat.ID, message.From.UserID)
	if err != nil {
		return fmt.Errorf("Can't change expected message status to IsTriggerResponse: %w", err)
	}

	S.logger.Println("Trigger was successfully saved")

	return nil
}

func (S *Storage) AddTriggerResponse(message *types.Message) error {
	var IsSticker = false
	if message.Sticker.FileID != "" {
		IsSticker = true //!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	}

	query := `
		UPDATE TRIGGERS 
		SET 
			TriggerResp = $1,
			IsRespSticker = $2
		WHERE ChatID = $3 AND UserID = $4 AND TriggerResp IS NULL
	`

	_, err := S.db.Exec(query, strings.TrimSpace(message.Text), IsSticker, message.Chat.ID, message.From.UserID)
	if err != nil {
		return fmt.Errorf("Can't add trigger response: %w", err)
	}

	err = S.DeleteExpectedMessage(message)
	if err != nil {
		return fmt.Errorf("Error during adding trigger response: %w", err)
	}

	return nil
}

func (S *Storage) IsTrigger(message *types.Message) (IsTrigger bool, err error) {
	query := `
		SELECT EXISTS (SELECT 1 FROM TRIGGERS
		WHERE ChatID = $1 AND TriggerPhrase = $2 AND TriggerSticker = $3 AND TriggerResp IS NOT NULL)  
	`

	err = S.db.QueryRow(query, message.Chat.ID, strings.TrimSpace(message.Text), message.Sticker.FileID).Scan(&IsTrigger)
	if err != nil {
		return IsTrigger, fmt.Errorf("Can't check whether message is trigger: %w", err)
	}

	return IsTrigger, nil
}

func (S *Storage) GetTriggerResp(message *types.Message) (resp string, IsSticker bool, err error) {
	query := `
		SELECT TriggerResp, IsRespSticker FROM TRIGGERS
		WHERE ChatID = $1 AND TriggerPhrase = $2 AND TriggerSticker = $3 AND TriggerResp IS NOT NULL 
	`

	err = S.db.QueryRow(query, message.Chat.ID, strings.TrimSpace(message.Text), message.Sticker.FileID).Scan(&resp, &IsSticker)
	if err != nil {
		return "", false, fmt.Errorf("Can't get trigger response: %w", err)
	}

	return resp, IsSticker, nil
}

func (S *Storage) DeleteTrigger(message *types.Message) error {
	query := `
		DELETE FROM TRIGGERS WHERE ChatID = $1 AND TriggerPhrase = $2 AND TriggerSticker = $3
	`

	_, err := S.db.Exec(query, message.Chat.ID, strings.TrimSpace(message.Text), message.Sticker.FileID)

	if err != nil {
		return fmt.Errorf("Can'delete trigger message: %w", err)
	}

	S.logger.Println("Trigger was successfully deleted")

	return nil
}
