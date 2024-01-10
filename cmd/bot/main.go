package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lucasmenendez/expensesbot/bot"
)

func main() {
	// parse env variables
	telegramToken := os.Getenv("TELEGRAM_TOKEN")
	if telegramToken == "" {
		log.Fatal("token, username and password are required")
	}
	snapshotPath := os.Getenv("SNAPSHOT_PATH")
	if snapshotPath == "" {
		snapshotPath = "./snapshot.json"
	}
	// create and start the bot
	b, err := bot.New(context.Background(), bot.BotConfig{
		Token:          telegramToken,
		SnapshotPath:   snapshotPath,
		ExpirationDays: 120,
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := b.Start(); err != nil {
		log.Fatal(err)
	}
	log.Printf("bot started at %s\n", time.Now().Format(time.RFC850))
	// wait until an interrupt received
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	fmt.Println("")
	log.Printf("received SIGTERM, exiting at %s\n", time.Now().Format(time.RFC850))
	// stop the bot
	b.Stop()
}
