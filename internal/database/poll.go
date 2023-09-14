package database

import (
	"context"
	"fmt"

	"github.com/LeBulldoge/gungus/internal/poll"
	"github.com/LeBulldoge/sqlighter"
)

func (m *Storage) GetPoll(ID string) (poll.Poll, error) {
	p := poll.Poll{}
	err := m.db.Tx(context.TODO(), func(ctx context.Context, tx *sqlighter.Tx) error {
		err := tx.GetContext(ctx, &p, "SELECT * FROM Polls WHERE id = ?", ID)
		if err != nil {
			return fmt.Errorf("error while getting poll: %w", err)
		}

		p.Options = make(map[string][]string)
		rows, err := tx.QueryContext(ctx, "SELECT name FROM PollOptions WHERE poll_id = ?", p.ID)
		if err != nil {
			return err
		}

		for rows.Next() {
			var opt string
			err = rows.Scan(&opt)
			if err != nil {
				return fmt.Errorf("error while getting options: %w", err)
			}

			p.Options[opt] = []string{}
		}

		rows, err = tx.QueryContext(ctx, "SELECT PollOptions.name, Votes.voter_id FROM Polls JOIN PollOptions, Votes ON PollOptions.poll_id = Polls.id AND PollOptions.id = Votes.option_id WHERE Polls.id = ?", p.ID)
		if err != nil {
			return err
		}

		for rows.Next() {
			var opt string
			var voterID string
			err = rows.Scan(&opt, &voterID)
			if err != nil {
				return fmt.Errorf("error while getting votes: %w", err)
			}

			p.Options[opt] = append(p.Options[opt], voterID)
		}

		return err
	})

	return p, err
}

func (m *Storage) AddPoll(p poll.Poll) error {
	err := m.db.Tx(context.TODO(), func(ctx context.Context, tx *sqlighter.Tx) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO Polls (id, owner, title) VALUES (?, ?, ?)", p.ID, p.Owner, p.Title)
		if err != nil {
			return err
		}

		for o := range p.Options {
			_, err := tx.ExecContext(ctx, "INSERT INTO PollOptions (poll_id, name) VALUES (?, ?)", p.ID, o)
			if err != nil {
				return err
			}
		}

		return err
	})

	return err
}

func (m *Storage) CastVote(pollID string, option string, voterID string) error {
	err := m.db.Tx(context.TODO(), func(ctx context.Context, tx *sqlighter.Tx) error {
		var optionID string
		err := tx.GetContext(ctx, &optionID, "SELECT id FROM PollOptions WHERE poll_id = ? AND name = ?", pollID, option)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, `DELETE FROM Votes
      WHERE option_id IN (SELECT id FROM PollOptions WHERE poll_id = ?)
      AND voter_id = ?`,
			pollID, voterID)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, "INSERT INTO Votes (option_id, voter_id) VALUES (?, ?)", optionID, voterID)

		return err
	})

	return err
}
