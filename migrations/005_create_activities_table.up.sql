-- SQLite schema for activities table
CREATE TABLE IF NOT EXISTS activities (
    charity_number INTEGER NOT NULL,
    description TEXT NOT NULL,
    last_updated DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (charity_number, description),
    FOREIGN KEY (charity_number) REFERENCES charities(registered_number)
);