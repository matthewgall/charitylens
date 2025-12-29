-- Scraper checkpoints for resumable scraping
CREATE TABLE IF NOT EXISTS scraper_checkpoints (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    last_charity_number INTEGER NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
