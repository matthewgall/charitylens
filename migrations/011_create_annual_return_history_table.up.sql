CREATE TABLE IF NOT EXISTS annual_return_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    organisation_number INTEGER NOT NULL,
    registered_charity_number INTEGER NOT NULL,
    fin_period_start_date DATETIME,
    fin_period_end_date DATETIME,
    ar_cycle_reference VARCHAR(10),
    reporting_due_date DATETIME,
    date_annual_return_received DATETIME,
    date_accounts_received DATETIME,
    total_gross_income DECIMAL(15,2),
    total_gross_expenditure DECIMAL(15,2),
    accounts_qualified BOOLEAN,
    suppression_ind BOOLEAN,
    suppression_type VARCHAR(50),
    date_of_extract DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_arh_charity_number ON annual_return_history(registered_charity_number);
CREATE INDEX IF NOT EXISTS idx_arh_organisation_number ON annual_return_history(organisation_number);
CREATE INDEX IF NOT EXISTS idx_arh_fin_period_end ON annual_return_history(fin_period_end_date);
CREATE INDEX IF NOT EXISTS idx_arh_ar_cycle ON annual_return_history(ar_cycle_reference);
