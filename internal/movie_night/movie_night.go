package movienight

import (
	"context"
	"database/sql"
	"errors"
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

	Ratings []MovieRating
	Cast    []CastMember
}

func (m *Movie) GetURL() string {
	return searchSource + "/title/" + m.ID
}

func doesMovieExist(tx *sqlighter.Tx, ID string) (bool, error) {
	row := tx.QueryRowx("SELECT ID FROM Movies WHERE ID = ?", ID)

	var id string
	err := row.Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	return true, err
}

func AddMovie(ctx context.Context, storage *database.Storage, ID string, user string, date time.Time) error {
	return storage.Tx(ctx, func(ctx context.Context, tx *sqlighter.Tx) error {
		movieExists, err := doesMovieExist(tx, ID)
		if err != nil {
			return err
		}
		if movieExists {
			return fmt.Errorf("movie already exists")
		}

		movie, err := BuildMovieFromID(ID)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, "INSERT INTO Movies VALUES(?, ?, ?, ?, ?, ?)", movie.ID, movie.Title, movie.Description, movie.Image, user, date.UTC())
		if err != nil {
			return fmt.Errorf("failure adding a movie: %w", err)
		}

		return nil
	})
}

func GetMovie(ctx context.Context, storage *database.Storage, ID string) (Movie, error) {
	res := Movie{}

	return res, storage.Tx(ctx, func(ctx context.Context, tx *sqlighter.Tx) error {
		err := tx.GetContext(ctx, &res, "SELECT * FROM Movies WHERE ID = ?", ID)
		if err != nil {
			return fmt.Errorf("failure getting movies: %w", err)
		}

		err = tx.SelectContext(ctx, &res.Ratings, "SELECT * FROM MovieRatings WHERE movieId = ?", res.ID)
		if err != nil {
			return fmt.Errorf("failure getting movie ratings: %w", err)
		}

		err = tx.SelectContext(ctx, &res.Cast, "SELECT * FROM MovieCast WHERE movieId = ?", res.ID)
		if err != nil {
			return fmt.Errorf("failure getting movie cast: %w", err)
		}

		return nil
	})
}

func GetMovies(ctx context.Context, storage *database.Storage) ([]Movie, error) {
	res := []Movie{}

	return res, storage.Tx(ctx, func(ctx context.Context, tx *sqlighter.Tx) error {
		err := tx.SelectContext(ctx, &res, "SELECT * FROM Movies ORDER BY watchedOn DESC")
		if err != nil {
			return fmt.Errorf("failure getting movies: %w", err)
		}

		for i, movie := range res {
			err = tx.SelectContext(ctx, &res[i].Ratings, "SELECT * FROM MovieRatings WHERE movieId = ?", movie.ID)
			if err != nil {
				return fmt.Errorf("failure getting movie ratings: %w", err)
			}

			err = tx.SelectContext(ctx, &res[i].Cast, "SELECT * FROM MovieCast WHERE movieId = ?", movie.ID)
			if err != nil {
				return fmt.Errorf("failure getting movie cast: %w", err)
			}
		}

		return nil
	})
}

func GetMoviesByTitle(ctx context.Context, storage *database.Storage, title string) ([]Movie, error) {
	res := []Movie{}

	return res, storage.Tx(ctx, func(ctx context.Context, tx *sqlighter.Tx) error {
		err := tx.SelectContext(ctx, &res, "SELECT * FROM Movies WHERE title LIKE '%"+title+"%'")
		if err != nil {
			return fmt.Errorf("failure getting movies: %w", err)
		}

		for i, movie := range res {
			err = tx.SelectContext(ctx, &res[i].Ratings, "SELECT * FROM MovieRatings WHERE movieId = ?", movie.ID)
			if err != nil {
				return fmt.Errorf("failure getting movie ratings: %w", err)
			}
		}

		return nil
	})
}

func RateMovie(ctx context.Context, storage *database.Storage, ID string, user string, rating float64) error {
	return storage.Tx(ctx, func(ctx context.Context, tx *sqlighter.Tx) error {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO MovieRatings VALUES(?, ?, ?)
      ON CONFLICT(movieId, userId) DO UPDATE SET rating=excluded.rating`,
			ID, user, rating,
		)

		if err != nil {
			return fmt.Errorf("failure adding a rating: %w", err)
		}

		return nil
	})
}

func DeleteMovie(ctx context.Context, storage *database.Storage, ID string) error {
	return storage.Tx(ctx, func(ctx context.Context, tx *sqlighter.Tx) error {
		_, err := tx.ExecContext(ctx,
			`DELETE FROM Movies WHERE id = ?`,
			ID,
		)
		return err
	})
}

type MovieRating struct {
	MovieID string `db:"movieId"`
	UserID  string `db:"userId"`
	Rating  float64
}

func GetRatings(ctx context.Context, storage *database.Storage, movieID string) ([]MovieRating, error) {
	res := []MovieRating{}

	return res, storage.Tx(ctx, func(ctx context.Context, tx *sqlighter.Tx) error {
		err := tx.SelectContext(ctx, &res, "SELECT * FROM MovieRatings WHERE movieId = ?", movieID)
		if err != nil {
			return fmt.Errorf("failure getting movie ratings: %w", err)
		}

		return nil
	})
}

type CastMember struct {
	MovieID   string `db:"movieId"`
	UserID    string `db:"userId"`
	Character string
}

func GetCast(ctx context.Context, storage *database.Storage, movieID string) ([]CastMember, error) {
	res := []CastMember{}

	return res, storage.Tx(ctx, func(ctx context.Context, tx *sqlighter.Tx) error {
		err := tx.SelectContext(ctx, &res, "SELECT * FROM MovieCast WHERE movieId = ?", movieID)
		if err != nil {
			return fmt.Errorf("failure getting movie cast: %w", err)
		}

		return nil
	})
}

func AddUserAsCastMember(ctx context.Context, storage *database.Storage, movieID string, userId string, character string) error {
	return storage.Tx(ctx, func(ctx context.Context, tx *sqlighter.Tx) error {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO MovieCast VALUES(?, ?, ?)
      ON CONFLICT(movieId, userId) DO UPDATE SET character=excluded.character`,
			movieID, userId, character)

		if err != nil {
			return fmt.Errorf("failure adding a character: %w", err)
		}

		return nil
	})
}
