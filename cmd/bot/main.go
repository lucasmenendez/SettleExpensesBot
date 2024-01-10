package main

import (
	"context"
	"os"

	"github.com/lucasmenendez/expensesbot/bot"
)

func main() {
	// parse env variables
	telegramToken := os.Getenv("TELEGRAM_TOKEN")

	if telegramToken == "" {
		panic("token, username and password are required")
	}

	// create and start the bot
	b := bot.New(context.Background(), telegramToken)

	if err := b.Start(); err != nil {
		panic(err)
	}
	defer b.Stop()
	// wait for ever to keep the bot running
	// TODO: add a signal handler to stop the bot
	b.Wait()
}
