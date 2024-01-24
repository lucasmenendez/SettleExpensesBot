package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/lucasmenendez/expensesbot/bot"
	"github.com/lucasmenendez/expensesbot/settler"
)

var publicCommands = []string{
	HELP_CMD,
	ADD_EXPENSE_CMD,
	ADD_FOR_EXPENSE_CMD,
	LIST_EXPENSES_CMD,
	SUMMARY_CMD,
	IMPORT_CMD,
	EXPORT_CMD,
}

var commandsDescriptions = map[string]string{
	HELP_CMD:            HELP_DESC,
	ADD_EXPENSE_CMD:     ADD_EXPENSE_DESC,
	ADD_FOR_EXPENSE_CMD: ADD_FOR_EXPENSE_DESC,
	LIST_EXPENSES_CMD:   LIST_EXPENSES_DESC,
	SUMMARY_CMD:         SUMMARY_DESC,
	IMPORT_CMD:          IMPORT_DESC,
	EXPORT_CMD:          EXPORT_DESC,
}

// format: /start
func handleStart(b *bot.Bot, update *bot.Update) error {
	_, err := b.SendMessage(update.Message.Chat.ID, 0, WelcomeMessage)
	return err
}

// format: /help
func handleHelp(b *bot.Bot, update *bot.Update) error {
	texts := []string{HelpHeader}
	for _, cmd := range publicCommands {
		texts = append(texts, fmt.Sprintf(HelperCommandTemplate, cmd, commandsDescriptions[cmd]))
	}
	_, err := b.SendMessage(update.Message.Chat.ID, 0, strings.Join(texts, "\n"))
	return err
}

// format: /add
func handleAddExpense(b *bot.Bot, update *bot.Update) error {
	from := update.Message.From.Username
	payer := fmt.Sprintf("@%s", update.Message.From.Username)
	// answer for the participants
	return b.SendMessageToReply(update.Message.Chat.ID,
		fmt.Sprintf(RequestParticipantsTemplate, from), RequestParticipantsPrompt,
		func(messageID int64, update *bot.Update) {
			// validate the participants
			participants := strings.Split(update.Message.Text, " ")
			if len(participants) == 0 {
				if _, err := b.SendMessage(update.Message.Chat.ID, 0, ErrAddInvalidArguments); err != nil {
					log.Printf("error sending message: %s\n", err)
				}
				return
			}
			// answer for the amount
			if err := requestAmount(b, update.Message.Chat.ID, RequestAmountMessage, func(amount float64) {
				// get the settler of the chat and add the expense
				iSettler := b.GetSession(update, settler.NewSettler())
				settler, ok := iSettler.(*settler.Settler)
				if !ok {
					log.Println("error getting settler")
				}
				settler.AddExpense(payer, participants, amount)
				// send the message
				msg := fmt.Sprintf(AddSuccessTemplate, payer, amount, strings.Join(participants, ", "))
				if _, err := b.SendMessage(update.Message.Chat.ID, 0, msg); err != nil {
					log.Printf("error sending message: %s\n", err)
				}
			}); err != nil {
				log.Println(err)
			}
		},
	)
}

// format: /addfor
func handleAddForExpense(b *bot.Bot, update *bot.Update) error {
	from := update.Message.From.Username
	// answer for the payer
	return b.SendMessageToReply(update.Message.Chat.ID,
		fmt.Sprintf(RequestPayerTemplate, from), RequestPayerPrompt,
		func(messageID int64, update *bot.Update) {
			payer := update.Message.Text
			// answer for the participants
			if err := b.SendMessageToReply(update.Message.Chat.ID,
				fmt.Sprintf(RequestParticipantsTemplate, from), RequestParticipantsPrompt,
				func(messageID int64, update *bot.Update) {
					// validate the participants
					participants := strings.Split(update.Message.Text, " ")
					if len(participants) == 0 {
						if _, err := b.SendMessage(update.Message.Chat.ID, 0, ErrAddInvalidArguments); err != nil {
							log.Printf("error sending message: %s\n", err)
						}
						return
					}
					// answer for the amount
					if err := requestAmount(b, update.Message.Chat.ID, RequestAmountMessage, func(amount float64) {
						// get the settler of the chat and add the expense
						iSettler := b.GetSession(update, settler.NewSettler())
						settler, ok := iSettler.(*settler.Settler)
						if !ok {
							log.Println("error getting settler")
						}
						settler.AddExpense(payer, participants, amount)
						// send the message
						msg := fmt.Sprintf(AddSuccessTemplate, payer, amount, strings.Join(participants, ", "))
						if _, err := b.SendMessage(update.Message.Chat.ID, 0, msg); err != nil {
							log.Printf("error sending message: %s\n", err)
						}
					}); err != nil {
						log.Println(err)
					}
				},
			); err != nil {
				log.Println(err)
			}
		},
	)
}

// format: /expenses
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
		_, err := b.SendMessage(update.Message.Chat.ID, 0, ErrNoExpenses)
		return err
	}
	buttonsPerRow := 5
	labels := make([][]string, len(expenses)/buttonsPerRow+1)
	// compose and send the message
	texts := []string{ListExpensesHeader}
	currentRow := 0
	for i, expense := range expenses {
		texts = append(texts, fmt.Sprintf(ExpenseItemTemplate,
			ids[i],
			expense.Payer,
			expense.Amount,
			strings.Join(expense.Participants, ", "),
		))
		if len(labels[currentRow]) == buttonsPerRow {
			currentRow++
		}
		labels[currentRow] = append(labels[currentRow], strconv.Itoa(ids[i]))
	}
	values := append([][]string{}, labels...)
	labels = append(labels, []string{CancelButton})
	values = append(values, []string{"cancel"})

	if _, err := b.SendMessage(update.Message.Chat.ID, 0, strings.Join(texts, "\n")); err != nil {
		_, err := b.SendMessage(update.Message.Chat.ID, 0, fmt.Sprintf(ErrProcesingRequestTemplate, err))
		return err
	}
	return confirm(b, update.Message.Chat.ID, RemoveExpenseMessage, func(remove bool) {
		if remove {
			if _, err := b.InlineMenu(update.Message.Chat.ID, 0,
				SelectExpenseMessage, labels, values,
				func(messageID int64, data string) {
					if data == "cancel" {
						if err := b.RemoveMessage(update.Message.Chat.ID, messageID); err != nil {
							log.Println(err)
						}
						return
					}
					id, err := strconv.Atoi(data)
					if err != nil {
						log.Println(err)
						return
					}
					settler.RemoveExpense(id)
					if _, err := b.SendMessage(update.Message.Chat.ID, messageID, fmt.Sprintf(RemoveSuccessTemplate, id)); err != nil {
						log.Println(err)
					}
				},
			); err != nil {
				log.Println(err)
			}
			return
		}
	})
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
		_, err := b.SendMessage(update.Message.Chat.ID, 0, ErrNoExpenses)
		return err
	}
	// compose and send the message
	texts := []string{BalancesHeader}
	for participant, balance := range balances {
		texts = append(texts, fmt.Sprintf(BalanceItemTemplate, participant, balance))
	}
	texts = append(texts, SummaryHeader)
	for _, transaction := range transactions {
		texts = append(texts, fmt.Sprintf(SummaryItemTemplate,
			transaction.Payer,
			transaction.Amount,
			transaction.Participants[0],
		))
	}
	if _, err := b.SendMessage(update.Message.Chat.ID, 0, strings.Join(texts, "\n")); err != nil {
		_, err := b.SendMessage(update.Message.Chat.ID, 0, fmt.Sprintf(ErrProcesingRequestTemplate, err))
		return err
	}

	return confirm(b, update.Message.Chat.ID, ConfirmClearExpensesMessage, func(clear bool) {
		if clear {
			settler.Clean()
			if _, err := b.SendMessage(update.Message.Chat.ID, 0, ExpensesClearedMessage); err != nil {
				log.Println(err)
			}
			return
		}
	})
}

// format: /import
func handleImport(b *bot.Bot, update *bot.Update) error {
	from := update.Message.From.Username
	text := fmt.Sprintf(ImportFileTemplate, from)
	return b.SendMessageToReply(update.Message.Chat.ID, text, ImportFilePrompt,
		func(messageID int64, update *bot.Update) {
			if update.Message.Document == nil {
				if _, err := b.SendMessage(update.Message.Chat.ID, 0, ErrInvalidImportFile); err != nil {
					log.Printf("error sending message: %s\n", err)
				}
				return
			}
			// download the file
			fileContent, err := b.DownloadFile(update.Message.Document.ID)
			if err != nil {
				if _, err := b.SendMessage(update.Message.Chat.ID, 0, ErrInvalidImportFile); err != nil {
					log.Printf("error sending message: %s\n", err)
				}
				return
			}
			// parse the file
			buffer := bytes.NewBuffer(fileContent)
			csvReader := csv.NewReader(buffer)
			records, err := csvReader.ReadAll()
			if err != nil {
				log.Println(err)
				if _, err := b.SendMessage(update.Message.Chat.ID, 0, ErrInvalidImportFile); err != nil {
					log.Printf("error sending message: %s\n", err)
				}
				return
			}
			// validate the records
			expenses := []*settler.Transaction{}
			for _, record := range records {
				if len(record) != 3 {
					if _, err := b.SendMessage(update.Message.Chat.ID, 0, ErrInvalidImportFile); err != nil {
						log.Printf("error sending message: %s\n", err)
					}
					return
				}
				payer, rawParticipants, rawAmount := record[0], record[1], record[2]
				participants := strings.Split(rawParticipants, ";")
				amount, err := strconv.ParseFloat(rawAmount, 64)
				if err != nil {
					log.Println(err)
					if _, err := b.SendMessage(update.Message.Chat.ID, 0, ErrInvalidImportFile); err != nil {
						log.Printf("error sending message: %s\n", err)
					}
					return
				}
				expenses = append(expenses, &settler.Transaction{
					Payer:        payer,
					Participants: participants,
					Amount:       amount,
				})
			}
			// get the settler of the chat and add the expense
			iSettler := b.GetSession(update, settler.NewSettler())
			settler, ok := iSettler.(*settler.Settler)
			if !ok {
				log.Println("error getting settler")
			}
			// if there are no expenses, add them without confirmation
			if _, ids := settler.ListExpenses(); len(ids) == 0 {
				for _, expense := range expenses {
					settler.AddExpense(expense.Payer, expense.Participants, expense.Amount)
				}
				// send the message
				msg := fmt.Sprintf(ImportDoneTemplate, len(expenses))
				if _, err := b.SendMessage(update.Message.Chat.ID, 0, msg); err != nil {
					log.Printf("error sending message: %s\n", err)
				}
				return
			}
			// if there are expenses, ask for confirmation and add them if confirmed
			if err := confirm(b, update.Message.Chat.ID, ImportAlertMessage, func(continueImport bool) {
				if continueImport {
					settler.Clean()
					for _, expense := range expenses {
						settler.AddExpense(expense.Payer, expense.Participants, expense.Amount)
					}
					// send the message
					msg := fmt.Sprintf(ImportDoneTemplate, len(expenses))
					if _, err := b.SendMessage(update.Message.Chat.ID, 0, msg); err != nil {
						log.Printf("error sending message: %s\n", err)
					}
					return
				}
			}); err != nil {
				log.Println(err)
			}
		})
}

// format: /export
func handleExport(b *bot.Bot, update *bot.Update) error {
	// get the settler of the chat, the balances of the participants and the
	// list of transactions to settle the expenses
	iSettler := b.GetSession(update, settler.NewSettler())
	settler, ok := iSettler.(*settler.Settler)
	if !ok {
		return nil
	}
	expenses, _ := settler.ListExpenses()

	strBuffer := strings.Builder{}
	csvWriter := csv.NewWriter(&strBuffer)
	for _, expense := range expenses {
		if err := csvWriter.Write([]string{
			expense.Payer,
			strings.Join(expense.Participants, ";"),
			fmt.Sprintf("%.2f", expense.Amount),
		}); err != nil {
			log.Println(err)
			if _, err := b.SendMessage(update.Message.Chat.ID, 0, ErrInternalProcess); err != nil {
				return err
			}
		}
	}
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		log.Println(err)
		if _, err := b.SendMessage(update.Message.Chat.ID, 0, ErrInternalProcess); err != nil {
			return err
		}
	}
	if _, err := b.SendMessage(update.Message.Chat.ID, 0, ExportFileMessage); err != nil {
		if _, err := b.SendMessage(update.Message.Chat.ID, 0, ErrInternalProcess); err != nil {
			return err
		}
	}
	return b.SendDocument(update.Message.Chat.ID, "expenses.csv", strBuffer.String())
}

// format: /adduser 123456789 alias
func handleAddUser(b *bot.Bot, update *bot.Update) error {
	args := update.CommandArgs()
	if len(args) != 2 {
		_, err := b.SendMessage(update.Message.Chat.ID, 0, ErrInvalidArguments)
		return err
	}
	userID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		_, err := b.SendMessage(update.Message.Chat.ID, 0, ErrInvalidArguments)
		return err
	}
	userAlias := args[1]
	if err := b.Auth.AddAllowedUser(userID, userAlias); err != nil {
		log.Printf("error adding user: %s", err)
		_, err := b.SendMessage(update.Message.Chat.ID, 0, ErrInternalProcess)
		return err
	}
	_, err = b.SendMessage(update.Message.Chat.ID, 0, SuccessInternalMessage)
	return err
}

// format: /removeuser 123456789
func handleRemoveUser(b *bot.Bot, update *bot.Update) error {
	args := update.CommandArgs()
	userID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		_, err := b.SendMessage(update.Message.Chat.ID, 0, ErrInvalidArguments)
		return err
	}
	if found := b.Auth.RemoveAllowedUser(userID); found {
		_, err := b.SendMessage(update.Message.Chat.ID, 0, SuccessInternalMessage)
		return err
	}
	log.Println("user not found")
	_, err = b.SendMessage(update.Message.Chat.ID, 0, ErrInternalProcess)
	return err
}

// format: /listusers
func handleListUsers(b *bot.Bot, update *bot.Update) error {
	users := b.Auth.ListAllowedUsers()
	if len(users) == 0 {
		_, err := b.SendMessage(update.Message.Chat.ID, 0, ErrInternalProcess)
		return err
	}
	texts := []string{UserListHeader}
	for userID, userAlias := range users {
		texts = append(texts, fmt.Sprintf(UserItemTemplate, userAlias, userID))
	}
	_, err := b.SendMessage(update.Message.Chat.ID, 0, strings.Join(texts, "\n"))
	return err
}
