package settler

import (
	"testing"
)

func TestSettler(t *testing.T) {
	// initialize settler with transactions
	transactions := []*Transaction{
		{Payer: "Alice", Participants: []string{"Bob", "Carol"}, Amount: 30.0},
		{Payer: "Carol", Participants: []string{"Bob"}, Amount: 20.0},
		{Payer: "Bob", Participants: []string{"Alice"}, Amount: 10.0},
	}
	settler := NewSettler()
	for _, transaction := range transactions {
		settler.AddExpense(transaction.Payer, transaction.Participants, transaction.Amount)
	}
	// settle transactions and check results with expected results
	settledTransactions := settler.Settle()
	expectedTransactions := []*Transaction{
		{Payer: "Bob", Participants: []string{"Alice"}, Amount: 20.0},
		{Payer: "Bob", Participants: []string{"Carol"}, Amount: 5.0},
	}
	// check if the number of transactions is the same
	if len(settledTransactions) != len(expectedTransactions) {
		t.Errorf("Expected 2 transactions, got %d", len(settledTransactions))
	}
	// check if the transactions are the same by checking the number of founded
	// expected transactions
	founded := 0
	for _, transaction := range settledTransactions {
		for _, expectedTransaction := range expectedTransactions {
			if transaction.Payer == expectedTransaction.Payer &&
				transaction.Participants[0] == expectedTransaction.Participants[0] &&
				transaction.Amount == expectedTransaction.Amount {
				founded++
			}
		}
	}
	if founded != len(expectedTransactions) {
		t.Errorf("Expected %d transactions, got %d", len(expectedTransactions), founded)
	}
	// add a new expense and settle transactions again
	lastAdded := settler.AddExpense("Bob", []string{"Carol", "Alice"}, 10.0)
	settledTransactions = settler.Settle()
	// check if the number of transactions is the same as expected and check
	// again if the transactions are the same
	expectedTransactions2 := []*Transaction{
		{Payer: "Bob", Participants: []string{"Alice"}, Amount: 15.0},
	}
	if len(settledTransactions) != len(expectedTransactions2) {
		t.Errorf("Expected 2 transactions, got %d", len(settledTransactions))
	}
	founded = 0
	for _, transaction := range settledTransactions {
		for _, expectedTransaction := range expectedTransactions2 {
			if transaction.Payer == expectedTransaction.Payer &&
				transaction.Participants[0] == expectedTransaction.Participants[0] &&
				transaction.Amount == expectedTransaction.Amount {
				founded++
			}
		}
	}
	if founded != len(expectedTransactions2) {
		t.Errorf("Expected %d transactions, got %d", len(expectedTransactions2), founded)
	}
	// remove the last added expense and settle transactions again
	settler.RemoveExpense(lastAdded)
	settledTransactions = settler.Settle()
	// check if the number of transactions is the same as expected and check
	// again if the transactions are the same of the first settlement
	if len(settledTransactions) != len(expectedTransactions) {
		t.Errorf("Expected %d transactions, got %d", len(expectedTransactions), len(settledTransactions))
	}
	founded = 0
	for _, transaction := range settledTransactions {
		for _, expectedTransaction := range expectedTransactions {
			if transaction.Payer == expectedTransaction.Payer &&
				transaction.Participants[0] == expectedTransaction.Participants[0] &&
				transaction.Amount == expectedTransaction.Amount {
				founded++
			}
		}
	}
	if founded != len(expectedTransactions) {
		t.Errorf("Expected %d transactions, got %d", len(expectedTransactions), founded)
	}
}
