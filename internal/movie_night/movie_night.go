package movienight

import (
	"context"
	"fmt"
	"time"

	"github.com/LeBulldoge/gungus/internal/database"
	"github.com/LeBulldoge/sqlighter"
)

type Movie struct {
	ID          string
	Title       string
	Description string
	Image       string
	AddedBy     string    `db:"addedBy"`
	WatchedOn   time.Time `db:"watchedOn"`
}

func (m *Movie) GetURL() string {
	return SOURCE + "/title/" + m.ID
}

func AddMovie(ctx context.Context, storage *database.Storage, ID string, user string, date time.Time) error {
	movie, err := BuildMovieFromID(ID)
	if err != nil {
		return err
	}

	return storage.Tx(ctx, func(ctx context.Context, tx *sqlighter.Tx) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO Movies VALUES(?, ?, ?, ?, ?, ?)", movie.ID, movie.Title, movie.Description, movie.Image, user, date.UTC())
		if err != nil {
			return fmt.Errorf("failure adding a movie: %w", err)
		}

		return nil
	})
}

func GetMovies(ctx context.Context, storage *database.Storage) ([]Movie, error) {
	res := []Movie{}

	return res, storage.Tx(ctx, func(ctx context.Context, tx *sqlighter.Tx) error {
		err := tx.SelectContext(ctx, &res, "SELECT * FROM Movies")
		if err != nil {
			return fmt.Errorf("failure getting a quotes: %w", err)
		}

		return nil
	})
}
