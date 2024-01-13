package bot

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
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
	updates    chan *Update
	lastUpdate int64
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
		updates:      make(chan *Update),
		lastUpdate:   0,
	}
	// initialize the handlers and admin handlers and register the bot
	b.handlers = map[string]handler{
		START_CMD:           b.handleStart,
		HELP_CMD:            b.handleHelp,
		ADD_EXPENSE_CMD:     b.handleAddExpense,
		ADD_FOR_EXPENSE_CMD: b.handleAddForExpense,
		LIST_EXPENSES_CMD:   b.handleListExpenses,
		REMOVE_EXPENSE_CMD:  b.handleRemoveExpense,
		SUMMARY_CMD:         b.handleSummary,
		SETTLE_CMD:          b.handleSettle,
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
	// get updates from the bot in background
	b.listenForCommands()
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		for {
			select {
			case <-b.ctx.Done():
				return
			case update := <-b.updates:
				if update.Message == nil || !update.IsCommand() {
					continue
				}
				go b.handleCommand(update)
			}
		}
	}()
	// clean expired sessions in background
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		// create a forever loop that runs every 24 hours using a ticker to
		// clean expired sessions
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-b.ctx.Done():
				return
			case <-ticker.C:
				deleted := b.sessions.cleanExpired()
				if len(deleted) > 0 {
					log.Printf("cleaned %d expired sessions\n", len(deleted))
					for _, id := range deleted {
						if err := b.sendMessage(id, "Your session has expired."); err != nil {
							log.Println(err)
						}
					}
				}
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
