package bot

import (
	"context"
	"log"
	"sync"

	tgapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	// config
	token string
	// users
	admins       map[int64]string
	allowedUsers sync.Map
	// handlers
	handlers      map[string]handler
	adminHandlers map[string]handler
	// context and sessions
	ctx      context.Context
	cancel   context.CancelFunc
	sessions *sessions
	// third party apis
	api *tgapi.BotAPI
}

func New(ctx context.Context, token string, admins map[int64]string) *Bot {
	// create a new context for the bot and initialize it
	botCtx, cancel := context.WithCancel(ctx)
	b := &Bot{
		token:        token,
		admins:       admins,
		allowedUsers: sync.Map{},
		ctx:          botCtx,
		cancel:       cancel,
		sessions:     initSessions(3),
	}
	for id, alias := range admins {
		b.allowedUsers.Store(id, alias)
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
	return b
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
	updateChan := b.api.GetUpdatesChan(u)
	// get updates from the bot in background
	go func() {
		for {
			select {
			case <-b.ctx.Done():
				b.api.StopReceivingUpdates()
				return
			case update := <-updateChan:
				if update.Message != nil || update.Message.IsCommand() {
					normalHandler, isNormalHandler := b.handlers[update.Message.Command()]
					adminHandler, isAdminHandler := b.adminHandlers[update.Message.Command()]
					// if the command is not registered, ignore it
					if !isNormalHandler && !isAdminHandler {
						continue
					}
					// if the command is registered, check if the user is allowed
					// to use it before executing it, no matter if it is an admin
					// command or not
					if isAdminHandler {
						if b.isAdmin(update.Message.From.ID) {
							if err := adminHandler(update); err != nil {
								log.Println(err)
							}
						}
					} else if isNormalHandler && b.isAllowed(update.Message.From.ID) {
						if err := normalHandler(update); err != nil {
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
}

// Wait method blocks until the bot is stopped.
func (b *Bot) Wait() {
	<-b.ctx.Done()
}

// AddUser method adds a user to the list of admin users.
func (b *Bot) isAdmin(userID int64) bool {
	_, exists := b.admins[userID]
	return exists
}

// AddUser method adds a user to the list of allowed users.
func (b *Bot) isAllowed(userID int64) bool {
	_, exists := b.allowedUsers.Load(userID)
	return exists
}
