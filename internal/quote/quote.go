package quote

import (
	"context"
	"fmt"
	"time"

	"github.com/LeBulldoge/gungus/internal/database"
	"github.com/LeBulldoge/sqlighter"
)

type Quote struct {
	User string
	Text string
	Date time.Time
}

func AddQuote(ctx context.Context, storage *database.Storage, user string, text string, date time.Time) error {
	return storage.Tx(ctx, func(ctx context.Context, tx *sqlighter.Tx) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO Quotes VALUES(?, ?, ?)", user, text, date.UTC())
		if err != nil {
			return fmt.Errorf("failure saving a quote: %w", err)
		}

		return nil
	})
}

func GetQuotesByUser(ctx context.Context, storage *database.Storage, user string) ([]Quote, error) {
	res := []Quote{}

	return res, storage.Tx(ctx, func(ctx context.Context, tx *sqlighter.Tx) error {
		err := tx.SelectContext(ctx, &res, "SELECT * FROM Quotes WHERE user = ?", user)
		if err != nil {
			return fmt.Errorf("failure getting a quote for user %s: %w", user, err)
		}

		return nil
	})
}

func GetQuotes(ctx context.Context, storage *database.Storage) ([]Quote, error) {
	res := []Quote{}

	return res, storage.Tx(ctx, func(ctx context.Context, tx *sqlighter.Tx) error {
		err := tx.SelectContext(ctx, &res, "SELECT * FROM Quotes")
		if err != nil {
			return fmt.Errorf("failure getting a quotes: %w", err)
		}

		return nil
	})
}
