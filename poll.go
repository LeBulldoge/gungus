package main

import (
	"fmt"

	"golang.org/x/exp/slices"
)

type Poll struct {
	Options map[string][]string
}

func (p *Poll) castVote(user string, vote string) error {
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
