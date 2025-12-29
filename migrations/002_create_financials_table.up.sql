-- SQLite schema for financials table
CREATE TABLE IF NOT EXISTS financials (
    charity_number INTEGER NOT NULL,
    financial_year_end DATETIME NOT NULL,
    total_income REAL,
    total_spending REAL,
    charitable_activities_spend REAL,
    raising_funds_spend REAL,
    other_spend REAL,
    reserves REAL,
    assets REAL,
    employees INTEGER,
    volunteers INTEGER,
    trustees INTEGER,
    last_updated DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (charity_number, financial_year_end),
    FOREIGN KEY (charity_number) REFERENCES charities(registered_number)
);