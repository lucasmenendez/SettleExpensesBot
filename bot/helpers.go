package bot

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func (b *Bot) tryToLoadSnapshot() error {
	// check if the snapshot file exists
	if _, err := os.Stat(b.snapshotPath); err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			// create the file if it does not exist and return
			_, err := os.Create(b.snapshotPath)
			return err
		}
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
	b.sessions.importSnapshot(data)
	return nil
}

func (b *Bot) listenForCommands() {
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		// set the last update to the current time to listen for updates now
		// and check if there are any updates in the last 5 minutes from now
		lastNonEmptyUpdate := time.Now()
		for {
			// check if the context is done, if so, return
			select {
			case <-b.ctx.Done():
				if err := b.ctx.Err(); err != nil {
					return
				}
			default:
			}
			// compose the url to get updates from the telegram api and make the
			// request
			url := fmt.Sprintf(updatesEndpointTemplate, b.token, b.lastUpdate)
			resp, err := http.Get(url)
			// if something fails, log the error and retry after 5 seconds
			if err != nil {
				logger.Printf("error getting updates: %s", err)
				logger.Println("retrying in 5 seconds...")
				time.Sleep(5 * time.Second)
				continue
			}
			// read and parse the response body
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				logger.Printf("error reading update response body: %s", err)
				continue
			}
			res := &struct {
				Ok     bool      `json:"ok"`
				Result []*Update `json:"result"`
			}{}
			err = json.Unmarshal(body, res)
			if err != nil {
				logger.Printf("error unmarshalling update response: %s", err)
				continue
			}
			// if the response is not ok, log the error and continue
			if !res.Ok {
				logger.Printf("error response from telegram: %s", string(body))
				continue
			}
			// if there are no updates, and the last update was more than 5
			// minutes ago, sleep for 10 seconds to avoid spamming the api, if
			// there are updates, update the lastNonEmptyUpdate time
			if len(res.Result) == 0 {
				if time.Since(lastNonEmptyUpdate) > 5*time.Minute {
					logger.Println("no updates for 5 minutes, sleeping for 10s...")
					time.Sleep(10 * time.Second)
				}
				continue
			}
			lastNonEmptyUpdate = time.Now()
			// for each update, check if it is a command, if so, send it to the
			// updates channel
			for _, update := range res.Result {
				if update.UpdateID < b.lastUpdate {
					continue
				}
				b.lastUpdate = update.UpdateID + 1
				if update.IsCommand() {
					b.updates <- update
				}
			}
		}
	}()
}

func (b *Bot) handleCommand(update *Update) {
	cmd := update.Command()
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
		if b.Auth.IsAdmin(from.ID) {
			logger.Printf("admin command '%s' received in chat '%d' from '%s'",
				cmd, chatID, from.Username)
			if err := adminHandler(b, update); err != nil {
				logger.Println(err)
			}
		}
	} else if isNormalHandler && b.Auth.IsAllowed(from.ID) {
		logger.Printf("command '%s' received in chat '%d' from '%s'",
			cmd, chatID, from.Username)
		if err := normalHandler(b, update); err != nil {
			logger.Println(err)
		}
	}
}

func (b *Bot) saveSnapshot() error {
	snapshot, err := b.sessions.exportSnapshot()
	if err != nil {
		return err
	}
	// overwrite the snapshot file if it is not empty
	if len(snapshot) > 0 {
		if err := os.WriteFile(b.snapshotPath, snapshot, 0644); err != nil {
			return err
		}
	}
	return nil
}
