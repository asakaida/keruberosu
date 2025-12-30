-- Create entity_closure table for pre-computed ancestor relationships
-- This enables O(1) ancestor lookups instead of recursive CTE traversal

CREATE TABLE IF NOT EXISTS entity_closure (
    tenant_id TEXT NOT NULL,
    descendant_type TEXT NOT NULL,
    descendant_id TEXT NOT NULL,
    ancestor_type TEXT NOT NULL,
    ancestor_id TEXT NOT NULL,
    depth INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (tenant_id, descendant_type, descendant_id, ancestor_type, ancestor_id)
);

-- Index for LookupSubject: find all ancestors of a descendant
CREATE INDEX idx_entity_closure_descendant
ON entity_closure (tenant_id, descendant_type, descendant_id);

-- Index for finding all descendants of an ancestor (useful for deletion)
CREATE INDEX idx_entity_closure_ancestor
ON entity_closure (tenant_id, ancestor_type, ancestor_id);

-- Index for depth-based queries
CREATE INDEX idx_entity_closure_depth
ON entity_closure (tenant_id, descendant_type, descendant_id, depth);
