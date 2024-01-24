package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"sync"
	"time"
)

var logger *slog.Logger

func init() {
	// get the log output file from env variables
	writer := io.Writer(os.Stdout)
	if logFile := os.Getenv("LOG_FILE"); logFile != "" {
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			panic(fmt.Errorf("error opening log file: %v", err))
		}
		writer = file
	}

	logLevel := slog.LevelInfo
	logLevelEnv := os.Getenv("LOG_LEVEL")
	switch logLevelEnv {
	case "debug":
		logLevel = slog.LevelDebug
	case "error":
		logLevel = slog.LevelError
	}
	// set up the logger
	logger = slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{
		AddSource: true,
		Level:     &logLevel,
	}))
}

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
	handlers       map[string]CmdHandler
	adminHandlers  map[string]CmdHandler
	menuCallbacks  map[int64]MenuCallback
	replyCallbacks map[int64]ReplyCallback
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
type ReplyCallback func(int64, *Update)

func New(ctx context.Context, config BotConfig) *Bot {
	logger.Info("bot started", "admins", config.AuthManager.ListAdmins())
	// create a new context for the bot and initialize it
	botCtx, cancel := context.WithCancel(ctx)
	return &Bot{
		Auth:           config.AuthManager,
		token:          config.Token,
		snapshotPath:   config.SnapshotPath,
		handlers:       make(map[string]CmdHandler),
		adminHandlers:  make(map[string]CmdHandler),
		menuCallbacks:  make(map[int64]MenuCallback),
		replyCallbacks: make(map[int64]ReplyCallback),
		ctx:            botCtx,
		cancel:         cancel,
		wg:             sync.WaitGroup{},
		sessions:       initSessions(config.ExpirationDays),
		updates:        make(chan *Update),
		lastUpdate:     0,
	}
}

// AddCommand method adds a command to the bot. It receives the command and the
// handler function that will be executed when the command is received.
func (b *Bot) AddCommand(cmd string, handler CmdHandler) {
	b.handlers[cmd] = handler
}

// AddAdminCommand method adds an admin command to the bot. It receives the
// command and the handler function that will be executed when the command is
// received. Only the users that are registered as admins can execute this
// command.
func (b *Bot) AddAdminCommand(cmd string, handler CmdHandler) {
	b.adminHandlers[cmd] = handler
}

// AddSessionImporter method adds a function that will be executed when the bot
// starts to import the session data. It receives the encoded data and returns
// the decoded data and an error if something goes wrong.
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
				case update.IsReply():
					go b.handleReply(update)
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
					logger.Info("expired sessions cleaned",
						"expiredSessions", len(deleted))
					for _, id := range deleted {
						if _, err := b.SendMessage(id, 0, "Your session has expired."); err != nil {
							logger.Error("error sending expired session message", "error", err)
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
		logger.Error("error saving snapshot", "error", err)
	}
}

// GetSession method returns the session data for the given update using the
// chat id as the key. If the session does not exist, it creates a new one using
// the initial data provided.
func (b *Bot) GetSession(update *Update, initial Data) any {
	return b.sessions.getOrCreate(update.Message.Chat.ID, initial)
}

// SendMessage method sends a message to the given chat id. If messageID is 0
// then it is a new message, otherwise it is an edit.
func (b *Bot) SendMessage(chatID, messageID int64, text string) (int64, error) {
	method := sendMessageMethod
	params := map[string]any{
		"chat_id": chatID,
		"text":    text,
	}
	if messageID > 0 {
		method = editMessageTextMethod
		params["message_id"] = messageID
	}
	return b.sendRequest(method, params)
}

// InlineMenu method sends a message with an inline menu to the given chat id.
// If messageID is 0 then it is a new message, otherwise it is an edit. It
// receives a matrix of labels and values to create the menu. The callback
// function is executed when the user selects an option from the menu.
func (b *Bot) InlineMenu(chatID, messageID int64, text string, labels, values [][]string, callback MenuCallback) (int64, error) {
	// create the inline keyboard
	var keyboard [][]map[string]string
	for i, labelsRow := range labels {
		row := []map[string]string{}
		if len(labelsRow) != len(values[i]) {
			return 0, fmt.Errorf("labels and values must have the same length")
		}
		for j, label := range labelsRow {
			row = append(row, map[string]string{
				"text":          label,
				"callback_data": encodeCallback(messageID, values[i][j]),
			})
		}
		keyboard = append(keyboard, row)
	}
	params := map[string]any{
		"chat_id": chatID,
		"text":    text,
	}
	if len(keyboard) > 0 {
		params["reply_markup"] = map[string]any{"inline_keyboard": keyboard}
	}
	method := sendMessageMethod
	// if messageID is 0 then it is a new message, otherwise it is an edit
	if messageID > 0 {
		method = editMessageTextMethod
		params["message_id"] = messageID
	}
	menuMessageID, err := b.sendRequest(method, params)
	if err != nil {
		return menuMessageID, err
	}
	// add the callback handler
	if callback != nil {
		b.menuCallbacks[messageID] = callback
	}
	return menuMessageID, nil
}

// SendMessageToReply method sends a message to the given chat that enables the
// reply input field. If the user replies to the message, the callback function
// is executed. It also receives a placeholder for the input field.
func (b *Bot) SendMessageToReply(chatID int64, text, placeholder string, callback ReplyCallback) error {
	replyID, err := b.sendRequest(sendMessageMethod, map[string]any{
		"chat_id": chatID,
		"text":    text,
		"reply_markup": map[string]any{
			"force_reply":             true,
			"input_field_placeholder": placeholder,
			"selective":               true,
		},
	})
	if err != nil {
		return err
	}
	// add the callback handler
	if callback != nil {
		b.replyCallbacks[replyID] = callback
	}
	return nil
}

// RemoveMessage method removes a message from the given chat.
func (c *Bot) RemoveMessage(chatID, messageID int64) error {
	if _, err := c.sendRequest(removeMessageMethod, map[string]any{
		"chat_id":    chatID,
		"message_id": messageID,
	}); err != nil {
		return err
	}
	// delete the callback id from the map
	delete(c.menuCallbacks, messageID)
	return nil
}

// SendDocument method sends a document to the given chat. It receives the
// filename and the content of the file as a string. It returns an error if
// something goes wrong.
func (b *Bot) SendDocument(chatID int64, filename, content string) error {
	// create a temporary file with the content
	tmpFile, err := os.CreateTemp("", filename)
	if err != nil {
		return err
	}
	// remove the file when the function returns and write the content to it
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(content); err != nil {
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	// create the multipart form
	var buffer bytes.Buffer
	w := multipart.NewWriter(&buffer)
	// add the file field
	fw, err := w.CreateFormFile("document", filename)
	if err != nil {
		return err
	}
	// read the file content and write it to the multipart form
	fileContent, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return err
	}
	if _, err = fw.Write(fileContent); err != nil {
		return err
	}
	// add the chat id field
	if fw, err = w.CreateFormField("chat_id"); err != nil {
		return err
	}
	if _, err = fw.Write([]byte(fmt.Sprint(chatID))); err != nil {
		return err
	}
	// close the multipart form
	w.Close()
	// create the request
	endpoint := fmt.Sprintf(baseEndpointTemplate, b.token, sendDocumentMethod)
	req, err := http.NewRequest("POST", endpoint, &buffer)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	// create a new client and execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// check the status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

// DownloadFile method downloads a file from the given id and returns the file
// content as a byte array. It returns an error if something goes wrong.
func (b *Bot) DownloadFile(id string) ([]byte, error) {
	// create the request to get the file path
	filepathEndpoint := fmt.Sprintf(baseEndpointTemplate, b.token, getFileMethod)
	filepathEndpoint += fmt.Sprintf("?file_id=%s", id)
	filepathReq, err := http.Get(filepathEndpoint)
	if err != nil {
		return nil, err
	}
	defer filepathReq.Body.Close()
	// check the status code
	if filepathReq.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", filepathReq.StatusCode)
	}
	// parse the response
	response := struct {
		Ok     bool `json:"ok"`
		Result struct {
			FilePath string `json:"file_path"`
		} `json:"result"`
	}{}
	if err := json.NewDecoder(filepathReq.Body).Decode(&response); err != nil {
		return nil, err
	}
	if !response.Ok {
		return nil, fmt.Errorf("error downloading file")
	}
	// create the request to download the file
	fileEndpoint := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", b.token, response.Result.FilePath)
	fileReq, err := http.Get(fileEndpoint)
	if err != nil {
		return nil, err
	}
	defer fileReq.Body.Close()
	// check the status code
	if fileReq.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", fileReq.StatusCode)
	}
	// read the file content
	fileBody, err := io.ReadAll(fileReq.Body)
	if err != nil {
		return nil, err
	}
	return fileBody, nil
}
