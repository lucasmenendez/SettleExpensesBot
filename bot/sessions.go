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
