package settler

import (
	"math"
	"sort"
)

// Transaction struct represents an expense transaction.
type Transaction struct {
	Payer        string
	Participants []string
	Amount       float64
}

// Settler struct contains the list of expenses. They can be settled and
// cleaned, or just settled.
type Settler struct {
	balances map[string]float64
	expenses map[int]*Transaction
	lastID   int
}

// NewSettler creates a new Settler instance.
func NewSettler() *Settler {
	return &Settler{
		balances: make(map[string]float64),
		expenses: make(map[int]*Transaction),
		lastID:   0,
	}
}

// AddExpense method adds an expense to the list of expenses.
func (s *Settler) AddExpense(payer string, participants []string, amount float64) int {
	s.lastID++
	s.expenses[s.lastID] = &Transaction{payer, participants, amount}
	s.balances[payer] += amount
	amountByParticipant := amount / float64(len(participants))
	for _, participant := range participants {
		s.balances[participant] -= amountByParticipant
	}
	return s.lastID
}

// RemoveExpense method removes an expense from the list of expenses.
func (s *Settler) RemoveExpense(id int) {
	if expense, exist := s.expenses[id]; exist {
		s.balances[expense.Payer] -= expense.Amount
		amountByParticipant := expense.Amount / float64(len(expense.Participants))
		for _, participant := range expense.Participants {
			s.balances[participant] += amountByParticipant
		}
	}
	delete(s.expenses, id)
}

// Expenses method returns the map of expenses with their IDs.
func (s *Settler) Expenses() ([]*Transaction, []int) {
	ids := sort.IntSlice{}
	for id := range s.expenses {
		ids = append(ids, id)
	}
	// sort the IDs
	sort.Sort(ids)
	// create the list of expenses
	expenses := []*Transaction{}
	for _, id := range ids {
		expenses = append(expenses, s.expenses[id])
	}
	return expenses, ids
}

// Balances method returns the map of balances for each person. If the balance
// is positive, the person is owed money, if it is negative, the person owes
// money.
func (s *Settler) Balances() map[string]float64 {
	// create a copy of the balances map
	balances := make(map[string]float64)
	for person, balance := range s.balances {
		balances[person] = balance
	}
	return balances
}

// Settle method returns the list of transactions resulting from the settlement.
// It does not clean the list of expenses. The settlement algorithm consists in
// minimizing the number of transactions needed to settle shared expenses. It
// calculates the balance for each person and then settles the debts by finding
// the person who has paid the most and the person who has paid the least and
// the amounts. It then settles the debt getting the minimum between the amount
// owed and the amount owed. It repeats this process until all debts are
// settled.
func (s *Settler) Settle(clean bool) []*Transaction {
	// clean the list of expenses and balances if requested
	if clean {
		defer s.Clean()
	}
	// get a copy of current balances of the participants
	balances := s.Balances()
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
		if maxAmount <= 0.001 && minAmount <= 0.001 {
			break
		}
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

// Clean method cleans the list of expenses and balances of the settler.
func (b *Settler) Clean() {
	b.expenses = make(map[int]*Transaction)
	b.balances = make(map[string]float64)
	b.lastID = 0
}
