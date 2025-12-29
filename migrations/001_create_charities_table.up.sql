-- SQLite schema for charities table
CREATE TABLE IF NOT EXISTS charities (
    registered_number INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    status TEXT,
    date_registered DATETIME,
    date_removed DATETIME,
    address TEXT,
    website TEXT,
    email TEXT,
    phone TEXT,
    what_the_charity_does TEXT,
    who_the_charity_helps TEXT,
    how_the_charity_works TEXT,
    last_updated DATETIME DEFAULT CURRENT_TIMESTAMP
);