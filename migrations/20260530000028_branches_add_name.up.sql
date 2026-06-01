-- Extend auth7 branches with a denormalized branch_name.
--
-- Originally migration 20260530000002 dropped name as part of the "minimal
-- projection" rebaseline (branch hierarchy owned by enterprise domain). In
-- practice the shell dropdown needs a human label and a per-render
-- BFF→enterprise lookup is too chatty. Carry name here as a snapshot;
-- the enterprise→auth7 push subscription (future, event-driven) will
-- keep it in sync.
--
-- For full backfill of all 70+ branches, use
-- scripts/sync-branches-from-enterprise.sh. This migration only names the
-- two seeded by migration 26 — enough for demo users to log in and see a
-- non-empty branch label in the shell.

ALTER TABLE branches ADD COLUMN IF NOT EXISTS name VARCHAR(255) NOT NULL DEFAULT '';

-- Backfill the 2 seeded demo branches (canonical enterprise UUIDs +
-- canonical names from core7_enterprise.branches).
UPDATE branches SET name = 'KANTOR PUSAT BANK DEMO' WHERE id = 'd0000000-0000-0000-0000-000000000000';
UPDATE branches SET name = 'KANTOR CABANG BANDUNG'  WHERE id = 'd0000000-0000-0000-0000-000000000001';
