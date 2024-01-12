package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	tgapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var publicCommands = map[string]string{
	HELP_CMD:            HELP_DESC,
	ADD_EXPENSE_CMD:     ADD_EXPENSE_DESC,
	ADD_FOR_EXPENSE_CMD: ADD_FOR_EXPENSE_DESC,
	LIST_EXPENSES_CMD:   LIST_EXPENSES_DESC,
	REMOVE_EXPENSE_CMD:  REMOVE_EXPENSE_DESC,
	SUMMARY_CMD:         SUMMARY_DESC,
	SETTLE_CMD:          SETTLE_DESC,
}

type handler func(tgapi.Update) error

// format: /start
func (b *Bot) handleStart(update tgapi.Update) error {
	return b.sendMessage(update.Message.Chat.ID, WelcomeMessage)
}

// format: /help
func (b *Bot) handleHelp(update tgapi.Update) error {
	texts := []string{HelpHeader}
	for cmd, desc := range publicCommands {
		texts = append(texts, fmt.Sprintf(HelperCommandTemplate, cmd, desc))
	}
	return b.sendMessage(update.Message.Chat.ID, strings.Join(texts, "\n"))
}

// format: /add @participant1,@participant2 12.5
func (b *Bot) handleAddExpense(update tgapi.Update) error {
	args := update.Message.CommandArguments()
	parts := strings.Split(args, " ")
	if len(parts) != 2 {
		return b.sendMessage(update.Message.Chat.ID, ErrAddInvalidArguments)
	}
	// parse the participants
	participants := strings.Split(parts[0], ",")
	if len(participants) < 1 {
		return b.sendMessage(update.Message.Chat.ID, ErrAddInvalidArguments)
	}
	// parse the amount
	amount, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return b.sendMessage(update.Message.Chat.ID, ErrAddInvalidArguments)
	}
	payer := fmt.Sprintf("@%s", update.Message.From.UserName)
	settler := b.sessions.getOrCreate(update.Message.Chat.ID)
	settler.AddExpense(payer, participants, amount)
	// send the message
	msg := fmt.Sprintf(AddSuccessTemplate, payer, amount, strings.Join(participants, ", "))
	if err := b.sendMessage(update.Message.Chat.ID, msg); err != nil {
		return b.sendMessage(update.Message.Chat.ID, fmt.Sprintf(ErrProcesingRequestTemplate, err))
	}
	return nil
}

// format: /addfor @payer @participant1,@participant2 12.5
func (b *Bot) handleAddForExpense(update tgapi.Update) error {
	args := update.Message.CommandArguments()
	parts := strings.Split(args, " ")
	if len(parts) != 3 {
		return b.sendMessage(update.Message.Chat.ID, ErrAddForInvalidArguments)
	}
	// parse the payer
	payer := parts[0]
	if payer == "" {
		return b.sendMessage(update.Message.Chat.ID, ErrAddForInvalidArguments)
	}
	// parse the participants
	participants := strings.Split(parts[1], ",")
	if len(participants) < 1 {
		return b.sendMessage(update.Message.Chat.ID, ErrAddForInvalidArguments)
	}
	// parse the amount
	amount, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return b.sendMessage(update.Message.Chat.ID, ErrAddForInvalidArguments)
	}
	// get the settler of the chat and add the expense
	settler := b.sessions.getOrCreate(update.Message.Chat.ID)
	settler.AddExpense(payer, participants, amount)
	// send the message
	msg := fmt.Sprintf(AddSuccessTemplate, payer, amount, strings.Join(participants, ", "))
	if err = b.sendMessage(update.Message.Chat.ID, msg); err != nil {
		return b.sendMessage(update.Message.Chat.ID, fmt.Sprintf(ErrProcesingRequestTemplate, err))
	}
	return nil
}

// format: /list
func (b *Bot) handleListExpenses(update tgapi.Update) error {
	// get the settler of the chat and list the expenses
	settler := b.sessions.getOrCreate(update.Message.Chat.ID)
	expenses, ids := settler.Expenses()
	// if there are no expenses, send an error message
	if len(expenses) == 0 {
		return b.sendMessage(update.Message.Chat.ID, ErrNoExpenses)
	}
	// compose and send the message
	texts := []string{ListExpensesHeader}
	for i, expense := range expenses {
		texts = append(texts, fmt.Sprintf(ExpenseItemTemplate,
			ids[i],
			expense.Payer,
			expense.Amount,
			strings.Join(expense.Participants, ", "),
		))
	}
	return b.sendMessage(update.Message.Chat.ID, texts...)
}

// format: /remove 1
func (b *Bot) handleRemoveExpense(update tgapi.Update) error {
	args := update.Message.CommandArguments()
	// parse the expense ID
	id, err := strconv.Atoi(args)
	if err != nil {
		return b.sendMessage(update.Message.Chat.ID, ErrRemoveInvalidArguments)
	}
	// get the settler of the chat and remove the expense
	settler := b.sessions.getOrCreate(update.Message.Chat.ID)
	settler.RemoveExpense(id)
	// send the message
	msg := fmt.Sprintf(RemoveSuccessTemplate, id)
	if err := b.sendMessage(update.Message.Chat.ID, msg); err != nil {
		return b.sendMessage(update.Message.Chat.ID, fmt.Sprintf(ErrProcesingRequestTemplate, err))
	}
	return nil
}

// format: /summary
func (b *Bot) handleSummary(update tgapi.Update) error {
	// get the settler of the chat, the balances of the participants and the
	// list of transactions to settle the expenses
	settler := b.sessions.getOrCreate(update.Message.Chat.ID)
	balances := settler.Balances()
	transactions := settler.Settle(false)
	// if there are no transactions, send an error message
	if len(transactions) == 0 {
		return b.sendMessage(update.Message.Chat.ID, ErrNoExpenses)
	}
	// compose and send the message
	texts := []string{BalancesHeader}
	for participant, balance := range balances {
		texts = append(texts, fmt.Sprintf(BalanceItemTemplate, participant, balance))
	}
	texts = append(texts, SettleHeader)
	for _, transaction := range transactions {
		texts = append(texts, fmt.Sprintf(SettleItemTemplate,
			transaction.Payer,
			transaction.Amount,
			transaction.Participants[0],
		))
	}
	return b.sendMessage(update.Message.Chat.ID, texts...)
}

// format: /settle
func (b *Bot) handleSettle(update tgapi.Update) error {
	// get the settler of the chat, the balances of the participants and the
	// list of transactions to settle the expenses
	settler := b.sessions.getOrCreate(update.Message.Chat.ID)
	balances := settler.Balances()
	transactions := settler.Settle(true)
	// if there are no transactions, send an error message
	if len(transactions) == 0 {
		return b.sendMessage(update.Message.Chat.ID, ErrNoExpenses)
	}
	// compose and send the message
	texts := []string{BalancesHeader}
	for participant, balance := range balances {
		texts = append(texts, fmt.Sprintf(BalanceItemTemplate, participant, balance))
	}
	texts = append(texts, SettleHeader)
	for _, transaction := range transactions {
		texts = append(texts, fmt.Sprintf(SettleItemTemplate,
			transaction.Payer,
			transaction.Amount,
			transaction.Participants[0],
		))
	}
	texts = append(texts, SettleBottomMessage)
	return b.sendMessage(update.Message.Chat.ID, texts...)
}

// format: /adduser 123456789 alias
func (b *Bot) handleAddUser(update tgapi.Update) error {
	args := strings.Split(update.Message.CommandArguments(), " ")
	if len(args) != 2 {
		return b.sendMessage(update.Message.Chat.ID, ErrInvalidArguments)
	}
	userID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return b.sendMessage(update.Message.Chat.ID, ErrInvalidArguments)
	}
	userAlias := args[1]
	if err := b.auth.AddAllowedUser(userID, userAlias); err != nil {
		log.Printf("error adding user: %s", err)
		return b.sendMessage(update.Message.Chat.ID, ErrInternalProcess)
	}
	return b.sendMessage(update.Message.Chat.ID, SuccessInternalMessage)
}

// format: /removeuser 123456789
func (b *Bot) handleRemoveUser(update tgapi.Update) error {
	userID, err := strconv.ParseInt(update.Message.CommandArguments(), 10, 64)
	if err != nil {
		return b.sendMessage(update.Message.Chat.ID, ErrInvalidArguments)
	}
	if found := b.auth.RemoveAllowedUser(userID); found {
		return b.sendMessage(update.Message.Chat.ID, SuccessInternalMessage)
	}
	log.Println("user not found")
	return b.sendMessage(update.Message.Chat.ID, ErrInternalProcess)
}

// format: /listusers
func (b *Bot) handleListUsers(update tgapi.Update) error {
	users := b.auth.ListAllowedUsers()
	if len(users) == 0 {
		return b.sendMessage(update.Message.Chat.ID, ErrInternalProcess)
	}
	texts := []string{UserListHeader}
	for userID, userAlias := range users {
		texts = append(texts, fmt.Sprintf(UserItemTemplate, userAlias, userID))
	}
	return b.sendMessage(update.Message.Chat.ID, texts...)
}
