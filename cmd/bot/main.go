package main

import (
	"context"
	"os"
	"strconv"
	"strings"

	"github.com/lucasmenendez/expensesbot/bot"
)

func parseStrs(strs string) []string {
	return strings.Split(strings.TrimSpace(strs), ",")
}

func parseIDs(ids string) ([]int64, error) {
	var parsedIDs []int64
	for _, strID := range parseStrs(ids) {
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
	adminUsersIDs, err := parseIDs(os.Getenv("ADMIN_USER_IDS"))
	if err != nil {
		panic(err)
	}
	// parse admin users
	adminUsersAliases := parseStrs(os.Getenv("ADMIN_USER_ALIASES"))
	if len(adminUsersAliases) == 0 {
		panic("at least one admin user alias is required")
	} else if len(adminUsersIDs) != len(adminUsersAliases) {
		panic("admin user ids and aliases must have the same length")
	}
	adminUsers := make(map[int64]string)
	for i, id := range adminUsersIDs {
		adminUsers[id] = adminUsersAliases[i]
	}
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
