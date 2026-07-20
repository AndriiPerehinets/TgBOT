package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sv/types"
)

type Client struct {
	url    *url.URL
	client *http.Client
	logger *log.Logger
}

func NewClient(token string) *Client {
	return &Client{
		url: &url.URL{
			Scheme: "https",
			Host:   "api.telegram.org",
			Path:   "bot" + token,
		},

		client: &http.Client{},

		logger: log.New(os.Stdout, "Client log:\t", log.Lshortfile|log.LstdFlags),
	}
}

func (c *Client) GetMe() (*types.User, error) {
	req, err := http.NewRequest("GET", c.url.String()+"/getMe", nil)
	if err != nil {
		return nil, fmt.Errorf("Can't create request %w ", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error during request execution %w", err)
	}

	respStruct := &types.GetMeResponse{}

	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(respStruct)
	if err != nil {
		return nil, fmt.Errorf("Can't read decode response: %w", err)
	}

	if !respStruct.Ok {
		return nil, fmt.Errorf("Error during GetMe execution bad response: %t", respStruct.Ok)
	}

	U := respStruct.Result

	return &U, nil
}

func (c *Client) Send(method string, Param types.InputStruct) (*types.Message, error) {
	resp, err := c.doPostRequest(method, Param)
	if err != nil {
		return nil, err
	}

	var A types.SendResponse

	if err = json.Unmarshal(resp, &A); err != nil {
		return nil, fmt.Errorf("can't decode response: %w", err)
	}

	if !A.Ok {
		return nil, fmt.Errorf("Error during Send execution bad response, A.Ok = %t", A.Ok)
	}

	c.logger.Println("Opperation " + method + " was successfully completed")

	return &A.Result, nil
}

func (c *Client) DeleteMessage(param *types.DeleteMessage) error {
	_, err := c.doPostRequest("deletemessage", param)
	if err != nil {
		return err
	}

	c.logger.Println("Message was successfuly deleted")

	return nil
}

func (c *Client) doPostRequest(method string, param types.InputStruct) (response []byte, err error) {
	p, err := json.Marshal(param)
	if err != nil {
		return nil, fmt.Errorf("Paraments can't be marshalled, %w", err)
	}

	b := bytes.NewBuffer(p)

	req, err := http.NewRequest("POST", c.url.String()+"/"+method, b)
	if err != nil {
		return nil, fmt.Errorf("Can't create request %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error during execution of the request %w", err)
	}

	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Can't read data from respoce body %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("StatusCode is not OK, it is:  %s, %s", resp.Status, string(data))
	}

	c.logger.Println("Response data:\t", string(data))

	return data, nil
}

func (c *Client) GetUpdate(offset int64) ([]types.Update, error) {
	param := types.GetUpdate{
		Offset:  offset,
		Timeout: 60,
	}

	resp, err := c.doPostRequest("getUpdates", &param)
	if err != nil {
		return nil, fmt.Errorf("An error occurred during getUpdate request %w", err)
	}

	A := &types.GetUpdateResponse{}

	err = json.Unmarshal(resp, A)
	if err != nil {
		return nil, fmt.Errorf("Can't unmarshal response %w", err)
	}

	if !A.Ok {
		return nil, fmt.Errorf("Bad response, A.Ok = %t", A.Ok)
	}

	return A.Result, nil
}

func (c *Client) SetCommands() error {
	param := &types.SetBotCommand{
		Commands: []types.BotCommand{
			{
				Command:     "add_trigger",
				Description: "Adds a new trigger. Whenever the trigger message is sent, the bot will respond with a predefined by user message",
			},
			{
				Command:     "delete_trigger",
				Description: "Deletes an existing trigger, only creator of trigger or administrator may execute it",
			},
		},
		Scope: types.Scope{
			Type: "all_private_chats",
		},
	}
	resp, err := c.doPostRequest("setMyCommands", param)
	if err != nil {
		return fmt.Errorf("Can't set bot commands: %w", err)
	}

	A := &types.SetCommandsResponse{}
	err = json.Unmarshal(resp, A)
	if err != nil {
		return fmt.Errorf("Can't unmarhsal respose: %w", err)
	}

	if !A.Ok || !A.Result {
		return fmt.Errorf("An error occurred during SetCommads execution, OK = %t , Result = %t", A.Ok, A.Result)
	}

	param.Scope.Type = "all_group_chats"
	param.Commands = append(param.Commands, types.BotCommand{Command: "start", Description: "Starts bot"})

	resp, err = c.doPostRequest("setMyCommands", param)
	if err != nil {
		return fmt.Errorf("Can't set bot commands: %w", err)
	}

	A = &types.SetCommandsResponse{}
	err = json.Unmarshal(resp, A)
	if err != nil {
		return fmt.Errorf("Can't unmarhsal respose: %w", err)
	}

	if !A.Ok || !A.Result {
		return fmt.Errorf("An error occurred during SetCommads execution, OK = %t , Result = %t", A.Ok, A.Result)
	}

	return nil
}

func (c *Client) IsAdministrator(ChatID int64, UserID int64) (bool, error) {
	resp, err := c.doPostRequest("getChatAdministrators", ChatID)
	if err != nil {
		return false, fmt.Errorf("Can't get chat %d administrators, %w", ChatID, err)
	}

	A := &types.GetAdministratorsResponse{}

	err = json.Unmarshal(resp, A)
	if err != nil {
		return false, fmt.Errorf("Can't unmarshal response %w", err)
	}

	for _, i := range A.Result {
		if i.User.UserID == UserID {
			return true, nil
		}
	}

	return false, nil
}
