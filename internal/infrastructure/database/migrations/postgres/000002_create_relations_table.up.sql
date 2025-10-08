CREATE TABLE IF NOT EXISTS relations (
    id BIGSERIAL PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    entity_type VARCHAR(255) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    relation VARCHAR(255) NOT NULL,
    subject_type VARCHAR(255) NOT NULL,
    subject_id VARCHAR(255) NOT NULL,
    subject_relation VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_relations_unique ON relations(
    tenant_id, entity_type, entity_id, relation, subject_type, subject_id, COALESCE(subject_relation, '')
);

CREATE INDEX idx_relations_entity ON relations(tenant_id, entity_type, entity_id);
CREATE INDEX idx_relations_subject ON relations(tenant_id, subject_type, subject_id);
CREATE INDEX idx_relations_lookup ON relations(tenant_id, entity_type, relation, subject_type, subject_id);
