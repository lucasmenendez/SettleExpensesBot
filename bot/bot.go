package bot

import (
	"context"
	"log"

	tgapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/lucasmenendez/expensesbot/settler"
)

type Bot struct {
	token    string
	handlers map[string]handler
	ctx      context.Context
	cancel   context.CancelFunc
	// third party structs
	settler *settler.Settler
	api     *tgapi.BotAPI
}

func New(ctx context.Context, token string) *Bot {
	botCtx, cancel := context.WithCancel(ctx)
	b := &Bot{
		token:   token,
		ctx:     botCtx,
		cancel:  cancel,
		settler: settler.NewSettler(),
	}
	b.handlers = map[string]handler{
		ADD_EXPENSE_CMD:      b.handleAddExpense,
		ADD_FOR_EXPENSE_CMD:  b.handleAddForExpense,
		LIST_EXPENSES_CMD:    b.handleListExpenses,
		REMOVE_EXPENSE_CMD:   b.handleRemoveExpense,
		SETTLE_CMD:           b.handleSettle,
		SETTLE_AND_CLEAN_CMD: b.handleSettleAndClean,
	}
	return b
}

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
					switch update.Message.Command() {
					case ADD_EXPENSE_CMD:
						if err := b.handleAddExpense(update); err != nil {
							log.Println(err)
						}
					case ADD_FOR_EXPENSE_CMD:
						if err := b.handleAddForExpense(update); err != nil {
							log.Println(err)
						}
					case LIST_EXPENSES_CMD:
						if err := b.handleListExpenses(update); err != nil {
							log.Println(err)
						}
					case REMOVE_EXPENSE_CMD:
						if err := b.handleRemoveExpense(update); err != nil {
							log.Println(err)
						}
					case SETTLE_CMD:
						if err := b.handleSettle(update); err != nil {
							log.Println(err)
						}
					case SETTLE_AND_CLEAN_CMD:
						if err := b.handleSettleAndClean(update); err != nil {
							log.Println(err)
						}
					}
				}
			}
		}
	}()
	return nil
}

func (b *Bot) Stop() {
	b.cancel()
}

func (b *Bot) Wait() {
	<-b.ctx.Done()
}
