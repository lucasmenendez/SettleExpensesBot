package bot

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type Data interface {
	Export() ([]byte, error)
}

type DataImporter func(encoded []byte) (Data, error)

type session struct {
	id     int64
	data   Data
	expire time.Time
}

type sessionDump map[int64]string

type sessions struct {
	daysToExpire int
	list         map[int64]*session
	mtx          sync.RWMutex
	importer     DataImporter
}

func initSessions(daysToExpire int) *sessions {
	return &sessions{
		daysToExpire: daysToExpire,
		list:         make(map[int64]*session),
		mtx:          sync.RWMutex{},
	}
}

func (s *sessions) getOrCreate(id int64, initial Data) any {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if current, exist := s.list[id]; exist {
		s.list[id].expire = time.Now().AddDate(0, 0, s.daysToExpire)
		return current.data
	}
	newSession := &session{
		id:     id,
		data:   initial,
		expire: time.Now().AddDate(0, 0, s.daysToExpire),
	}
	s.list[id] = newSession
	return newSession.data
}

func (s *sessions) cleanExpired() []int64 {
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
	return toDelete
}

func (s *sessions) importSnapshot(snapshot []byte) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if s.importer == nil {
		return fmt.Errorf("no importer set")
	}
	sessionsData := sessionDump{}
	if err := json.Unmarshal(snapshot, &sessionsData); err != nil {
		return err
	}
	for id, encData := range sessionsData {
		bData, err := hex.DecodeString(encData)
		if err != nil {
			return err
		}
		data, err := s.importer(bData)
		if err != nil {
			return err
		}
		s.list[id] = &session{
			id:     id,
			data:   data,
			expire: time.Now().AddDate(0, 0, s.daysToExpire),
		}
	}

	return nil
}

func (s *sessions) exportSnapshot() ([]byte, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	sessionsData := sessionDump{}
	for id, session := range s.list {
		encData, err := session.data.Export()
		if err != nil {
			return nil, err
		}
		sessionsData[id] = hex.EncodeToString(encData)
	}

	snapshot, err := json.Marshal(sessionsData)
	if err != nil {
		return nil, err
	}

	return snapshot, nil
}
