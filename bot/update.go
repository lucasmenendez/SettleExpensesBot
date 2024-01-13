package bot

import (
	"strings"
)

const (
	botEntity = "bot_command"
	cmdPrefix = "/"
	argsSep   = " "
)

type User struct {
	ID           int64  `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName    string `json:"first_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
}

type Chat struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
	Type      string `json:"type"`
}

type Entity struct {
	Offset int64  `json:"offset"`
	Length int64  `json:"length"`
	Type   string `json:"type"`
}

type Message struct {
	Text     string    `json:"text"`
	Date     int64     `json:"date"`
	From     *User     `json:"from"`
	Chat     *Chat     `json:"chat"`
	Entities []*Entity `json:"entities"`
}

type Update struct {
	UpdateID int64    `json:"update_id"`
	Message  *Message `json:"message"`
}

func (u *Update) IsCommand() bool {
	if u.Message == nil {
		return false
	}
	if len(u.Message.Entities) == 0 {
		return false
	}
	return u.Message.Entities[0].Type == botEntity
}

func (u *Update) Command() string {
	if !u.IsCommand() {
		return ""
	}
	entity := u.Message.Entities[0]
	cmd := u.Message.Text[entity.Offset : entity.Offset+entity.Length]
	return strings.TrimPrefix(cmd, cmdPrefix)
}

func (u *Update) CommandArgs() []string {
	if !u.IsCommand() {
		return nil
	}
	entity := u.Message.Entities[0]
	args := strings.TrimSpace(u.Message.Text[entity.Offset+entity.Length:])
	return strings.Split(args, argsSep)
}
