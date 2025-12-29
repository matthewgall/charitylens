-- Add support for organisation_number and linked charities
-- The Charity Commission data model has:
--   - organisation_number: Unique ID for each organization (PRIMARY KEY)
--   - registered_charity_number: The charity registration number (can be shared by linked charities)
--   - linked_charity_number: 0 = main charity, 1+ = subsidiary/linked charities

-- Recreate charities table with organisation_number as primary key
-- This allows storing multiple organizations per registered_charity_number

-- Step 1: Rename old table
ALTER TABLE charities RENAME TO charities_old;

-- Step 2: Create new table with organisation_number as PK
CREATE TABLE charities (
    organisation_number INTEGER PRIMARY KEY,
    registered_number INTEGER NOT NULL,
    linked_charity_number INTEGER DEFAULT 0,
    company_number TEXT,
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

-- Step 3: Copy data from old table (set organisation_number = registered_number for old data)
-- This preserves existing data but won't have the proper org numbers until re-imported
INSERT INTO charities (
    organisation_number, registered_number, linked_charity_number, company_number,
    name, status, date_registered, date_removed, address, website, email, phone,
    what_the_charity_does, who_the_charity_helps, how_the_charity_works, last_updated
)
SELECT 
    registered_number, registered_number, 0, company_number,
    name, status, date_registered, date_removed, address, website, email, phone,
    what_the_charity_does, who_the_charity_helps, how_the_charity_works, last_updated
FROM charities_old;

-- Step 4: Drop old table
DROP TABLE charities_old;

-- Step 5: Create indexes
CREATE INDEX idx_charities_registered_number ON charities(registered_number);
CREATE INDEX idx_charities_reg_linked ON charities(registered_number, linked_charity_number);
CREATE INDEX idx_charities_name ON charities(name);

-- Step 6: Update foreign key references in other tables
-- Update financials to use registered_number (which is what it already uses)
-- No changes needed as it references charity_number which maps to registered_number

-- Update trustees to use registered_number
-- No changes needed as it references charity_number which maps to registered_number

-- Update charity_scores to use registered_number
-- No changes needed as it references charity_number which maps to registered_number

-- Note: For lookups, queries should:
--   - Use registered_number for the main charity number (for backwards compatibility)
--   - Filter by linked_charity_number = 0 to get the main charity
--   - Query all linked_charity_number values to get subsidiaries/linked entities
