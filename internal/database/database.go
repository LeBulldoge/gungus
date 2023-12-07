package database

import (
	"context"
	"sync"

	"github.com/LeBulldoge/sqlighter"
)

type Storage struct {
	db *sqlighter.DB
	mu sync.Mutex
}

func New(configDir string) *Storage {
	return &Storage{db: sqlighter.New(configDir, targetVersion, versionMap)}
}

func (m *Storage) Open(ctx context.Context) error {
	return m.db.Open(ctx, "foreign_keys=ON")
}

func (m *Storage) Close() error {
	return m.db.Close()
}

func (m *Storage) Tx(ctx context.Context, f func(context.Context, *sqlighter.Tx) error) error {
	return m.db.Tx(ctx, func(ctx context.Context, tx *sqlighter.Tx) error {
		m.mu.Lock()
		defer m.mu.Unlock()
		return f(ctx, tx)
	})
}

type WithStorage struct {
	storage *Storage
}

func (s *WithStorage) SetStorageConnection(storage *Storage) {
	s.storage = storage
}

func (s *WithStorage) Storage() *Storage {
	return s.storage
}
