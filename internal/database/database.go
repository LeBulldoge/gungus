package database

import (
	"context"

	"github.com/LeBulldoge/sqlighter"
)

type Storage struct {
	db *sqlighter.DB
}

func New(configDir string) *Storage {
	return &Storage{sqlighter.New(configDir, targetVersion, versionMap)}
}

func (m *Storage) Open(ctx context.Context) error {
	return m.db.Open(ctx)
}

func (m *Storage) Close() error {
	return m.db.Close()
}

func (m *Storage) Tx(ctx context.Context, f func(context.Context, *sqlighter.Tx) error) error {
	return m.db.Tx(ctx, f)
}
