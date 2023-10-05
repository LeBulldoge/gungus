package database

import (
	"github.com/LeBulldoge/sqlighter/schema"
)

const targetVersion = 2

var versionMap = schema.VersionMap{
	2: schema.Version{
		Up: version2Up,
	},
	1: schema.Version{
		Up: version1Up,
	},
}

const version2Up = `CREATE TABLE Quotes (
  user TEXT     NOT NULL,
  text TEXT     NOT NULL,
  date DATETIME NOT NULL
);`

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
