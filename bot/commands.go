package bot

import (
	"fmt"
	"strconv"
	"strings"

	tgapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	ADD_EXPENSE_CMD      = "add"
	ADD_FOR_EXPENSE_CMD  = "addfor"
	LIST_EXPENSES_CMD    = "list"
	REMOVE_EXPENSE_CMD   = "remove"
	SETTLE_CMD           = "summary"
	SETTLE_AND_CLEAN_CMD = "settle"
	ADD_USER_CMD         = "adduser"
	REMOVE_USER_CMD      = "removeuser"
	LIST_USERS_CMD       = "listusers"
)

type handler func(tgapi.Update) error

// format: /add @participant1,@participant2 12.5
func (b *Bot) handleAddExpense(update tgapi.Update) error {
	args := update.Message.CommandArguments()
	parts := strings.Split(args, " ")
	if len(parts) != 2 {
		return fmt.Errorf("invalid arguments")
	}

	participants := strings.Split(parts[0], ",")
	if len(participants) < 1 {
		return fmt.Errorf("invalid participants")
	}
	amount, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return fmt.Errorf("invalid amount")
	}
	payer := fmt.Sprintf("@%s", update.Message.From.UserName)
	settler := b.sessions.getOrCreate(update.Message.Chat.ID)
	settler.AddExpense(payer, participants, amount)

	text := fmt.Sprintf("Ok, %s paid %.2f for %s\n", payer, amount, strings.Join(participants, ", "))
	msg := tgapi.NewMessage(update.Message.Chat.ID, text)
	_, err = b.api.Send(msg)
	return err
}

// format: /addfor @payer @participant1,@participant2 12.5
func (b *Bot) handleAddForExpense(update tgapi.Update) error {
	args := update.Message.CommandArguments()
	parts := strings.Split(args, " ")
	if len(parts) != 3 {
		return fmt.Errorf("invalid arguments")
	}
	payer := parts[0]
	participants := strings.Split(parts[1], ",")
	if len(participants) < 1 {
		return fmt.Errorf("invalid participants")
	}
	amount, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return fmt.Errorf("invalid amount")
	}
	settler := b.sessions.getOrCreate(update.Message.Chat.ID)
	settler.AddExpense(payer, participants, amount)

	text := fmt.Sprintf("Ok, %s paid %.2f for %s\n", payer, amount, strings.Join(participants, ", "))
	msg := tgapi.NewMessage(update.Message.Chat.ID, text)
	_, err = b.api.Send(msg)
	return err
}

// format: /list
func (b *Bot) handleListExpenses(update tgapi.Update) error {
	settler := b.sessions.getOrCreate(update.Message.Chat.ID)
	expenses := settler.Expenses()
	if len(expenses) == 0 {
		msg := tgapi.NewMessage(update.Message.Chat.ID, "No expenses yet\n")
		_, err := b.api.Send(msg)
		return err
	}
	msg := tgapi.NewMessage(update.Message.Chat.ID, "Expenses:\n")
	for id, expense := range expenses {
		msg.Text += fmt.Sprintf(" %d. %s paid %.2f for %s\n", id, expense.Payer, expense.Amount, strings.Join(expense.Participants, ", "))
	}
	_, err := b.api.Send(msg)
	return err
}

// format: /remove 1
func (b *Bot) handleRemoveExpense(update tgapi.Update) error {
	args := update.Message.CommandArguments()
	id, err := strconv.Atoi(args)
	if err != nil {
		return fmt.Errorf("invalid id")
	}
	settler := b.sessions.getOrCreate(update.Message.Chat.ID)
	settler.RemoveExpense(id)
	msg := tgapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Expense %d removed\n", id))
	_, err = b.api.Send(msg)
	return err
}

// format: /summary
func (b *Bot) handleSettle(update tgapi.Update) error {
	settler := b.sessions.getOrCreate(update.Message.Chat.ID)
	transactions := settler.Settle()
	msg := tgapi.NewMessage(update.Message.Chat.ID, "Settlement:\n")
	for _, transaction := range transactions {
		msg.Text += fmt.Sprintf(" - %s must pay %.2f to %s\n", transaction.Payer, transaction.Amount, transaction.Participants[0])
	}
	_, err := b.api.Send(msg)
	return err
}

// format: /settle
func (b *Bot) handleSettleAndClean(update tgapi.Update) error {
	settler := b.sessions.getOrCreate(update.Message.Chat.ID)
	transactions := settler.SettleAndClean()
	msg := tgapi.NewMessage(update.Message.Chat.ID, "Settlement:\n")
	for _, transaction := range transactions {
		msg.Text += fmt.Sprintf(" - %s must pay %.2f to %s\n", transaction.Payer, transaction.Amount, transaction.Participants[0])
	}
	msg.Text += "Expenses cleaned"
	_, err := b.api.Send(msg)
	return err
}

// format: /adduser 123456789 alias
func (b *Bot) handleAddUser(update tgapi.Update) error {
	args := strings.Split(update.Message.CommandArguments(), " ")
	if len(args) != 2 {
		return fmt.Errorf("invalid arguments")
	}
	userID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid user id")
	}
	if userAlias, exists := b.allowedUsers.Load(userID); exists {
		sAlias, ok := userAlias.(string)
		if !ok {
			return fmt.Errorf("invalid user alias")
		}
		msg := tgapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("User %s (%d) already added\n", sAlias, userID))
		_, err := b.api.Send(msg)
		return err
	}
	userAlias := args[1]
	b.allowedUsers.Store(userID, userAlias)
	msg := tgapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("User %s (%d) added\n", userAlias, userID))
	_, err = b.api.Send(msg)
	return err
}

// format: /removeuser 123456789
func (b *Bot) handleRemoveUser(update tgapi.Update) error {
	userID, err := strconv.ParseInt(update.Message.CommandArguments(), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid user id")
	}

	if userAlias, exists := b.allowedUsers.Load(userID); exists {
		sAlias, ok := userAlias.(string)
		if !ok {
			return fmt.Errorf("invalid user alias")
		}
		msg := tgapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("User %s (%d) removed\n", sAlias, userID))
		if _, err := b.api.Send(msg); err != nil {
			return err
		}
		b.allowedUsers.Delete(userID)
		return nil
	}
	msg := tgapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("User %d not found\n", userID))
	_, err = b.api.Send(msg)
	return err
}

// format: /listusers
func (b *Bot) handleListUsers(update tgapi.Update) error {
	users := map[int64]string{}
	b.allowedUsers.Range(func(iUserID, iAlias interface{}) bool {
		userID, ok := iUserID.(int64)
		if !ok {
			return false
		}
		sAlias, ok := iAlias.(string)
		if !ok {
			return false
		}
		users[userID] = sAlias
		return true
	})
	if len(users) == 0 {
		msg := tgapi.NewMessage(update.Message.Chat.ID, "No users allowed yet\n")
		_, err := b.api.Send(msg)
		return err
	}
	msg := tgapi.NewMessage(update.Message.Chat.ID, "Users allowed:\n")
	for userID, userAlias := range users {
		msg.Text += fmt.Sprintf(" - %s (%d)\n", userAlias, userID)
	}
	_, err := b.api.Send(msg)
	return err
}
