package bot

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

var logger = log.New(os.Stdout, "BOT:", log.LstdFlags|log.Lshortfile)

type BotConfig struct {
	Token          string
	SnapshotPath   string
	ExpirationDays int
	AuthManager    Auth
}

type Bot struct {
	// auth manager
	Auth Auth
	// config
	token        string
	snapshotPath string
	// handlers
	handlers         map[string]CmdHandler
	adminHandlers    map[string]CmdHandler
	callbackHandlers map[int64]MenuCallback
	// context and sessions
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	sessions *sessions
	// third party apis
	updates    chan *Update
	lastUpdate int64
}

type CmdHandler func(*Bot, *Update) error
type MenuCallback func(int64, string)

func New(ctx context.Context, config BotConfig) *Bot {
	logger.Printf("bot started! admin users: %v\n", config.AuthManager.ListAdmins())
	// create a new context for the bot and initialize it
	botCtx, cancel := context.WithCancel(ctx)
	return &Bot{
		Auth:             config.AuthManager,
		token:            config.Token,
		snapshotPath:     config.SnapshotPath,
		handlers:         make(map[string]CmdHandler),
		adminHandlers:    make(map[string]CmdHandler),
		callbackHandlers: make(map[int64]MenuCallback),
		ctx:              botCtx,
		cancel:           cancel,
		wg:               sync.WaitGroup{},
		sessions:         initSessions(config.ExpirationDays),
		updates:          make(chan *Update),
		lastUpdate:       0,
	}
}

func (b *Bot) AddCommand(cmd string, handler CmdHandler) {
	b.handlers[cmd] = handler
}

func (b *Bot) AddAdminCommand(cmd string, handler CmdHandler) {
	b.adminHandlers[cmd] = handler
}

func (b *Bot) AddSessionImporter(importer DataImporter) {
	b.sessions.importer = importer
}

// Start method starts the bot and returns an error if something goes wrong.
// It starts a goroutine that listens to the updates from the bot and executes
// the corresponding handler only if the user is allowed to use it.
func (b *Bot) Start() error {
	// try to load the snapshot
	if err := b.tryToLoadSnapshot(); err != nil {
		return fmt.Errorf("error loading snapshot: %v", err)
	}
	// get updates from the bot in background
	b.listenForUpdates()
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		for {
			select {
			case <-b.ctx.Done():
				return
			case update := <-b.updates:
				switch {
				case update.IsCallback():
					go b.handleCallback(update)
				case update.IsCommand():
					go b.handleCommand(update)
				}
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
					logger.Printf("cleaned %d expired sessions\n", len(deleted))
					for _, id := range deleted {
						if err := b.SendMessage(id, "Your session has expired."); err != nil {
							logger.Println(err)
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
		logger.Println(err)
	}
}

func (b *Bot) GetSession(update *Update, initial Data) any {
	return b.sessions.getOrCreate(update.Message.Chat.ID, initial)
}

func (b *Bot) SendMessage(chatID int64, text string) error {
	return b.sendRequest(sendMessageMethod, map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	})
}

func (b *Bot) InlineMenu(chatID, messageID int64, text string, options map[string]string, callback MenuCallback) error {
	// create the inline keyboard
	callbackID := time.Now().Unix()
	var keyboard [][]map[string]string
	for text, data := range options {
		keyboard = append(keyboard, []map[string]string{
			{"text": text, "callback_data": encodeCallback(callbackID, data)},
		})
	}

	// if messageID is 0 then it is a new message, otherwise it is an edit
	if messageID == 0 {
		if err := b.sendRequest(sendMessageMethod, map[string]any{
			"chat_id":      chatID,
			"text":         text,
			"reply_markup": map[string]any{"inline_keyboard": keyboard},
		}); err != nil {
			return err
		}
	} else {
		if err := b.sendRequest(editMessageReplyMarkupMethod, map[string]any{
			"chat_id":      chatID,
			"message_id":   messageID,
			"reply_markup": map[string]any{"inline_keyboard": keyboard},
		}); err != nil {
			return err
		}
	}
	// add the callback handler
	b.callbackHandlers[callbackID] = callback
	return nil
}

func (c *Bot) RemoveInlineMenu(chatID, messageID int64) error {
	return c.sendRequest(removeMessageMethod, map[string]any{
		"chat_id":    chatID,
		"message_id": messageID,
	})
}
