package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/lucasmenendez/expensesbot/bot"
)

func parseIDs(ids string) ([]int64, error) {
	ids = strings.TrimSpace(ids)
	var parsedIDs []int64
	for _, strID := range strings.Split(ids, ",") {
		intID, err := strconv.ParseInt(strID, 10, 64)
		if err != nil {
			return nil, err
		}
		parsedIDs = append(parsedIDs, intID)
	}
	return parsedIDs, nil
}

func main() {
	// parse env variables
	telegramToken := os.Getenv("TELEGRAM_TOKEN")
	if telegramToken == "" {
		panic("token, username and password are required")
	}
	// parse admin users
	adminUsers, err := parseIDs(os.Getenv("ADMIN_USERS"))
	if err != nil {
		panic(err)
	}
	log.Println(adminUsers)

	// create and start the bot
	b := bot.New(context.Background(), telegramToken, adminUsers)

	if err := b.Start(); err != nil {
		panic(err)
	}
	defer b.Stop()
	// wait for ever to keep the bot running
	// TODO: add a signal handler to stop the bot
	b.Wait()
}
