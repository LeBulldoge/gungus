package database

import (
	"github.com/LeBulldoge/sqlighter/schema"
)

const targetVersion = 7

var versionMap = schema.VersionMap{
	7: schema.Version{
		Up: version7Up,
	},
	6: schema.Version{
		Up: version6Up,
	},
	5: schema.Version{
		Up: version5Up,
	},
	4: schema.Version{
		Up: version4Up,
	},
	3: schema.Version{
		Up: version3Up,
	},
	2: schema.Version{
		Up: version2Up,
	},
	1: schema.Version{
		Up: version1Up,
	},
}

// References Poll (id) -> References Polls(id)
// How did this even work before???
const version7Up = `
PRAGMA foreign_keys=OFF;

CREATE TABLE PollOptionsNew (
  id      INTEGER NOT NULL,
  poll_id TEXT    NOT NULL
                  REFERENCES Polls (id) ON DELETE CASCADE,
  name    TEXT    NOT NULL,
  UNIQUE(poll_id, name),
  PRIMARY KEY (
      id AUTOINCREMENT
  )
);

INSERT INTO PollOptionsNew
SELECT * FROM PollOptions;

DROP TABLE PollOptions;

ALTER TABLE PollOptionsNew RENAME TO PollOptions;

PRAGMA foreign_keys=ON;
`

const version6Up = `CREATE TABLE MovieCast (
    movieId   TEXT NOT NULL
                   REFERENCES Movies(id) ON DELETE CASCADE,
    userId    TEXT NOT NULL,
    character TEXT NOT NULL,
    PRIMARY KEY(movieId, userId),
    UNIQUE(movieId, userId)
);
`

const version5Up = `
PRAGMA foreign_keys=OFF;

CREATE TABLE MovieRatingsNew (
    movieId     TEXT     NOT NULL
                         REFERENCES Movies(id) ON DELETE CASCADE,
    userId      TEXT     NOT NULL,
    rating      NUMBER   NOT NULL,
    PRIMARY KEY(movieId, userId),
    UNIQUE(movieId, userId)
);

INSERT INTO MovieRatingsNew
SELECT * FROM MovieRatings;

DROP TABLE MovieRatings;

ALTER TABLE MovieRatingsNew RENAME TO MovieRatings;

PRAGMA foreign_keys=ON;
`

// Add Movies and MovieRatings tables
const version4Up = `CREATE TABLE Movies (
    id          TEXT     NOT NULL PRIMARY KEY
                UNIQUE,
    title       TEXT     NOT NULL,
    description TEXT     NOT NULL,
    image       TEXT     NOT NULL,
    addedBy     TEXT     NOT NULL,
    watchedOn   DATETIME NOT NULL
);

CREATE TABLE MovieRatings (
    movieId     TEXT     NOT NULL,
    userId      TEXT     NOT NULL,
    rating      NUMBER   NOT NULL,
    PRIMARY KEY(movieId, userId),
    UNIQUE(movieId, userId)
);
`

// Fix PollOptions names
const version3Up = `
  UPDATE PollOptions SET name = "option_" || name
`

// Add quotes table
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
