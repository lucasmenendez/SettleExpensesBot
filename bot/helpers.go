package bot

import (
	"encoding/json"
	"log"
	"os"
)

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
		if b.auth.IsAdmin(from.ID) {
			log.Printf("admin command '%s' received in chat '%d' from '%s'",
				cmd, chatID, from.Username)
			if err := adminHandler(update); err != nil {
				log.Println(err)
			}
		}
	} else if isNormalHandler && b.auth.IsAllowed(from.ID) {
		log.Printf("command '%s' received in chat '%d' from '%s'",
			cmd, chatID, from.Username)
		if err := normalHandler(update); err != nil {
			log.Println(err)
		}
	}
}
