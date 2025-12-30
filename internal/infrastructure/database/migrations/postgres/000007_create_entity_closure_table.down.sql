-- Remove entity_closure table and indexes
DROP INDEX IF EXISTS idx_entity_closure_depth;
DROP INDEX IF EXISTS idx_entity_closure_ancestor;
DROP INDEX IF EXISTS idx_entity_closure_descendant;
DROP TABLE IF EXISTS entity_closure;
