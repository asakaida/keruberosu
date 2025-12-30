-- Create transactions table for snapshot token management
-- This table tracks write operations and provides snapshot tokens for cache invalidation
CREATE TABLE IF NOT EXISTS transactions (
    id BIGSERIAL PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transactions_tenant ON transactions(tenant_id);
CREATE INDEX idx_transactions_created ON transactions(created_at DESC);

-- Create function to notify on snapshot changes
-- This function is called by triggers when data changes occur
CREATE OR REPLACE FUNCTION notify_snapshot_change()
RETURNS TRIGGER AS $$
BEGIN
    -- Notify all listeners with the new transaction ID
    PERFORM pg_notify('snapshot_changed', NEW.id::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger on transactions table
-- Fires after each new transaction is inserted
CREATE TRIGGER snapshot_change_trigger
AFTER INSERT ON transactions
FOR EACH ROW EXECUTE FUNCTION notify_snapshot_change();

-- Create function to automatically insert transaction record
-- This is called by triggers on relations and attributes tables
CREATE OR REPLACE FUNCTION insert_transaction_record()
RETURNS TRIGGER AS $$
BEGIN
    -- Insert a new transaction record to track this change
    INSERT INTO transactions (tenant_id) VALUES (NEW.tenant_id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create function for delete operations
CREATE OR REPLACE FUNCTION insert_transaction_record_on_delete()
RETURNS TRIGGER AS $$
BEGIN
    -- Insert a new transaction record to track this deletion
    INSERT INTO transactions (tenant_id) VALUES (OLD.tenant_id);
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

-- Add triggers to relations table
CREATE TRIGGER relations_insert_transaction
AFTER INSERT ON relations
FOR EACH ROW EXECUTE FUNCTION insert_transaction_record();

CREATE TRIGGER relations_update_transaction
AFTER UPDATE ON relations
FOR EACH ROW EXECUTE FUNCTION insert_transaction_record();

CREATE TRIGGER relations_delete_transaction
AFTER DELETE ON relations
FOR EACH ROW EXECUTE FUNCTION insert_transaction_record_on_delete();

-- Add triggers to attributes table
CREATE TRIGGER attributes_insert_transaction
AFTER INSERT ON attributes
FOR EACH ROW EXECUTE FUNCTION insert_transaction_record();

CREATE TRIGGER attributes_update_transaction
AFTER UPDATE ON attributes
FOR EACH ROW EXECUTE FUNCTION insert_transaction_record();

CREATE TRIGGER attributes_delete_transaction
AFTER DELETE ON attributes
FOR EACH ROW EXECUTE FUNCTION insert_transaction_record_on_delete();
