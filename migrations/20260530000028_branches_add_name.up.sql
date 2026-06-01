-- Extend auth7 branches with a denormalized branch_name.
--
-- Originally migration 20260530000002 dropped name as part of the "minimal
-- projection" rebaseline (branch hierarchy owned by enterprise domain). In
-- practice the shell dropdown needs a human label and a per-render
-- BFF→enterprise lookup is too chatty. Carry name here as a snapshot;
-- the enterprise→auth7 push subscription (future, event-driven) will
-- keep it in sync.

ALTER TABLE branches ADD COLUMN IF NOT EXISTS name VARCHAR(255) NOT NULL DEFAULT '';

-- Backfill the 5 seeded enterprise-synced branches with their canonical
-- names from seed/demo/seed_002_branches.sql (uuid5-derived ids).
UPDATE branches SET name = 'KANTOR PUSAT BJB SYARIAH'  WHERE id = '5c7850c8-0c4e-5e5b-b899-c7b933122888';
UPDATE branches SET name = 'KANTOR CABANG BANDUNG'    WHERE id = '30f4c0e9-0540-5aa7-937f-d620f2cc6293';
UPDATE branches SET name = 'KANTOR CABANG TASIKMALAYA' WHERE id = '11cdfdbe-a90c-5b67-bdd7-866c802a0875';
UPDATE branches SET name = 'KANTOR CABANG CIREBON'    WHERE id = '7f2a153c-71a4-5f70-8452-fe7aaac58bee';
UPDATE branches SET name = 'KANTOR CABANG BOGOR'      WHERE id = 'bc45b517-5d40-5838-aa48-d9c4b6e459a5';
