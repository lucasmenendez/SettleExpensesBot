package bot

const (
	// commands
	START_CMD           = "start"
	HELP_CMD            = "help"
	ADD_EXPENSE_CMD     = "add"
	ADD_FOR_EXPENSE_CMD = "addfor"
	LIST_EXPENSES_CMD   = "list"
	REMOVE_EXPENSE_CMD  = "remove"
	SUMMARY_CMD         = "summary"
	SETTLE_CMD          = "settle"
	ADD_USER_CMD        = "adduser"
	REMOVE_USER_CMD     = "removeuser"
	LIST_USERS_CMD      = "listusers"
	// descriptions
	HELP_DESC            = "Show this help."
	ADD_EXPENSE_DESC     = "Add an expense for you. Format: /add @participant1,@participant2 12.5"
	ADD_FOR_EXPENSE_DESC = "Add an expense for another user. Format: /addfor @payer @participant1,@participant2 12.5"
	LIST_EXPENSES_DESC   = "List all the expenses with their IDs."
	REMOVE_EXPENSE_DESC  = "Remove an expense by its ID. Format: /remove 1"
	SUMMARY_DESC         = "Show a summary of current debs."
	SETTLE_DESC          = "Show the final summary and suggest transactions to settle debts. This command will remove all the expenses."
	// messages
	WelcomeMessage = "Hello, I'm SettlerBot! Use /help to see the available commands."
	// headers
	HelpHeader             = "Available commands:"
	ListExpensesHeader     = "Current list of expenses:"
	BalancesHeader         = "Current participant balances:"
	SettleHeader           = "Suggestions for debt settlement transactions:"
	UserListHeader         = "Allowed users:"
	SuccessInternalMessage = "Done!"
	// templates
	HelperCommandTemplate = " /%s: %s"
	AddSuccessTemplate    = "Ok, so %s paid %.2f for %s."
	RemoveSuccessTemplate = "Ok, expense %d removed."
	BalanceItemTemplate   = " - %s: %.2f"
	ExpenseItemTemplate   = " %d. %s paid %.2f for %s"
	SettleItemTemplate    = " - %s must pay %.2f to %s"
	UserItemTemplate      = " - %s (%d)"
	// errors
	ErrInvalidArguments         = "invalid arguments"
	ErrInternalProcess          = "internal process error"
	ErrAddInvalidArguments      = "Sorry, I can understand your message. Please use the format: /add @participant1,@participant2 12.5"
	ErrAddForInvalidArguments   = "Sorry, I can understand your message. Please use the format: /addfor @payer @participant1,@participant2 12.5"
	ErrRemoveInvalidArguments   = "Sorry, I can understand your message. Please use the format: /remove 29"
	ErrProcesingRequestTemplate = "Sorry, I can't process your request right now. Please try again later: %s"
	ErrNoExpenses               = "Sorry, there are no expenses yet. Use /add or /addfor to add a new expense."
)
