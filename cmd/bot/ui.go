package main

import (
	"log"
	"strconv"

	"github.com/lucasmenendez/expensesbot/bot"
)

func numPad() ([][]string, [][]string) {
	labels := [][]string{
		{"1", "2", "3"},
		{"4", "5", "6"},
		{"7", "8", "9"},
		{".", "0", "Del"},
		{"Cancel", "Done"},
	}
	values := [][]string{
		{"1", "2", "3"},
		{"4", "5", "6"},
		{"7", "8", "9"},
		{".", "0", "del"},
		{"cancel", "done"},
	}

	return labels, values
}

func requestAmount(b *bot.Bot, chatID int64, text string, callback func(float64)) error {
	labels, values := numPad()
	_, err := b.InlineMenu(chatID, 0, text, [][]string{{"Open numpad"}}, [][]string{{"open_numpad"}}, func(messageID int64, data string) {
		if data == "open_numpad" {
			text := "0"
			if _, err := b.InlineMenu(chatID, messageID, text, labels, values, func(_ int64, char string) {
				switch char {
				case "cancel":
					if err := b.RemoveMessage(chatID, messageID); err != nil {
						log.Println(err)
					}
					return
				case "done":
					if _, err := b.InlineMenu(chatID, messageID, text, nil, nil, nil); err != nil {
						log.Println(err)
					}
					if amount, err := strconv.ParseFloat(text, 64); err == nil {
						callback(amount)
					}
					return
				case "del":
					if len(text) > 0 {
						text = text[:len(text)-1]
					} else {
						text = "0"
					}
				default:
					if text != "0" {
						text += char
					} else {
						text = char
					}
				}
				if _, err := b.InlineMenu(chatID, messageID, text, labels, values, nil); err != nil {
					log.Println(err)
				}
			}); err != nil {
				log.Println(err)
			}
		}
	})
	return err
}

func confirm(b *bot.Bot, chatID int64, prompt string, callback func(bool)) error {
	labels := [][]string{{ConfirmYesButton, ConfirmNoButton}}
	values := [][]string{{"1", "0"}}
	_, err := b.InlineMenu(chatID, 0, prompt, labels, values, func(messageID int64, data string) {
		callback(data == "1")
		if err := b.RemoveMessage(chatID, messageID); err != nil {
			log.Println(err)
		}
	})
	return err
}
