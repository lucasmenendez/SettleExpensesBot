package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/lucasmenendez/expensesbot/bot"
	"github.com/lucasmenendez/expensesbot/settler"
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
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// parse env variables
	telegramToken := os.Getenv("TELEGRAM_TOKEN")
	if telegramToken == "" {
		log.Fatal("token, username and password are required")
	}
	snapshotPath := os.Getenv("SNAPSHOT_PATH")
	if snapshotPath == "" {
		snapshotPath = "./snapshot.json"
	}
	// parse admin users
	adminUsersIDs, err := parseIDs(os.Getenv("ADMIN_USER_IDS"))
	if err != nil {
		fmt.Println("invalid admin user ids: %w", err)
		return
	}
	// parse admin users
	adminUsersAliases := parseStrs(os.Getenv("ADMIN_USER_ALIASES"))
	if len(adminUsersAliases) == 0 {
		fmt.Println("no admin user aliases provided")
		return
	} else if len(adminUsersIDs) != len(adminUsersAliases) {
		fmt.Println("admin user ids and aliases must have the same length")
		return
	}
	admins := map[int64]string{}
	for i, id := range adminUsersIDs {
		admins[id] = adminUsersAliases[i]
	}
	// create and start the bot
	b := bot.New(context.Background(), bot.BotConfig{
		Token:          telegramToken,
		SnapshotPath:   snapshotPath,
		ExpirationDays: 120,
		AuthManager:    InitAuth(admins),
	})
	// register a function to import the settle data when the bot starts
	b.AddSessionImporter(func(encoded []byte) (bot.Data, error) {
		return settler.ImportSettle(encoded)
	})
	// register the commands
	b.AddCommand(START_CMD, handleStart)
	b.AddCommand(HELP_CMD, handleHelp)
	b.AddCommand(ADD_EXPENSE_CMD, handleAddExpense)
	b.AddCommand(ADD_FOR_EXPENSE_CMD, handleAddForExpense)
	b.AddCommand(LIST_EXPENSES_CMD, handleListExpenses)
	b.AddCommand(SUMMARY_CMD, handleSummary)
	// register the admin commands
	b.AddAdminCommand(ADD_USER_CMD, handleAddUser)
	b.AddAdminCommand(REMOVE_USER_CMD, handleRemoveUser)
	b.AddAdminCommand(LIST_USERS_CMD, handleListUsers)
	// start the bot
	if err := b.Start(); err != nil {
		log.Fatal(err)
	}
	// wait until an interrupt received
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	fmt.Println("")
	log.Printf("received SIGTERM, exiting at %s\n", time.Now().Format(time.RFC850))
	// stop the bot
	b.Stop()
}
