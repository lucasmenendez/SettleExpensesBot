package main

const (
	// commands
	START_CMD           = "start"
	HELP_CMD            = "help"
	ADD_EXPENSE_CMD     = "add"
	ADD_FOR_EXPENSE_CMD = "addfor"
	LIST_EXPENSES_CMD   = "expenses"
	SUMMARY_CMD         = "summary"
	ADD_USER_CMD        = "adduser"
	REMOVE_USER_CMD     = "removeuser"
	LIST_USERS_CMD      = "listusers"
	// descriptions
	HELP_DESC            = "Shows this help."
	ADD_EXPENSE_DESC     = "Adds an expense for you."
	ADD_FOR_EXPENSE_DESC = "Adds an expense for another user."
	LIST_EXPENSES_DESC   = "Lists all the expenses with their IDs and allows to remove them."
	SUMMARY_DESC         = "Shows a summary of current debs and allows to settle them."
	// messages
	WelcomeMessage              = "👋🏻 Hello, I'm SettlerBot 🤖💶! Use /help to see the available commands."
	RequestPayerPrompt          = "Type the payer username"
	RequestParticipantsPrompt   = "Type the participants usernames"
	RequestAmountMessage        = "How much was the expense? 💶"
	SuccessInternalMessage      = "🎉 Done!"
	ConfirmClearExpensesMessage = "Do you want to clear the list of expenses? 🗑️ 💸"
	ExpensesClearedMessage      = "🎉 Ok, the list of expenses has been cleared."
	RemoveExpenseMessage        = "Do you want to remove any expense? 🗑️ 💸"
	SelectExpenseMessage        = "Select the expense to remove ➡️ 🗑️"
	// headers
	HelpHeader         = "Available commands ❓:"
	ListExpensesHeader = "Current list of expenses 💸:"
	BalancesHeader     = "Current participant balances 💰:"
	SummaryHeader      = "\nSuggestions for debt settlement transactions 🔄:"
	UserListHeader     = "Allowed users:"
	// templates
	RequestPayerTemplate        = "@%s, Who paid the expense? 🤔"
	RequestParticipantsTemplate = "@%s, Who participated in the expense? 🤔"
	HelperCommandTemplate       = " /%s: %s"
	AddSuccessTemplate          = "Ok, so %s paid %.2f for %s. 👍🏻"
	RemoveSuccessTemplate       = "Ok, expense %d removed. 👍🏻"
	BalanceItemTemplate         = " - %s: %.2f"
	ExpenseItemTemplate         = " %d. %s paid %.2f for %s"
	SummaryItemTemplate         = " - %s must pay %.2f to %s"
	UserItemTemplate            = " - %s (%d)"
	// buttons
	ConfirmYesButton = "✅ Yes"
	ConfirmNoButton  = "❌ No"
	CancelButton     = "❌ Cancel"
	// errors
	ErrInvalidArguments         = "❌ Invalid arguments."
	ErrInternalProcess          = "☠️ Internal process error."
	ErrAddInvalidArguments      = "Sorry 😕, I can understand your message. Please use the format: /add @participant1,@participant2 12.5"
	ErrAddForInvalidArguments   = "Sorry 😕, I can understand your message. Please use the format: /addfor @payer @participant1,@participant2 12.5"
	ErrRemoveInvalidArguments   = "Sorry 😕, I can understand your message. Please use the format: /remove 29"
	ErrProcesingRequestTemplate = "Sorry 😕, I can't process your request right now. Please try again later: %s"
	ErrNoExpenses               = "Sorry 😕, there are no expenses yet. Use /add or /addfor to add a new expense."
)
