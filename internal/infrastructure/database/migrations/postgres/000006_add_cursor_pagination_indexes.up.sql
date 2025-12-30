-- Add indexes for cursor-based pagination on relations table
CREATE INDEX IF NOT EXISTS idx_relations_cursor
ON relations (tenant_id, entity_type, entity_id, created_at DESC);

-- Add index for subject lookups
CREATE INDEX IF NOT EXISTS idx_relations_subject_cursor
ON relations (tenant_id, subject_type, subject_id, created_at DESC);
