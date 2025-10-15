-- Add version column to schemas table
ALTER TABLE schemas ADD COLUMN version VARCHAR(26);

-- Set a default version for existing rows (using a fixed ULID-like value for migration)
-- This ensures existing schemas have a version
UPDATE schemas SET version = '01000000000000000000000000' WHERE version IS NULL;

-- Make version column NOT NULL after setting default values
ALTER TABLE schemas ALTER COLUMN version SET NOT NULL;

-- Drop the old UNIQUE constraint on tenant_id only
ALTER TABLE schemas DROP CONSTRAINT IF EXISTS schemas_tenant_id_key;

-- Add new UNIQUE constraint on (tenant_id, version)
-- This allows multiple versions per tenant
ALTER TABLE schemas ADD CONSTRAINT schemas_tenant_version_unique UNIQUE (tenant_id, version);

-- Add index on version for fast lookups
CREATE INDEX idx_schemas_version ON schemas(version);

-- Add index on (tenant_id, created_at DESC) for getting latest version efficiently
CREATE INDEX idx_schemas_tenant_created ON schemas(tenant_id, created_at DESC);
