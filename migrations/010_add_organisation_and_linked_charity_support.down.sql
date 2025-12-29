-- Rollback organisation_number and linked_charity support

-- Recreate old charities table structure
ALTER TABLE charities RENAME TO charities_new;

CREATE TABLE charities (
    registered_number INTEGER PRIMARY KEY,
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

-- Copy back main charities only (linked_charity_number = 0)
INSERT INTO charities (
    registered_number, company_number, name, status, date_registered, date_removed,
    address, website, email, phone, what_the_charity_does, who_the_charity_helps,
    how_the_charity_works, last_updated
)
SELECT 
    registered_number, company_number, name, status, date_registered, date_removed,
    address, website, email, phone, what_the_charity_does, who_the_charity_helps,
    how_the_charity_works, last_updated
FROM charities_new
WHERE linked_charity_number = 0;

DROP TABLE charities_new;
