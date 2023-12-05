package playback

import (
	"errors"
	"sync"
)

type PlayerStorage struct {
	mu sync.RWMutex

	services map[string]*Player
}

func NewManager() PlayerStorage {
	return PlayerStorage{
		services: make(map[string]*Player),
	}
}

func (m *PlayerStorage) Get(guildID string) *Player {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var res *Player
	if ps, ok := m.services[guildID]; ok {
		res = ps
	}

	return res
}

func (m *PlayerStorage) Add(guildID string, ps *Player) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.services[guildID]; ok {
		return errors.New("playback service already exists")
	}

	m.services[guildID] = ps

	return nil
}

func (m *PlayerStorage) Delete(guildID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.services[guildID]; !ok {
		return errors.New("playback service does not exist")
	}

	delete(m.services, guildID)

	return nil
}
