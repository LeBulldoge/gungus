package poll

import (
	"fmt"

	"golang.org/x/exp/slices"
)

type Poll struct {
	ID      string `db:"id"`
	Owner   string `db:"owner"`
	Title   string `db:"title"`
	Options map[string][]string
}

func New(title string) Poll {
	return Poll{
		Title:   title,
		Options: make(map[string][]string),
	}
}

func (p *Poll) CountVotes() map[string]int {
	res := make(map[string]int)

	for opt, votes := range p.Options {
		cnt := len(votes)
		res[opt] = cnt
	}

	return res
}

func (p *Poll) CastVote(user string, vote string) error {
	for opt, votes := range p.Options {
		i := slices.Index(votes, user)
		if i != -1 {
			p.Options[opt] = slices.Delete(votes, i, i+1)
			break
		}
	}

	if votes, ok := p.Options[vote]; ok {
		p.Options[vote] = append(votes, user)
	} else {
		return fmt.Errorf("failure casting vote %s for user %s", vote, user)
	}

	return nil
}
