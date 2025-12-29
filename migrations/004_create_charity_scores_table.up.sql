-- SQLite schema for charity_scores table
CREATE TABLE IF NOT EXISTS charity_scores (
    charity_number INTEGER PRIMARY KEY,
    overall_score REAL,
    efficiency_score REAL,
    financial_health_score REAL,
    transparency_score REAL,
    governance_score REAL,
    confidence_level TEXT,
    last_calculated DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (charity_number) REFERENCES charities(registered_number)
);