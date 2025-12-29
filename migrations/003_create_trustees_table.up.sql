-- SQLite schema for trustees table
CREATE TABLE IF NOT EXISTS trustees (
    charity_number INTEGER NOT NULL,
    name TEXT NOT NULL,
    last_updated DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (charity_number, name),
    FOREIGN KEY (charity_number) REFERENCES charities(registered_number)
);