package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	tgapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BotConfig struct {
	Token          string
	SnapshotPath   string
	ExpirationDays int
}

type Bot struct {
	// config
	token        string
	snapshotPath string
	auth         *Auth
	// handlers
	handlers      map[string]handler
	adminHandlers map[string]handler
	// context and sessions
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	sessions *sessions
	// third party apis
	api *tgapi.BotAPI
}

func New(ctx context.Context, config BotConfig) (*Bot, error) {
	// init auth
	auth, err := InitAuth()
	if err != nil {
		return nil, err
	}
	log.Printf("admin users: %v\n", auth.admins)
	// create a new context for the bot and initialize it
	botCtx, cancel := context.WithCancel(ctx)
	b := &Bot{
		token:        config.Token,
		snapshotPath: config.SnapshotPath,
		auth:         auth,
		ctx:          botCtx,
		cancel:       cancel,
		wg:           sync.WaitGroup{},
		sessions:     initSessions(config.ExpirationDays),
	}
	// initialize the handlers and admin handlers and register the bot
	b.handlers = map[string]handler{
		ADD_EXPENSE_CMD:      b.handleAddExpense,
		ADD_FOR_EXPENSE_CMD:  b.handleAddForExpense,
		LIST_EXPENSES_CMD:    b.handleListExpenses,
		REMOVE_EXPENSE_CMD:   b.handleRemoveExpense,
		SETTLE_CMD:           b.handleSettle,
		SETTLE_AND_CLEAN_CMD: b.handleSettleAndClean,
	}
	b.adminHandlers = map[string]handler{
		ADD_USER_CMD:    b.handleAddUser,
		REMOVE_USER_CMD: b.handleRemoveUser,
		LIST_USERS_CMD:  b.handleListUsers,
	}
	// try to load the snapshot
	if err := b.tryToLoadSnapshot(); err != nil {
		return nil, fmt.Errorf("error loading snapshot: %v", err)
	}
	return b, nil
}

// Start method starts the bot and returns an error if something goes wrong.
// It starts a goroutine that listens to the updates from the bot and executes
// the corresponding handler only if the user is allowed to use it.
func (b *Bot) Start() error {
	// init bot api and attach it to the current bot instance
	var err error
	b.api, err = tgapi.NewBotAPI(b.token)
	if err != nil {
		log.Fatal(err)
	}
	// config the updates channel
	u := tgapi.NewUpdate(0)
	u.Timeout = 60
	// get updates from the bot in background
	updates := b.api.GetUpdatesChan(u)
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		for {
			select {
			case <-b.ctx.Done():
				b.api.StopReceivingUpdates()
				return
			case update := <-updates:
				if update.Message == nil || !update.Message.IsCommand() {
					continue
				}
				go b.handleCommand(update)
			}
		}
	}()
	return nil
}

// Stop method stops the bot.
func (b *Bot) Stop() {
	b.cancel()
	b.wg.Wait()
	// save the snapshot
	if err := b.saveSnapshot(); err != nil {
		log.Println(err)
	}
}

// Wait method blocks until the bot is stopped.
func (b *Bot) Wait() {
	<-b.ctx.Done()
}

func (b *Bot) tryToLoadSnapshot() error {
	// check if the snapshot file exists
	if _, err := os.Stat(b.snapshotPath); os.IsNotExist(err) {
		// create the file if it does not exist and return
		_, err := os.Create(b.snapshotPath)
		return err
	}
	// load the snapshot file
	data, err := os.ReadFile(b.snapshotPath)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	// try to parse the snapshot
	var snapshot snapshotData
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return err
	}
	// import the snapshot
	for id, username := range snapshot.AllowedUsers {
		b.auth.AddAllowedUser(id, username)
	}
	b.sessions.importSnapshot(snapshot.Sessions)
	return nil
}

func (b *Bot) saveSnapshot() error {
	// export the snapshot
	snapshot := snapshotData{
		Sessions:     b.sessions.exportSnapshot(),
		AllowedUsers: b.auth.ListAllowedUsers(),
	}
	data, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	// overwrite the snapshot file
	if err := os.WriteFile(b.snapshotPath, data, 0644); err != nil {
		return err
	}
	return nil
}

func (b *Bot) handleCommand(update tgapi.Update) {
	cmd := update.Message.Command()
	// check if the command is registered
	normalHandler, isNormalHandler := b.handlers[cmd]
	adminHandler, isAdminHandler := b.adminHandlers[cmd]
	// if the command is not registered, ignore it
	if !isNormalHandler && !isAdminHandler {
		return
	}
	// if the command is registered, check if the user is allowed
	// to use it before executing it, no matter if it is an admin
	// command or not
	from := update.Message.From
	chatID := update.Message.Chat.ID
	if isAdminHandler {
		if b.auth.IsAdmin(from.ID) {
			log.Printf("admin command '%s' received in chat '%d' from '%s'",
				cmd, chatID, from.UserName)
			if err := adminHandler(update); err != nil {
				log.Println(err)
			}
		}
	} else if isNormalHandler && b.auth.IsAllowed(from.ID) {
		log.Printf("command '%s' received in chat '%d' from '%s'",
			cmd, chatID, from.UserName)
		if err := normalHandler(update); err != nil {
			log.Println(err)
		}
	}
}
