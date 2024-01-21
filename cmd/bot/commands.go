package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/lucasmenendez/expensesbot/bot"
	"github.com/lucasmenendez/expensesbot/settler"
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

// format: /start
func handleStart(b *bot.Bot, update *bot.Update) error {
	return b.SendMessage(update.Message.Chat.ID, WelcomeMessage)
}

// format: /help
func handleHelp(b *bot.Bot, update *bot.Update) error {
	texts := []string{HelpHeader}
	for cmd, desc := range publicCommands {
		texts = append(texts, fmt.Sprintf(HelperCommandTemplate, cmd, desc))
	}
	return b.SendMessage(update.Message.Chat.ID, strings.Join(texts, "\n"))
}

// format: /add @participant1,@participant2 12.5
func handleAddExpense(b *bot.Bot, update *bot.Update) error {
	args := update.CommandArgs()
	if len(args) != 2 {
		return b.SendMessage(update.Message.Chat.ID, ErrAddInvalidArguments)
	}
	// parse the participants
	participants := strings.Split(args[0], ",")
	if len(participants) < 1 {
		return b.SendMessage(update.Message.Chat.ID, ErrAddInvalidArguments)
	}
	// parse the amount
	amount, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return b.SendMessage(update.Message.Chat.ID, ErrAddInvalidArguments)
	}
	payer := fmt.Sprintf("@%s", update.Message.From.Username)
	iSettler := b.GetSession(update, settler.NewSettler())
	settler, ok := iSettler.(*settler.Settler)
	if !ok {
		return nil
	}

	settler.AddExpense(payer, participants, amount)
	// send the message
	msg := fmt.Sprintf(AddSuccessTemplate, payer, amount, strings.Join(participants, ", "))
	if err := b.SendMessage(update.Message.Chat.ID, msg); err != nil {
		return b.SendMessage(update.Message.Chat.ID, fmt.Sprintf(ErrProcesingRequestTemplate, err))
	}
	return nil
}

// format: /addfor @payer @participant1,@participant2 12.5
func handleAddForExpense(b *bot.Bot, update *bot.Update) error {
	args := update.CommandArgs()
	if len(args) != 3 {
		return b.SendMessage(update.Message.Chat.ID, ErrAddForInvalidArguments)
	}
	// parse the payer
	payer := args[0]
	if payer == "" {
		return b.SendMessage(update.Message.Chat.ID, ErrAddForInvalidArguments)
	}
	// parse the participants
	participants := strings.Split(args[1], ",")
	if len(participants) < 1 {
		return b.SendMessage(update.Message.Chat.ID, ErrAddForInvalidArguments)
	}
	// parse the amount
	amount, err := strconv.ParseFloat(args[2], 64)
	if err != nil {
		return b.SendMessage(update.Message.Chat.ID, ErrAddForInvalidArguments)
	}
	// get the settler of the chat and add the expense
	iSettler := b.GetSession(update, settler.NewSettler())
	settler, ok := iSettler.(*settler.Settler)
	if !ok {
		return nil
	}
	settler.AddExpense(payer, participants, amount)
	// send the message
	msg := fmt.Sprintf(AddSuccessTemplate, payer, amount, strings.Join(participants, ", "))
	if err = b.SendMessage(update.Message.Chat.ID, msg); err != nil {
		return b.SendMessage(update.Message.Chat.ID, fmt.Sprintf(ErrProcesingRequestTemplate, err))
	}
	return nil
}

// format: /list
func handleListExpenses(b *bot.Bot, update *bot.Update) error {
	// get the settler of the chat and list the expenses
	iSettler := b.GetSession(update, settler.NewSettler())
	settler, ok := iSettler.(*settler.Settler)
	if !ok {
		return nil
	}
	expenses, ids := settler.ListExpenses()
	// if there are no expenses, send an error message
	if len(expenses) == 0 {
		return b.SendMessage(update.Message.Chat.ID, ErrNoExpenses)
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
	return b.SendMessage(update.Message.Chat.ID, strings.Join(texts, "\n"))
}

// format: /remove 1
func handleRemoveExpense(b *bot.Bot, update *bot.Update) error {
	args := update.CommandArgs()
	// parse the expense ID
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return b.SendMessage(update.Message.Chat.ID, ErrRemoveInvalidArguments)
	}
	// get the settler of the chat and remove the expense
	iSettler := b.GetSession(update, settler.NewSettler())
	settler, ok := iSettler.(*settler.Settler)
	if !ok {
		return nil
	}
	settler.RemoveExpense(id)
	// send the message
	msg := fmt.Sprintf(RemoveSuccessTemplate, id)
	if err := b.SendMessage(update.Message.Chat.ID, msg); err != nil {
		return b.SendMessage(update.Message.Chat.ID, fmt.Sprintf(ErrProcesingRequestTemplate, err))
	}
	return nil
}

// format: /summary
func handleSummary(b *bot.Bot, update *bot.Update) error {
	// get the settler of the chat, the balances of the participants and the
	// list of transactions to settle the expenses
	iSettler := b.GetSession(update, settler.NewSettler())
	settler, ok := iSettler.(*settler.Settler)
	if !ok {
		return nil
	}
	balances := settler.ListBalances()
	transactions := settler.Settle(false)
	// if there are no transactions, send an error message
	if len(transactions) == 0 {
		return b.SendMessage(update.Message.Chat.ID, ErrNoExpenses)
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
	return b.SendMessage(update.Message.Chat.ID, strings.Join(texts, "\n"))
}

// format: /settle
func handleSettle(b *bot.Bot, update *bot.Update) error {
	// get the settler of the chat, the balances of the participants and the
	// list of transactions to settle the expenses
	iSettler := b.GetSession(update, settler.NewSettler())
	settler, ok := iSettler.(*settler.Settler)
	if !ok {
		return nil
	}
	balances := settler.ListBalances()
	transactions := settler.Settle(true)
	// if there are no transactions, send an error message
	if len(transactions) == 0 {
		return b.SendMessage(update.Message.Chat.ID, ErrNoExpenses)
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
	return b.SendMessage(update.Message.Chat.ID, strings.Join(texts, "\n"))
}

// format: /adduser 123456789 alias
func handleAddUser(b *bot.Bot, update *bot.Update) error {
	args := update.CommandArgs()
	if len(args) != 2 {
		return b.SendMessage(update.Message.Chat.ID, ErrInvalidArguments)
	}
	userID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return b.SendMessage(update.Message.Chat.ID, ErrInvalidArguments)
	}
	userAlias := args[1]
	if err := b.Auth.AddAllowedUser(userID, userAlias); err != nil {
		log.Printf("error adding user: %s", err)
		return b.SendMessage(update.Message.Chat.ID, ErrInternalProcess)
	}
	return b.SendMessage(update.Message.Chat.ID, SuccessInternalMessage)
}

// format: /removeuser 123456789
func handleRemoveUser(b *bot.Bot, update *bot.Update) error {
	args := update.CommandArgs()
	userID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return b.SendMessage(update.Message.Chat.ID, ErrInvalidArguments)
	}
	if found := b.Auth.RemoveAllowedUser(userID); found {
		return b.SendMessage(update.Message.Chat.ID, SuccessInternalMessage)
	}
	log.Println("user not found")
	return b.SendMessage(update.Message.Chat.ID, ErrInternalProcess)
}

// format: /listusers
func handleListUsers(b *bot.Bot, update *bot.Update) error {
	users := b.Auth.ListAllowedUsers()
	if len(users) == 0 {
		return b.SendMessage(update.Message.Chat.ID, ErrInternalProcess)
	}
	texts := []string{UserListHeader}
	for userID, userAlias := range users {
		texts = append(texts, fmt.Sprintf(UserItemTemplate, userAlias, userID))
	}
	return b.SendMessage(update.Message.Chat.ID, strings.Join(texts, "\n"))
}

func handleTest(b *bot.Bot, update *bot.Update) error {
	return b.InlineMenu(update.Message.Chat.ID, 0, "test", map[string]string{
		"Test": "test_data",
	}, func(messageID int64, data string) {
		if err := b.InlineMenu(update.Message.Chat.ID, messageID, "", map[string]string{
			"Test2": "test_data_2",
			"Test3": "test_data_3",
		}, func(i int64, s string) {
			log.Printf("callback 2: %d, %s", i, s)
			b.RemoveInlineMenu(update.Message.Chat.ID, i)
		}); err != nil {
			log.Println(err)
		}
	})
}
