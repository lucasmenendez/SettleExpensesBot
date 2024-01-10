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
	b.settler.AddExpense(payer, participants, amount)

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
	b.settler.AddExpense(payer, participants, amount)

	text := fmt.Sprintf("Ok, %s paid %.2f for %s\n", payer, amount, strings.Join(participants, ", "))
	msg := tgapi.NewMessage(update.Message.Chat.ID, text)
	_, err = b.api.Send(msg)
	return err
}

// format: /list
func (b *Bot) handleListExpenses(update tgapi.Update) error {
	expenses := b.settler.Expenses()
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
	b.settler.RemoveExpense(id)
	msg := tgapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Expense %d removed\n", id))
	_, err = b.api.Send(msg)
	return err
}

// format: /summary
func (b *Bot) handleSettle(update tgapi.Update) error {
	transactions := b.settler.Settle()
	msg := tgapi.NewMessage(update.Message.Chat.ID, "Settlement:\n")
	for _, transaction := range transactions {
		msg.Text += fmt.Sprintf(" - %s must pay %.2f to %s\n", transaction.Payer, transaction.Amount, transaction.Participants[0])
	}
	_, err := b.api.Send(msg)
	return err
}

// format: /settle
func (b *Bot) handleSettleAndClean(update tgapi.Update) error {
	transactions := b.settler.SettleAndClean()
	msg := tgapi.NewMessage(update.Message.Chat.ID, "Settlement:\n")
	for _, transaction := range transactions {
		msg.Text += fmt.Sprintf(" - %s must pay %.2f to %s\n", transaction.Payer, transaction.Amount, transaction.Participants[0])
	}
	msg.Text += "Expenses cleaned"
	_, err := b.api.Send(msg)
	return err
}
