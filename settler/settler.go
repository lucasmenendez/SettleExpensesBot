package settler

import "math"

// Transaction struct represents an expense transaction.
type Transaction struct {
	Payer        string
	Participants []string
	Amount       float64
}

// Settler struct contains the list of expenses. They can be settled and
// cleaned, or just settled.
type Settler struct {
	expenses map[int]*Transaction
	lastID   int
}

// NewSettler creates a new Settler instance.
func NewSettler() *Settler {
	return &Settler{
		expenses: make(map[int]*Transaction),
		lastID:   0,
	}
}

// AddExpense method adds an expense to the list of expenses.
func (s *Settler) AddExpense(payer string, participants []string, amount float64) int {
	s.lastID++
	s.expenses[s.lastID] = &Transaction{payer, participants, amount}
	return s.lastID
}

// RemoveExpense method removes an expense from the list of expenses.
func (s *Settler) RemoveExpense(id int) {
	delete(s.expenses, id)
}

// Expenses method returns the map of expenses with their IDs.
func (s *Settler) Expenses() map[int]*Transaction {
	return s.expenses
}

// Settle method returns the list of transactions resulting from the settlement.
// It does not clean the list of expenses. The settlement algorithm consists in
// minimizing the number of transactions needed to settle shared expenses. It
// calculates the balance for each person and then settles the debts by finding
// the person who has paid the most and the person who has paid the least and
// the amounts. It then settles the debt getting the minimum between the amount
// owed and the amount owed. It repeats this process until all debts are
// settled.
func (s *Settler) Settle() []*Transaction {
	balances := make(map[string]float64)
	// calculate the balance for each person
	for _, transaction := range s.expenses {
		for _, participant := range transaction.Participants {
			ammount := transaction.Amount / float64(len(transaction.Participants))
			balances[transaction.Payer] += ammount
			balances[participant] -= ammount
		}
	}
	// initialize the transactions list
	result := []*Transaction{}
	for {
		maxCreditor := ""
		maxDebtor := ""
		maxAmount := 0.0
		minAmount := 0.0
		// find the person who has paid the most and the person who has paid
		// the least and the amounts
		for person, balance := range balances {
			if balance > maxAmount {
				maxCreditor = person
				maxAmount = balance
			}
			if balance < minAmount {
				maxDebtor = person
				minAmount = balance
			}
		}
		// if no one owes the most or is owed the most, debts are settled
		if maxCreditor == "" && maxDebtor == "" {
			break
		}
		// settle the debt getting the minimum between the amount owed and the
		// amount owed
		settleAmount := math.Min(maxAmount, math.Abs(minAmount))
		// update the balances
		balances[maxCreditor] -= settleAmount
		balances[maxDebtor] += settleAmount
		// add this transaction to the result
		result = append(result, &Transaction{
			Payer:        maxDebtor,
			Participants: []string{maxCreditor},
			Amount:       settleAmount,
		})
	}
	return result
}

// SettleAndClean method cleans the list of expenses and return the list of
// transactions resulting from the settlement.
func (s *Settler) SettleAndClean() []*Transaction {
	res := s.Settle()
	s.expenses = make(map[int]*Transaction)
	return res
}
