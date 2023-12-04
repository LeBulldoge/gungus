package playback

import (
	"errors"
	"sync"
)

type PlaybackServiceManager struct {
	sync.RWMutex

	services map[string]*PlaybackService
}

func NewManager() PlaybackServiceManager {
	return PlaybackServiceManager{
		services: make(map[string]*PlaybackService),
	}
}

func (m *PlaybackServiceManager) Get(guildID string) *PlaybackService {
	m.RLock()
	defer m.RUnlock()

	var res *PlaybackService
	if ps, ok := m.services[guildID]; ok {
		res = ps
	}

	return res
}

func (m *PlaybackServiceManager) Add(guildID string, ps *PlaybackService) error {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.services[guildID]; ok {
		return errors.New("playback service already exists")
	}

	m.services[guildID] = ps

	return nil
}

func (m *PlaybackServiceManager) Delete(guildID string) error {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.services[guildID]; !ok {
		return errors.New("playback service does not exist")
	}

	delete(m.services, guildID)

	return nil
}
