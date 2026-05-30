DROP INDEX IF EXISTS idx_branches_branch_type;
DROP INDEX IF EXISTS idx_branches_area_id;
DROP INDEX IF EXISTS idx_branches_parent_branch_id;

ALTER TABLE branches
    DROP COLUMN IF EXISTS branch_classification,
    DROP COLUMN IF EXISTS area_id,
    DROP COLUMN IF EXISTS parent_branch_id,
    DROP COLUMN IF EXISTS branch_type;
