-- Drop indexes
DROP INDEX IF EXISTS idx_schemas_tenant_created;
DROP INDEX IF EXISTS idx_schemas_version;

-- Drop the UNIQUE constraint on (tenant_id, version)
ALTER TABLE schemas DROP CONSTRAINT IF EXISTS schemas_tenant_version_unique;

-- Restore the old UNIQUE constraint on tenant_id only
ALTER TABLE schemas ADD CONSTRAINT schemas_tenant_id_key UNIQUE (tenant_id);

-- Drop version column
ALTER TABLE schemas DROP COLUMN IF EXISTS version;
