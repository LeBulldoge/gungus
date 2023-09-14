package database

import (
	"github.com/LeBulldoge/sqlighter/schema"
)

const targetVersion = 1

var versionMap = schema.VersionMap{
	1: schema.Version{
		Up: version1Up,
	},
}

// The initial schema
const version1Up = `CREATE TABLE Polls (
  id     TEXT   NOT NULL PRIMARY KEY
                UNIQUE,
  owner  TEXT   NOT NULL,
  title  TEXT   NOT NULL
);

CREATE TABLE PollOptions (
  id      INTEGER NOT NULL,
  poll_id TEXT    NOT NULL
                  REFERENCES Polls (id) ON DELETE CASCADE,
  name    TEXT    NOT NULL,
  UNIQUE(poll_id, name)
  FOREIGN KEY (
      poll_id
  )
  REFERENCES Poll (id),
  PRIMARY KEY (
      id AUTOINCREMENT
  )
);

CREATE TABLE Votes (
  option_id INTEGER NOT NULL
                    REFERENCES PollOptions (id) ON DELETE CASCADE,
  voter_id  TEXT    NOT NULL
);`
