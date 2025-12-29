-- Add company_number field to charities table to track trading subsidiaries
ALTER TABLE charities ADD COLUMN company_number TEXT;