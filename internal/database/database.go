package database

import (
	"context"

	"github.com/LeBulldoge/sqlighter"
)

type Storage struct {
	db *sqlighter.DB
}

func New() *Storage {
	return &Storage{sqlighter.New(targetVersion, versionMap)}
}

func (m *Storage) Open(ctx context.Context) error {
	return m.db.Open(ctx)
}

func (m *Storage) Close() error {
	return m.db.Close()
}
