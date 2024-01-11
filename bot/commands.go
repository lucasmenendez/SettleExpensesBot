package bot

import (
	"fmt"
	"strconv"
	"strings"

	tgapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	HELP_CMD             = "help"
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

var publicCommands = map[string]string{
	HELP_CMD:             "Show this help.",
	ADD_EXPENSE_CMD:      "Add an expense for you. Format: /add @participant1,@participant2 12.5",
	ADD_FOR_EXPENSE_CMD:  "Add an expense for another user. Format: /addfor @payer @participant1,@participant2 12.5",
	LIST_EXPENSES_CMD:    "List all the expenses with their IDs.",
	REMOVE_EXPENSE_CMD:   "Remove an expense by its ID. Format: /remove 1",
	SETTLE_CMD:           "Show the settlement.",
	SETTLE_AND_CLEAN_CMD: "Show the settlement and clean the expenses.",
}

type handler func(tgapi.Update) error

// format: /help
func (b *Bot) handleHelp(update tgapi.Update) error {
	msg := tgapi.NewMessage(update.Message.Chat.ID, "Available commands:\n")
	for cmd, desc := range publicCommands {
		msg.Text += fmt.Sprintf(" /%s: %s\n", cmd, desc)
	}
	_, err := b.api.Send(msg)
	return err
}

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
	if len(transactions) == 0 {
		msg := tgapi.NewMessage(update.Message.Chat.ID, "No expenses yet\n")
		_, err := b.api.Send(msg)
		return err
	}
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
	userAlias := args[1]
	if err := b.auth.AddAllowedUser(userID, userAlias); err != nil {
		return err
	}
	msg := tgapi.NewMessage(update.Message.Chat.ID, "User added")
	_, err = b.api.Send(msg)
	return err
}

// format: /removeuser 123456789
func (b *Bot) handleRemoveUser(update tgapi.Update) error {
	userID, err := strconv.ParseInt(update.Message.CommandArguments(), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid user id")
	}
	if found := b.auth.RemoveAllowedUser(userID); found {
		_, err := b.api.Send(tgapi.NewMessage(update.Message.Chat.ID, "User removed"))
		return err
	}
	msg := tgapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("User %d not found\n", userID))
	_, err = b.api.Send(msg)
	return err
}

// format: /listusers
func (b *Bot) handleListUsers(update tgapi.Update) error {
	users := b.auth.ListAllowedUsers()
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
