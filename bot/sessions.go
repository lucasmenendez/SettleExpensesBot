package bot

import (
	"sync"
	"time"

	"github.com/lucasmenendez/expensesbot/settler"
)

type session struct {
	id      int64
	settler *settler.Settler
	expire  time.Time
}

type sessions struct {
	daysToExpire int
	list         map[int64]*session
	mtx          sync.RWMutex
}

func initSessions(daysToExpire int) *sessions {
	return &sessions{
		daysToExpire: daysToExpire,
		list:         make(map[int64]*session),
		mtx:          sync.RWMutex{},
	}
}

func (s *sessions) getOrCreate(id int64) *settler.Settler {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if current, exist := s.list[id]; exist {
		s.list[id].expire = time.Now().AddDate(0, 0, s.daysToExpire)
		return current.settler
	}
	newSession := &session{
		id:      id,
		settler: settler.NewSettler(),
		expire:  time.Now().AddDate(0, 0, s.daysToExpire),
	}
	s.list[id] = newSession
	return newSession.settler
}

func (s *sessions) cleanExpired() {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	toDelete := []int64{}
	for id, session := range s.list {
		if session.expire.Before(time.Now()) {
			toDelete = append(toDelete, id)
		}
	}
	for _, id := range toDelete {
		delete(s.list, id)
	}
}

func (s *sessions) exportSnapshot() []sessionSnapshot {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	snapshot := []sessionSnapshot{}
	for _, session := range s.list {
		transactions := []transactionSnapshot{}
		for _, transaction := range session.settler.Expenses() {
			transactions = append(transactions, transactionSnapshot{
				Amount:       transaction.Amount,
				Payer:        transaction.Payer,
				Participants: transaction.Participants,
			})
		}
		snapshot = append(snapshot, sessionSnapshot{
			ID:           session.id,
			Transactions: transactions,
			Expire:       session.expire.Unix(),
		})
	}
	return snapshot
}

func (s *sessions) importSnapshot(snapshot []sessionSnapshot) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	for _, snapshotData := range snapshot {
		newSession := &session{
			id:      snapshotData.ID,
			settler: settler.NewSettler(),
			expire:  time.Unix(snapshotData.Expire, 0),
		}
		for _, t := range snapshotData.Transactions {
			newSession.settler.AddExpense(t.Payer, t.Participants, t.Amount)
		}
		s.list[snapshotData.ID] = newSession
	}
}
