-- Create search_cache table to track when searches were last performed
CREATE TABLE IF NOT EXISTS search_cache (
    query TEXT NOT NULL,
    search_type TEXT NOT NULL,
    last_searched DATETIME DEFAULT CURRENT_TIMESTAMP,
    result_count INTEGER DEFAULT 0,
    PRIMARY KEY (query, search_type)
);

CREATE INDEX IF NOT EXISTS idx_search_cache_last_searched ON search_cache(last_searched);
