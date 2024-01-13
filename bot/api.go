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

type UpdateResponse struct {
	Ok     bool      `json:"ok"`
	Result []*Update `json:"result"`
}

func (b *Bot) listenForCommands() {
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		for {
			select {
			case <-b.ctx.Done():
				if err := b.ctx.Err(); err != nil {
					log.Println(err)
					return
				}
			default:
			}
			url := fmt.Sprintf(updatesEndpointTemplate, b.token, b.lastUpdate)
			resp, err := http.Get(url)
			if err != nil {
				log.Printf("error getting updates: %s", err)
				log.Println("retrying in 5 seconds...")
				time.Sleep(5 * time.Second)
				continue
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Printf("error reading update response body: %s", err)
				continue
			}

			res := &UpdateResponse{}
			err = json.Unmarshal(body, res)
			if err != nil {
				log.Printf("error unmarshalling update response: %s", err)
				continue
			}
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
	url := fmt.Sprintf(messageEndpointTemplate, b.token)
	requestBody, err := json.Marshal(map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	})
	if err != nil {
		return err
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
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
	if !messageResponse.Ok {
		return fmt.Errorf("failed to send message")
	}
	return nil
}
