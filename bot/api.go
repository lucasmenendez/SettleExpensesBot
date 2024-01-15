package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	updatesEndpointTemplate = "https://api.telegram.org/bot%s/getUpdates?offset=%d"
	messageEndpointTemplate = "https://api.telegram.org/bot%s/sendMessage"
)

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
				log.Printf("error getting updates: %s", err)
				log.Println("retrying in 5 seconds...")
				time.Sleep(5 * time.Second)
				continue
			}
			// read and parse the response body
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Printf("error reading update response body: %s", err)
				continue
			}
			res := &struct {
				Ok     bool      `json:"ok"`
				Result []*Update `json:"result"`
			}{}
			err = json.Unmarshal(body, res)
			if err != nil {
				log.Printf("error unmarshalling update response: %s", err)
				continue
			}
			// if the response is not ok, log the error and continue
			if !res.Ok {
				log.Printf("error response from telegram: %s", string(body))
				continue
			}
			// if there are no updates, and the last update was more than 5
			// minutes ago, sleep for 10 seconds to avoid spamming the api, if
			// there are updates, update the lastNonEmptyUpdate time
			if len(res.Result) == 0 {
				if time.Since(lastNonEmptyUpdate) > 5*time.Minute {
					log.Println("no updates for 5 minutes, sleeping for 10s...")
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

func (b *Bot) sendMessage(chatID int64, text string) error {
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
