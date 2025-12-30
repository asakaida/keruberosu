-- Remove triggers from attributes table
DROP TRIGGER IF EXISTS attributes_delete_transaction ON attributes;
DROP TRIGGER IF EXISTS attributes_update_transaction ON attributes;
DROP TRIGGER IF EXISTS attributes_insert_transaction ON attributes;

-- Remove triggers from relations table
DROP TRIGGER IF EXISTS relations_delete_transaction ON relations;
DROP TRIGGER IF EXISTS relations_update_transaction ON relations;
DROP TRIGGER IF EXISTS relations_insert_transaction ON relations;

-- Remove trigger from transactions table
DROP TRIGGER IF EXISTS snapshot_change_trigger ON transactions;

-- Remove functions
DROP FUNCTION IF EXISTS insert_transaction_record_on_delete();
DROP FUNCTION IF EXISTS insert_transaction_record();
DROP FUNCTION IF EXISTS notify_snapshot_change();

-- Remove indexes and table
DROP INDEX IF EXISTS idx_transactions_created;
DROP INDEX IF EXISTS idx_transactions_tenant;
DROP TABLE IF EXISTS transactions;
