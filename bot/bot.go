package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var logger = log.New(os.Stdout, "", log.LstdFlags)

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
	handlers      map[string]CmdHandler
	adminHandlers map[string]CmdHandler
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

func New(ctx context.Context, config BotConfig) *Bot {
	logger.Printf("admin users: %v\n", config.AuthManager.ListAdmins())
	// create a new context for the bot and initialize it
	botCtx, cancel := context.WithCancel(ctx)
	return &Bot{
		Auth:          config.AuthManager,
		token:         config.Token,
		snapshotPath:  config.SnapshotPath,
		handlers:      make(map[string]CmdHandler),
		adminHandlers: make(map[string]CmdHandler),
		ctx:           botCtx,
		cancel:        cancel,
		wg:            sync.WaitGroup{},
		sessions:      initSessions(config.ExpirationDays),
		updates:       make(chan *Update),
		lastUpdate:    0,
	}
}

func (b *Bot) AddCommand(cmd string, handler CmdHandler) {
	b.handlers[cmd] = handler
}

func (b *Bot) AddAdminCommand(cmd string, handler CmdHandler) {
	b.adminHandlers[cmd] = handler
}

func (b *Bot) SendMessage(chatID int64, text string) error {
	// compose the url to send a message to the telegram api and encode the
	// request body
	url := fmt.Sprintf(messageEndpointTemplate, b.token)
	requestBody, err := json.Marshal(map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	})
	if err != nil {
		return err
	}
	// make the request and check if the response
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
	// read and parse the response body
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	messageResponse := struct {
		Ok bool `json:"ok"`
	}{}
	if err := json.Unmarshal(body, &messageResponse); err != nil {
		return err
	}
	// if the response is not ok, return an error, otherwise return nil
	if !messageResponse.Ok {
		return fmt.Errorf("failed to send message")
	}
	return nil
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

func (b *Bot) AddSessionImporter(importer DataImporter) {
	b.sessions.importer = importer
}
