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

func (m *PlaybackServiceManager) Get(guildId string) *PlaybackService {
	m.RLock()
	defer m.RUnlock()

	var res *PlaybackService
	if ps, ok := m.services[guildId]; ok {
		res = ps
	}

	return res
}

func (m *PlaybackServiceManager) Add(guildId string, ps *PlaybackService) error {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.services[guildId]; ok {
		return errors.New("playback service already exists")
	}

	m.services[guildId] = ps

	return nil
}

func (m *PlaybackServiceManager) Delete(guildId string) error {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.services[guildId]; !ok {
		return errors.New("playback service does not exist")
	} else {
		delete(m.services, guildId)
	}

	return nil
}
