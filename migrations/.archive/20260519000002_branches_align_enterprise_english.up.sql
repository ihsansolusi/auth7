-- Migration: align auth7.branches with the English enterprise schema (Plan 13 W2 follow-up)
--
-- Adds columns mirrored from core7-service-enterprise.branches (migration 000006):
--   branch_type           ex-`tipe_cabang` translated to English (9 values)
--   parent_branch_id      ex-`lcabanginduk` hierarchical parent
--   area_id               ex-`lareacabang` (FK target in enterprise.branch_areas)
--   branch_classification ex-`status_cabang` (MAIN_BRANCH / SUB_BRANCH / HEAD_OFFICE / SYARIAH_SERVICE_UNIT)
--
-- The new columns are nullable: source-contract polling fills them lazily once the
-- enterprise upstream is producing the new English DTO (see source_contract.go in
-- core7-service-enterprise after Phase 1.T06).

ALTER TABLE branches
    ADD COLUMN IF NOT EXISTS branch_type            VARCHAR(30),
    ADD COLUMN IF NOT EXISTS parent_branch_id       UUID,
    ADD COLUMN IF NOT EXISTS area_id                UUID,
    ADD COLUMN IF NOT EXISTS branch_classification  VARCHAR(30);

CREATE INDEX IF NOT EXISTS idx_branches_parent_branch_id ON branches(parent_branch_id);
CREATE INDEX IF NOT EXISTS idx_branches_area_id          ON branches(area_id);
CREATE INDEX IF NOT EXISTS idx_branches_branch_type      ON branches(branch_type);

-- Self-FK is intentionally NOT added: auth7 mirrors the enterprise hierarchy lazily
-- and may have an out-of-band parent reference before it has been synced locally.
-- Application-layer validation enforces referential integrity instead.
