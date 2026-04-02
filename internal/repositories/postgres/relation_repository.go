package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/asakaida/keruberosu/internal/entities"
	"github.com/asakaida/keruberosu/internal/infrastructure/database"
	"github.com/asakaida/keruberosu/internal/repositories"
	"github.com/lib/pq"
)

const (
	// maxUsersetDepth is the maximum recursion depth for computed userset expansion in SQL queries.
	maxUsersetDepth = 10
)

// PostgresRelationRepository implements RelationRepository using PostgreSQL
type PostgresRelationRepository struct {
	cluster                  *database.DBCluster
	closureExcludedRelations map[string]bool
}

// NewPostgresRelationRepository creates a new PostgreSQL relation repository
func NewPostgresRelationRepository(cluster *database.DBCluster, closureExcluded map[string]bool) repositories.RelationRepository {
	if closureExcluded == nil {
		closureExcluded = make(map[string]bool)
	}
	return &PostgresRelationRepository{
		cluster:                  cluster,
		closureExcludedRelations: closureExcluded,
	}
}

// Write creates a new relation tuple and updates the closure table.
func (r *PostgresRelationRepository) Write(ctx context.Context, tenantID string, tuple *entities.RelationTuple) error {
	if err := tuple.Validate(); err != nil {
		return fmt.Errorf("invalid relation tuple: %w", err)
	}

	tx, err := r.cluster.PrimaryDB().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO relations (
			tenant_id, entity_type, entity_id, relation,
			subject_type, subject_id, subject_relation, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (tenant_id, entity_type, entity_id, relation, subject_type, subject_id, COALESCE(subject_relation, ''))
		DO NOTHING
	`
	now := time.Now()
	_, err = tx.ExecContext(ctx, query,
		tenantID, tuple.EntityType, tuple.EntityID, tuple.Relation,
		tuple.SubjectType, tuple.SubjectID, sql.NullString{String: tuple.SubjectRelation, Valid: tuple.SubjectRelation != ""}, now,
	)
	if err != nil {
		return fmt.Errorf("failed to write relation: %w", err)
	}

	if !r.closureExcludedRelations[tuple.Relation] && tuple.SubjectRelation == "" {
		if err := r.updateClosureOnAdd(ctx, tx, tenantID, tuple.EntityType, tuple.EntityID, tuple.SubjectType, tuple.SubjectID); err != nil {
			return fmt.Errorf("failed to update closure table: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.cluster.RecordWrite(tenantID)
	return nil
}

// Delete removes a relation tuple and updates the closure table.
func (r *PostgresRelationRepository) Delete(ctx context.Context, tenantID string, tuple *entities.RelationTuple) error {
	if err := tuple.Validate(); err != nil {
		return fmt.Errorf("invalid relation tuple: %w", err)
	}

	tx, err := r.cluster.PrimaryDB().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		DELETE FROM relations
		WHERE tenant_id = $1
			AND entity_type = $2
			AND entity_id = $3
			AND relation = $4
			AND subject_type = $5
			AND subject_id = $6
			AND COALESCE(subject_relation, '') = $7
	`
	_, err = tx.ExecContext(ctx, query,
		tenantID, tuple.EntityType, tuple.EntityID, tuple.Relation,
		tuple.SubjectType, tuple.SubjectID, tuple.SubjectRelation,
	)
	if err != nil {
		return fmt.Errorf("failed to delete relation: %w", err)
	}

	if !r.closureExcludedRelations[tuple.Relation] {
		if err := r.updateClosureOnDelete(ctx, tx, tenantID, tuple.EntityType, tuple.EntityID, tuple.SubjectType, tuple.SubjectID); err != nil {
			return fmt.Errorf("failed to update closure table: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.cluster.RecordWrite(tenantID)
	return nil
}

// Read retrieves relation tuples matching the filter
func (r *PostgresRelationRepository) Read(ctx context.Context, tenantID string, filter *repositories.RelationFilter) ([]*entities.RelationTuple, error) {
	query := `
		SELECT entity_type, entity_id, relation, subject_type, subject_id, subject_relation, created_at
		FROM relations
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}
	argIdx := 2

	if filter != nil {
		if filter.EntityType != "" {
			query += fmt.Sprintf(" AND entity_type = $%d", argIdx)
			args = append(args, filter.EntityType)
			argIdx++
		}
		if filter.EntityID != "" {
			query += fmt.Sprintf(" AND entity_id = $%d", argIdx)
			args = append(args, filter.EntityID)
			argIdx++
		}
		if filter.Relation != "" {
			query += fmt.Sprintf(" AND relation = $%d", argIdx)
			args = append(args, filter.Relation)
			argIdx++
		}
		if filter.SubjectType != "" {
			query += fmt.Sprintf(" AND subject_type = $%d", argIdx)
			args = append(args, filter.SubjectType)
			argIdx++
		}
		if filter.SubjectID != "" {
			query += fmt.Sprintf(" AND subject_id = $%d", argIdx)
			args = append(args, filter.SubjectID)
			argIdx++
		}
		if filter.SubjectRelation != "" {
			query += fmt.Sprintf(" AND subject_relation = $%d", argIdx)
			args = append(args, filter.SubjectRelation)
			argIdx++
		}
	}

	db := r.cluster.ReaderFor(tenantID)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to read relations: %w", err)
	}
	defer rows.Close()

	return scanTuples(rows)
}

// CheckExists checks if a specific relation tuple exists (kept for backward compatibility)
func (r *PostgresRelationRepository) CheckExists(ctx context.Context, tenantID string, tuple *entities.RelationTuple) (bool, error) {
	return r.Exists(ctx, tenantID, tuple)
}

// Exists checks if a specific relation tuple exists
func (r *PostgresRelationRepository) Exists(ctx context.Context, tenantID string, tuple *entities.RelationTuple) (bool, error) {
	if err := tuple.Validate(); err != nil {
		return false, fmt.Errorf("invalid relation tuple: %w", err)
	}

	query := `
		SELECT EXISTS(
			SELECT 1 FROM relations
			WHERE tenant_id = $1
				AND entity_type = $2
				AND entity_id = $3
				AND relation = $4
				AND subject_type = $5
				AND subject_id = $6
				AND COALESCE(subject_relation, '') = $7
		)
	`
	db := r.cluster.ReaderFor(tenantID)
	var exists bool
	err := db.QueryRowContext(ctx, query,
		tenantID, tuple.EntityType, tuple.EntityID, tuple.Relation,
		tuple.SubjectType, tuple.SubjectID, tuple.SubjectRelation,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check relation existence: %w", err)
	}

	return exists, nil
}

// ExistsWithSubjectRelation checks existence including subject relation
func (r *PostgresRelationRepository) ExistsWithSubjectRelation(ctx context.Context, tenantID string,
	entityType, entityID, relation, subjectType, subjectID, subjectRelation string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM relations
			WHERE tenant_id = $1
				AND entity_type = $2
				AND entity_id = $3
				AND relation = $4
				AND subject_type = $5
				AND subject_id = $6
				AND subject_relation = $7
		)
	`
	db := r.cluster.ReaderFor(tenantID)
	var exists bool
	err := db.QueryRowContext(ctx, query,
		tenantID, entityType, entityID, relation, subjectType, subjectID, subjectRelation,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check relation existence with subject relation: %w", err)
	}
	return exists, nil
}

// FindByEntityWithRelation returns tuples for a specific entity and relation
func (r *PostgresRelationRepository) FindByEntityWithRelation(ctx context.Context, tenantID string,
	entityType, entityID, relation string, limit int) ([]*entities.RelationTuple, error) {
	query := `
		SELECT entity_type, entity_id, relation, subject_type, subject_id, subject_relation, created_at
		FROM relations
		WHERE tenant_id = $1 AND entity_type = $2 AND entity_id = $3 AND relation = $4
	`
	args := []interface{}{tenantID, entityType, entityID, relation}
	if limit > 0 {
		query += " LIMIT $5"
		args = append(args, limit)
	}
	db := r.cluster.ReaderFor(tenantID)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to find relations by entity with relation: %w", err)
	}
	defer rows.Close()

	return scanTuples(rows)
}

// LookupAncestorsViaRelation finds all ancestors via closure table
func (r *PostgresRelationRepository) LookupAncestorsViaRelation(ctx context.Context, tenantID string,
	entityType, entityID string, maxDepth int) ([]*entities.RelationTuple, error) {
	query := `
		SELECT c.ancestor_type, c.ancestor_id, c.depth
		FROM entity_closure c
		WHERE c.tenant_id = $1 AND c.descendant_type = $2 AND c.descendant_id = $3
	`
	args := []interface{}{tenantID, entityType, entityID}

	if maxDepth > 0 {
		query += " AND c.depth <= $4"
		args = append(args, maxDepth)
	}

	query += " ORDER BY c.depth"

	db := r.cluster.ReaderFor(tenantID)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup ancestors via relation: %w", err)
	}
	defer rows.Close()

	var tuples []*entities.RelationTuple
	for rows.Next() {
		var t entities.RelationTuple
		var depth int
		if err := rows.Scan(&t.SubjectType, &t.SubjectID, &depth); err != nil {
			return nil, fmt.Errorf("failed to scan ancestor: %w", err)
		}
		t.EntityType = entityType
		t.EntityID = entityID
		tuples = append(tuples, &t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating ancestors: %w", err)
	}
	return tuples, nil
}

// FindHierarchicalWithSubject checks if a subject exists in the hierarchy using recursive CTE
func (r *PostgresRelationRepository) FindHierarchicalWithSubject(ctx context.Context, tenantID string,
	entityType, entityID, relation, subjectType, subjectID string,
	maxDepth int) (bool, error) {
	query := `
		WITH RECURSIVE hierarchy AS (
			SELECT subject_type, subject_id, 1 AS depth
			FROM relations
			WHERE tenant_id = $1
				AND entity_type = $2
				AND entity_id = $3
				AND relation = $4
				AND COALESCE(subject_relation, '') = ''
			UNION ALL
			SELECT r.subject_type, r.subject_id, h.depth + 1
			FROM relations r
			INNER JOIN hierarchy h ON r.entity_type = h.subject_type AND r.entity_id = h.subject_id
			WHERE r.tenant_id = $1
				AND r.relation = $4
				AND COALESCE(r.subject_relation, '') = ''
				AND h.depth < $5
		)
		SELECT EXISTS(
			SELECT 1 FROM hierarchy
			WHERE subject_type = $6 AND subject_id = $7
		)
	`
	db := r.cluster.ReaderFor(tenantID)
	var exists bool
	err := db.QueryRowContext(ctx, query,
		tenantID, entityType, entityID, relation, maxDepth,
		subjectType, subjectID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to find hierarchical with subject: %w", err)
	}
	return exists, nil
}

// RebuildClosure rebuilds the closure table for a tenant from scratch.
// Uses a fixed-point iteration: repeatedly applies updateClosureOnAdd for all
// relations until no new closure entries are created. This makes the result
// independent of the order in which relations are processed.
func (r *PostgresRelationRepository) RebuildClosure(ctx context.Context, tenantID string) error {
	tx, err := r.cluster.PrimaryDB().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete all closure entries for this tenant
	_, err = tx.ExecContext(ctx, "DELETE FROM entity_closure WHERE tenant_id = $1", tenantID)
	if err != nil {
		return fmt.Errorf("failed to clear closure table: %w", err)
	}

	// Read all relations for this tenant (exclude subject_relation tuples -
	// computed usersets are not hierarchical parents and should not be in the closure table)
	rows, err := tx.QueryContext(ctx, `
		SELECT entity_type, entity_id, relation, subject_type, subject_id
		FROM relations WHERE tenant_id = $1 AND COALESCE(subject_relation, '') = ''
	`, tenantID)
	if err != nil {
		return fmt.Errorf("failed to read relations: %w", err)
	}
	defer rows.Close()

	type rel struct {
		entityType, entityID, relation, subjectType, subjectID string
	}
	var rels []rel
	for rows.Next() {
		var r rel
		if err := rows.Scan(&r.entityType, &r.entityID, &r.relation, &r.subjectType, &r.subjectID); err != nil {
			return fmt.Errorf("failed to scan relation: %w", err)
		}
		rels = append(rels, r)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating relations: %w", err)
	}

	// Filter to non-excluded relations
	var filteredRels []rel
	for _, rl := range rels {
		if !r.closureExcludedRelations[rl.relation] {
			filteredRels = append(filteredRels, rl)
		}
	}

	// Fixed-point iteration: repeat until no new entries are created.
	// Each pass may discover new transitive paths via entries created in previous passes.
	const maxIterations = 100
	for iter := 0; iter < maxIterations; iter++ {
		var countBefore int
		err := tx.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM entity_closure WHERE tenant_id = $1", tenantID).Scan(&countBefore)
		if err != nil {
			return fmt.Errorf("failed to count closure entries: %w", err)
		}

		for _, rl := range filteredRels {
			if err := r.updateClosureOnAdd(ctx, tx, tenantID, rl.entityType, rl.entityID, rl.subjectType, rl.subjectID); err != nil {
				return fmt.Errorf("failed to rebuild closure for %s:%s -> %s:%s: %w",
					rl.entityType, rl.entityID, rl.subjectType, rl.subjectID, err)
			}
		}

		var countAfter int
		err = tx.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM entity_closure WHERE tenant_id = $1", tenantID).Scan(&countAfter)
		if err != nil {
			return fmt.Errorf("failed to count closure entries: %w", err)
		}

		if countAfter == countBefore {
			break // Fixed point reached
		}
	}

	return tx.Commit()
}

// BatchWrite creates multiple relation tuples in a single transaction and updates the closure table.
func (r *PostgresRelationRepository) BatchWrite(ctx context.Context, tenantID string, tuples []*entities.RelationTuple) error {
	if len(tuples) == 0 {
		return nil
	}

	tx, err := r.cluster.PrimaryDB().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO relations (
			tenant_id, entity_type, entity_id, relation,
			subject_type, subject_id, subject_relation, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (tenant_id, entity_type, entity_id, relation, subject_type, subject_id, COALESCE(subject_relation, ''))
		DO NOTHING
	`
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, tuple := range tuples {
		if err := tuple.Validate(); err != nil {
			return fmt.Errorf("invalid relation tuple: %w", err)
		}

		_, err := stmt.ExecContext(ctx,
			tenantID, tuple.EntityType, tuple.EntityID, tuple.Relation,
			tuple.SubjectType, tuple.SubjectID, sql.NullString{String: tuple.SubjectRelation, Valid: tuple.SubjectRelation != ""}, now,
		)
		if err != nil {
			return fmt.Errorf("failed to write relation: %w", err)
		}

		if !r.closureExcludedRelations[tuple.Relation] && tuple.SubjectRelation == "" {
			if err := r.updateClosureOnAdd(ctx, tx, tenantID, tuple.EntityType, tuple.EntityID, tuple.SubjectType, tuple.SubjectID); err != nil {
				return fmt.Errorf("failed to update closure table: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.cluster.RecordWrite(tenantID)
	return nil
}

// BatchWriteInTx creates multiple relation tuples within an existing transaction.
func (r *PostgresRelationRepository) BatchWriteInTx(ctx context.Context, tx *sql.Tx, tenantID string, tuples []*entities.RelationTuple) error {
	query := `
		INSERT INTO relations (
			tenant_id, entity_type, entity_id, relation,
			subject_type, subject_id, subject_relation, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (tenant_id, entity_type, entity_id, relation, subject_type, subject_id, COALESCE(subject_relation, ''))
		DO NOTHING
	`
	now := time.Now()
	for _, tuple := range tuples {
		if err := tuple.Validate(); err != nil {
			return fmt.Errorf("invalid relation tuple: %w", err)
		}
		_, err := tx.ExecContext(ctx, query,
			tenantID, tuple.EntityType, tuple.EntityID, tuple.Relation,
			tuple.SubjectType, tuple.SubjectID,
			sql.NullString{String: tuple.SubjectRelation, Valid: tuple.SubjectRelation != ""},
			now,
		)
		if err != nil {
			return fmt.Errorf("failed to write relation: %w", err)
		}
		if !r.closureExcludedRelations[tuple.Relation] && tuple.SubjectRelation == "" {
			if err := r.updateClosureOnAdd(ctx, tx, tenantID, tuple.EntityType, tuple.EntityID, tuple.SubjectType, tuple.SubjectID); err != nil {
				return fmt.Errorf("failed to update closure table: %w", err)
			}
		}
	}

	return nil
}

// BatchDelete removes multiple relation tuples in a single transaction and updates the closure table.
func (r *PostgresRelationRepository) BatchDelete(ctx context.Context, tenantID string, tuples []*entities.RelationTuple) error {
	if len(tuples) == 0 {
		return nil
	}

	tx, err := r.cluster.PrimaryDB().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		DELETE FROM relations
		WHERE tenant_id = $1
			AND entity_type = $2
			AND entity_id = $3
			AND relation = $4
			AND subject_type = $5
			AND subject_id = $6
			AND COALESCE(subject_relation, '') = $7
	`
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, tuple := range tuples {
		if err := tuple.Validate(); err != nil {
			return fmt.Errorf("invalid relation tuple: %w", err)
		}

		_, err := stmt.ExecContext(ctx,
			tenantID, tuple.EntityType, tuple.EntityID, tuple.Relation,
			tuple.SubjectType, tuple.SubjectID, tuple.SubjectRelation,
		)
		if err != nil {
			return fmt.Errorf("failed to delete relation: %w", err)
		}

		if !r.closureExcludedRelations[tuple.Relation] {
			if err := r.updateClosureOnDelete(ctx, tx, tenantID, tuple.EntityType, tuple.EntityID, tuple.SubjectType, tuple.SubjectID); err != nil {
				return fmt.Errorf("failed to update closure table: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.cluster.RecordWrite(tenantID)
	return nil
}

// DeleteByFilter removes relation tuples matching the filter and updates the closure table.
func (r *PostgresRelationRepository) DeleteByFilter(ctx context.Context, tenantID string, filter *repositories.RelationFilter) error {
	if filter == nil {
		return fmt.Errorf("filter is required")
	}

	tx, err := r.cluster.PrimaryDB().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// First, SELECT tuples that will be deleted (for closure table cleanup)
	selectQuery := `SELECT entity_type, entity_id, relation, subject_type, subject_id FROM relations WHERE tenant_id = $1`
	deleteQuery := `DELETE FROM relations WHERE tenant_id = $1`
	args := []interface{}{tenantID}
	argIdx := 2

	var conditions string
	if filter.EntityType != "" {
		conditions += fmt.Sprintf(" AND entity_type = $%d", argIdx)
		args = append(args, filter.EntityType)
		argIdx++
	}
	if len(filter.EntityIDs) > 0 {
		conditions += fmt.Sprintf(" AND entity_id = ANY($%d)", argIdx)
		args = append(args, pq.Array(filter.EntityIDs))
		argIdx++
	} else if filter.EntityID != "" {
		conditions += fmt.Sprintf(" AND entity_id = $%d", argIdx)
		args = append(args, filter.EntityID)
		argIdx++
	}
	if filter.Relation != "" {
		conditions += fmt.Sprintf(" AND relation = $%d", argIdx)
		args = append(args, filter.Relation)
		argIdx++
	}
	if filter.SubjectType != "" {
		conditions += fmt.Sprintf(" AND subject_type = $%d", argIdx)
		args = append(args, filter.SubjectType)
		argIdx++
	}
	if len(filter.SubjectIDs) > 0 {
		conditions += fmt.Sprintf(" AND subject_id = ANY($%d)", argIdx)
		args = append(args, pq.Array(filter.SubjectIDs))
		argIdx++
	} else if filter.SubjectID != "" {
		conditions += fmt.Sprintf(" AND subject_id = $%d", argIdx)
		args = append(args, filter.SubjectID)
		argIdx++
	}
	if filter.SubjectRelation != "" {
		conditions += fmt.Sprintf(" AND subject_relation = $%d", argIdx)
		args = append(args, filter.SubjectRelation)
		argIdx++
	}

	selectQuery += conditions
	deleteQuery += conditions

	// Query tuples to be deleted for closure cleanup
	rows, err := tx.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return fmt.Errorf("failed to query relations for closure cleanup: %w", err)
	}
	type tupleRef struct {
		entityType, entityID, relation, subjectType, subjectID string
	}
	var refs []tupleRef
	for rows.Next() {
		var ref tupleRef
		if err := rows.Scan(&ref.entityType, &ref.entityID, &ref.relation, &ref.subjectType, &ref.subjectID); err != nil {
			rows.Close()
			return fmt.Errorf("failed to scan tuple for closure cleanup: %w", err)
		}
		refs = append(refs, ref)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating tuples for closure cleanup: %w", err)
	}

	// Delete the tuples
	_, err = tx.ExecContext(ctx, deleteQuery, args...)
	if err != nil {
		return fmt.Errorf("failed to delete relations by filter: %w", err)
	}

	// Update closure table for each deleted tuple
	for _, ref := range refs {
		if !r.closureExcludedRelations[ref.relation] {
			if err := r.updateClosureOnDelete(ctx, tx, tenantID, ref.entityType, ref.entityID, ref.subjectType, ref.subjectID); err != nil {
				return fmt.Errorf("failed to update closure table: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.cluster.RecordWrite(tenantID)
	return nil
}

// ReadByFilter retrieves relation tuples matching filter with pagination
func (r *PostgresRelationRepository) ReadByFilter(ctx context.Context, tenantID string, filter *repositories.RelationFilter, pageSize int, pageToken string) ([]*entities.RelationTuple, string, error) {
	if pageSize <= 0 {
		pageSize = 100
	}

	query := `
		SELECT id, entity_type, entity_id, relation, subject_type, subject_id, subject_relation, created_at
		FROM relations
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}
	argIdx := 2

	if filter != nil {
		if filter.EntityType != "" {
			query += fmt.Sprintf(" AND entity_type = $%d", argIdx)
			args = append(args, filter.EntityType)
			argIdx++
		}
		if len(filter.EntityIDs) > 0 {
			query += fmt.Sprintf(" AND entity_id = ANY($%d)", argIdx)
			args = append(args, pq.Array(filter.EntityIDs))
			argIdx++
		} else if filter.EntityID != "" {
			query += fmt.Sprintf(" AND entity_id = $%d", argIdx)
			args = append(args, filter.EntityID)
			argIdx++
		}
		if filter.Relation != "" {
			query += fmt.Sprintf(" AND relation = $%d", argIdx)
			args = append(args, filter.Relation)
			argIdx++
		}
		if filter.SubjectType != "" {
			query += fmt.Sprintf(" AND subject_type = $%d", argIdx)
			args = append(args, filter.SubjectType)
			argIdx++
		}
		if len(filter.SubjectIDs) > 0 {
			query += fmt.Sprintf(" AND subject_id = ANY($%d)", argIdx)
			args = append(args, pq.Array(filter.SubjectIDs))
			argIdx++
		} else if filter.SubjectID != "" {
			query += fmt.Sprintf(" AND subject_id = $%d", argIdx)
			args = append(args, filter.SubjectID)
			argIdx++
		}
		if filter.SubjectRelation != "" {
			query += fmt.Sprintf(" AND subject_relation = $%d", argIdx)
			args = append(args, filter.SubjectRelation)
			argIdx++
		}
	}

	if pageToken != "" {
		query += fmt.Sprintf(" AND id > $%d", argIdx)
		args = append(args, pageToken)
		argIdx++
	}

	query += " ORDER BY id"
	query += fmt.Sprintf(" LIMIT $%d", argIdx)
	args = append(args, pageSize+1)

	db := r.cluster.ReaderFor(tenantID)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read relations by filter: %w", err)
	}
	defer rows.Close()

	type tupleWithID struct {
		id    int64
		tuple *entities.RelationTuple
	}
	var results []tupleWithID
	for rows.Next() {
		var tw tupleWithID
		tw.tuple = &entities.RelationTuple{}
		var subjectRelation sql.NullString
		err := rows.Scan(
			&tw.id,
			&tw.tuple.EntityType, &tw.tuple.EntityID, &tw.tuple.Relation,
			&tw.tuple.SubjectType, &tw.tuple.SubjectID, &subjectRelation, &tw.tuple.CreatedAt,
		)
		if err != nil {
			return nil, "", fmt.Errorf("failed to scan relation: %w", err)
		}
		if subjectRelation.Valid {
			tw.tuple.SubjectRelation = subjectRelation.String
		}
		results = append(results, tw)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("error iterating relations: %w", err)
	}

	var nextToken string
	if len(results) > pageSize {
		nextToken = fmt.Sprintf("%d", results[pageSize-1].id)
		results = results[:pageSize]
	}

	tuples := make([]*entities.RelationTuple, len(results))
	for i, r := range results {
		tuples[i] = r.tuple
	}

	return tuples, nextToken, nil
}

// WriteWithClosure creates a new relation tuple and updates the closure table.
// Returns a snapshot token for cache consistency.
func (r *PostgresRelationRepository) WriteWithClosure(ctx context.Context, tenantID string, tuple *entities.RelationTuple) (string, error) {
	if err := tuple.Validate(); err != nil {
		return "", fmt.Errorf("invalid relation tuple: %w", err)
	}

	tx, err := r.cluster.PrimaryDB().BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO relations (
			tenant_id, entity_type, entity_id, relation,
			subject_type, subject_id, subject_relation, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (tenant_id, entity_type, entity_id, relation, subject_type, subject_id, COALESCE(subject_relation, ''))
		DO NOTHING
	`
	now := time.Now()
	_, err = tx.ExecContext(ctx, query,
		tenantID, tuple.EntityType, tuple.EntityID, tuple.Relation,
		tuple.SubjectType, tuple.SubjectID, sql.NullString{String: tuple.SubjectRelation, Valid: tuple.SubjectRelation != ""}, now,
	)
	if err != nil {
		return "", fmt.Errorf("failed to write relation: %w", err)
	}

	if !r.closureExcludedRelations[tuple.Relation] && tuple.SubjectRelation == "" {
		err = r.updateClosureOnAdd(ctx, tx, tenantID, tuple.EntityType, tuple.EntityID, tuple.SubjectType, tuple.SubjectID)
		if err != nil {
			return "", fmt.Errorf("failed to update closure: %w", err)
		}
	}

	snapshotMgr := NewSnapshotManager(r.cluster.PrimaryDB())
	token, err := snapshotMgr.GenerateWriteToken(ctx, tx)
	if err != nil {
		return "", fmt.Errorf("failed to generate snapshot token: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.cluster.RecordWrite(tenantID)
	return token, nil
}

// DeleteWithClosure removes a relation tuple and updates the closure table.
// Returns a snapshot token for cache consistency.
func (r *PostgresRelationRepository) DeleteWithClosure(ctx context.Context, tenantID string, tuple *entities.RelationTuple) (string, error) {
	if err := tuple.Validate(); err != nil {
		return "", fmt.Errorf("invalid relation tuple: %w", err)
	}

	tx, err := r.cluster.PrimaryDB().BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		DELETE FROM relations
		WHERE tenant_id = $1
			AND entity_type = $2
			AND entity_id = $3
			AND relation = $4
			AND subject_type = $5
			AND subject_id = $6
			AND COALESCE(subject_relation, '') = $7
	`
	_, err = tx.ExecContext(ctx, query,
		tenantID, tuple.EntityType, tuple.EntityID, tuple.Relation,
		tuple.SubjectType, tuple.SubjectID, tuple.SubjectRelation,
	)
	if err != nil {
		return "", fmt.Errorf("failed to delete relation: %w", err)
	}

	if !r.closureExcludedRelations[tuple.Relation] {
		err = r.updateClosureOnDelete(ctx, tx, tenantID, tuple.EntityType, tuple.EntityID, tuple.SubjectType, tuple.SubjectID)
		if err != nil {
			return "", fmt.Errorf("failed to update closure: %w", err)
		}
	}

	snapshotMgr := NewSnapshotManager(r.cluster.PrimaryDB())
	token, err := snapshotMgr.GenerateWriteToken(ctx, tx)
	if err != nil {
		return "", fmt.Errorf("failed to generate snapshot token: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.cluster.RecordWrite(tenantID)
	return token, nil
}

// LookupAncestors finds all ancestors of an entity using the closure table.
func (r *PostgresRelationRepository) LookupAncestors(
	ctx context.Context,
	tenantID, entityType, entityID string,
	maxDepth int,
) ([]*ClosureEntry, error) {
	query := `
		SELECT ancestor_type, ancestor_id, depth
		FROM entity_closure
		WHERE tenant_id = $1 AND descendant_type = $2 AND descendant_id = $3
	`
	args := []interface{}{tenantID, entityType, entityID}

	if maxDepth > 0 {
		query += " AND depth <= $4"
		args = append(args, maxDepth)
	}

	query += " ORDER BY depth"

	db := r.cluster.ReaderFor(tenantID)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup ancestors: %w", err)
	}
	defer rows.Close()

	var entries []*ClosureEntry
	for rows.Next() {
		var entry ClosureEntry
		if err := rows.Scan(&entry.AncestorType, &entry.AncestorID, &entry.Depth); err != nil {
			return nil, fmt.Errorf("failed to scan closure entry: %w", err)
		}
		entries = append(entries, &entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating closure entries: %w", err)
	}

	return entries, nil
}

// ClosureEntry represents an entry in the entity_closure table.
type ClosureEntry struct {
	AncestorType string
	AncestorID   string
	Depth        int
}

// updateClosureOnAdd updates the entity_closure table when a new relation is added.
// When adding edge entity→subject, four sets of entries are created:
//  1. Direct entry: entity→subject at depth 1
//  2. Entity to subject's ancestors: entity→A for each ancestor A of subject
//  3. Entity's descendants to subject: D→subject for each descendant D of entity
//  4. Entity's descendants to subject's ancestors: D→A (cross product)
func (r *PostgresRelationRepository) updateClosureOnAdd(
	ctx context.Context,
	tx *sql.Tx,
	tenantID, entityType, entityID, subjectType, subjectID string,
) error {
	// Step 1: Direct entry (entity→subject, depth=1)
	directQuery := `
		INSERT INTO entity_closure (tenant_id, descendant_type, descendant_id, ancestor_type, ancestor_id, depth)
		VALUES ($1, $2, $3, $4, $5, 1)
		ON CONFLICT (tenant_id, descendant_type, descendant_id, ancestor_type, ancestor_id)
		DO UPDATE SET depth = LEAST(entity_closure.depth, EXCLUDED.depth)
	`
	_, err := tx.ExecContext(ctx, directQuery, tenantID, entityType, entityID, subjectType, subjectID)
	if err != nil {
		return fmt.Errorf("failed to insert direct closure: %w", err)
	}

	// Step 2: Entity to subject's ancestors (entity→A, depth=A.depth+1)
	entityToAncestorsQuery := `
		INSERT INTO entity_closure (tenant_id, descendant_type, descendant_id, ancestor_type, ancestor_id, depth)
		SELECT $1, $2, $3, ancestor_type, ancestor_id, depth + 1
		FROM entity_closure
		WHERE tenant_id = $1 AND descendant_type = $4 AND descendant_id = $5
		ON CONFLICT (tenant_id, descendant_type, descendant_id, ancestor_type, ancestor_id)
		DO UPDATE SET depth = LEAST(entity_closure.depth, EXCLUDED.depth)
	`
	_, err = tx.ExecContext(ctx, entityToAncestorsQuery, tenantID, entityType, entityID, subjectType, subjectID)
	if err != nil {
		return fmt.Errorf("failed to insert entity-to-ancestor closures: %w", err)
	}

	// Step 3: Entity's descendants to subject (D→subject, depth=D.depth+1)
	descendantsToSubjectQuery := `
		INSERT INTO entity_closure (tenant_id, descendant_type, descendant_id, ancestor_type, ancestor_id, depth)
		SELECT $1, descendant_type, descendant_id, $4, $5, depth + 1
		FROM entity_closure
		WHERE tenant_id = $1 AND ancestor_type = $2 AND ancestor_id = $3
		ON CONFLICT (tenant_id, descendant_type, descendant_id, ancestor_type, ancestor_id)
		DO UPDATE SET depth = LEAST(entity_closure.depth, EXCLUDED.depth)
	`
	_, err = tx.ExecContext(ctx, descendantsToSubjectQuery, tenantID, entityType, entityID, subjectType, subjectID)
	if err != nil {
		return fmt.Errorf("failed to insert descendant-to-subject closures: %w", err)
	}

	// Step 4: Entity's descendants to subject's ancestors (D→A, depth=D.depth+A.depth+1)
	crossQuery := `
		INSERT INTO entity_closure (tenant_id, descendant_type, descendant_id, ancestor_type, ancestor_id, depth)
		SELECT $1, d.descendant_type, d.descendant_id, a.ancestor_type, a.ancestor_id, d.depth + a.depth + 1
		FROM entity_closure d
		CROSS JOIN entity_closure a
		WHERE d.tenant_id = $1 AND d.ancestor_type = $2 AND d.ancestor_id = $3
		  AND a.tenant_id = $1 AND a.descendant_type = $4 AND a.descendant_id = $5
		ON CONFLICT (tenant_id, descendant_type, descendant_id, ancestor_type, ancestor_id)
		DO UPDATE SET depth = LEAST(entity_closure.depth, EXCLUDED.depth)
	`
	_, err = tx.ExecContext(ctx, crossQuery, tenantID, entityType, entityID, subjectType, subjectID)
	if err != nil {
		return fmt.Errorf("failed to insert cross closures: %w", err)
	}

	return nil
}

// updateClosureOnDelete updates the entity_closure table when a relation is deleted.
// Uses a partial rebuild strategy:
//  1. Collect all affected descendants (entity itself + its descendants in the closure table)
//  2. Delete ALL closure entries where those descendants are the descendant side
//  3. Rebuild closure entries for those descendants from the remaining relations
//
// This approach correctly handles DAGs (multiple paths) and descendant cleanup.
func (r *PostgresRelationRepository) updateClosureOnDelete(
	ctx context.Context,
	tx *sql.Tx,
	tenantID, entityType, entityID, subjectType, subjectID string,
) error {
	// Step 1: Collect all affected descendants (entity itself + entities that have entity as ancestor)
	type descEntry struct {
		descType string
		descID   string
	}
	affectedDescendants := []descEntry{{entityType, entityID}}

	rows, err := tx.QueryContext(ctx, `
		SELECT descendant_type, descendant_id
		FROM entity_closure
		WHERE tenant_id = $1 AND ancestor_type = $2 AND ancestor_id = $3
	`, tenantID, entityType, entityID)
	if err != nil {
		return fmt.Errorf("failed to query descendants: %w", err)
	}
	for rows.Next() {
		var d descEntry
		if err := rows.Scan(&d.descType, &d.descID); err != nil {
			rows.Close()
			return fmt.Errorf("failed to scan descendant: %w", err)
		}
		affectedDescendants = append(affectedDescendants, d)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating descendants: %w", err)
	}

	// Step 2: Delete ALL closure entries for affected descendants
	for _, d := range affectedDescendants {
		_, err := tx.ExecContext(ctx, `
			DELETE FROM entity_closure
			WHERE tenant_id = $1 AND descendant_type = $2 AND descendant_id = $3
		`, tenantID, d.descType, d.descID)
		if err != nil {
			return fmt.Errorf("failed to delete closure for %s:%s: %w", d.descType, d.descID, err)
		}
	}

	// Step 3: Rebuild closure entries for affected descendants from remaining relations
	for _, d := range affectedDescendants {
		relRows, err := tx.QueryContext(ctx, `
			SELECT subject_type, subject_id FROM relations
			WHERE tenant_id = $1 AND entity_type = $2 AND entity_id = $3
			  AND COALESCE(subject_relation, '') = ''
		`, tenantID, d.descType, d.descID)
		if err != nil {
			return fmt.Errorf("failed to query relations for rebuild: %w", err)
		}

		type subj struct {
			subjectType string
			subjectID   string
		}
		var subjects []subj
		for relRows.Next() {
			var s subj
			if err := relRows.Scan(&s.subjectType, &s.subjectID); err != nil {
				relRows.Close()
				return fmt.Errorf("failed to scan relation: %w", err)
			}
			subjects = append(subjects, s)
		}
		relRows.Close()
		if err := relRows.Err(); err != nil {
			return fmt.Errorf("error iterating relations: %w", err)
		}

		for _, s := range subjects {
			// Skip the just-deleted relation
			if d.descType == entityType && d.descID == entityID &&
				s.subjectType == subjectType && s.subjectID == subjectID {
				continue
			}
			if err := r.updateClosureOnAdd(ctx, tx, tenantID, d.descType, d.descID, s.subjectType, s.subjectID); err != nil {
				return fmt.Errorf("failed to rebuild closure for %s:%s -> %s:%s: %w",
					d.descType, d.descID, s.subjectType, s.subjectID, err)
			}
		}
	}

	return nil
}

// GetSortedEntityIDs returns sorted unique entity IDs with cursor-based pagination.
func (r *PostgresRelationRepository) GetSortedEntityIDs(ctx context.Context, tenantID string,
	entityType string, cursor string, limit int) ([]string, error) {
	query := `SELECT DISTINCT entity_id FROM relations WHERE tenant_id = $1 AND entity_type = $2`
	args := []interface{}{tenantID, entityType}
	argIdx := 3

	if cursor != "" {
		query += fmt.Sprintf(" AND entity_id > $%d", argIdx)
		args = append(args, cursor)
		argIdx++
	}

	query += " ORDER BY entity_id"
	query += fmt.Sprintf(" LIMIT $%d", argIdx)
	args = append(args, limit)

	db := r.cluster.ReaderFor(tenantID)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get sorted entity IDs: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan entity ID: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// GetSortedSubjectIDs returns sorted unique subject IDs with cursor-based pagination.
func (r *PostgresRelationRepository) GetSortedSubjectIDs(ctx context.Context, tenantID string,
	subjectType string, cursor string, limit int) ([]string, error) {
	query := `SELECT DISTINCT subject_id FROM relations WHERE tenant_id = $1 AND subject_type = $2`
	args := []interface{}{tenantID, subjectType}
	argIdx := 3

	if cursor != "" {
		query += fmt.Sprintf(" AND subject_id > $%d", argIdx)
		args = append(args, cursor)
		argIdx++
	}

	query += " ORDER BY subject_id"
	query += fmt.Sprintf(" LIMIT $%d", argIdx)
	args = append(args, limit)

	db := r.cluster.ReaderFor(tenantID)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get sorted subject IDs: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan subject ID: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// LookupAccessibleEntitiesComplex finds entity IDs that a subject can access
// via direct relations, computed usersets, or hierarchical relations using closure table.
func (r *PostgresRelationRepository) LookupAccessibleEntitiesComplex(ctx context.Context, tenantID string,
	entityType string, relations []string, parentRelations []string,
	subjectType string, subjectID string,
	maxDepth int, cursor string, limit int) ([]string, error) {

	var subQueries []string
	var args []interface{}
	argIdx := 1

	// Helper to add arg and return $N placeholder
	addArg := func(val interface{}) string {
		args = append(args, val)
		placeholder := fmt.Sprintf("$%d", argIdx)
		argIdx++
		return placeholder
	}

	pTenantID := addArg(tenantID)
	pEntityType := addArg(entityType)
	pSubjectType := addArg(subjectType)
	pSubjectID := addArg(subjectID)

	if len(relations) > 0 {
		pRelations := addArg(pq.Array(relations))

		// Sub-query 1: Direct relations
		subQueries = append(subQueries, fmt.Sprintf(`
			SELECT DISTINCT r.entity_id
			FROM relations r
			WHERE r.tenant_id = %s AND r.entity_type = %s
			  AND r.relation = ANY(%s)
			  AND r.subject_type = %s AND r.subject_id = %s
			  AND COALESCE(r.subject_relation, '') = ''
		`, pTenantID, pEntityType, pRelations, pSubjectType, pSubjectID))

		// Sub-query 2: Computed usersets (recursive CTE for nested expansion)
		// Handles chains like: entity#rel@team#member -> team#member@group#member -> group#member@user
		pUsersetDepth := addArg(maxUsersetDepth)
		subQueries = append(subQueries, fmt.Sprintf(`
			SELECT DISTINCT outer_r.entity_id
			FROM relations outer_r
			INNER JOIN LATERAL (
			  WITH RECURSIVE userset_chain AS (
			    SELECT outer_r.subject_type AS cur_type, outer_r.subject_id AS cur_id,
			           outer_r.subject_relation AS cur_rel, 1 AS depth
			    UNION ALL
			    SELECT nr.subject_type, nr.subject_id, nr.subject_relation, uc.depth + 1
			    FROM userset_chain uc
			    INNER JOIN relations nr
			      ON nr.tenant_id = %s
			      AND nr.entity_type = uc.cur_type
			      AND nr.entity_id = uc.cur_id
			      AND nr.relation = uc.cur_rel
			      AND nr.subject_relation IS NOT NULL AND nr.subject_relation != ''
			    WHERE uc.depth < %s
			  )
			  SELECT 1 FROM userset_chain uc2
			  INNER JOIN relations leaf
			    ON leaf.tenant_id = %s
			    AND leaf.entity_type = uc2.cur_type
			    AND leaf.entity_id = uc2.cur_id
			    AND leaf.relation = uc2.cur_rel
			    AND leaf.subject_type = %s AND leaf.subject_id = %s
			    AND COALESCE(leaf.subject_relation, '') = ''
			  LIMIT 1
			) userset_match ON true
			WHERE outer_r.tenant_id = %s AND outer_r.entity_type = %s
			  AND outer_r.relation = ANY(%s)
			  AND outer_r.subject_relation IS NOT NULL AND outer_r.subject_relation != ''
		`, pTenantID, pUsersetDepth, pTenantID, pSubjectType, pSubjectID, pTenantID, pEntityType, pRelations))
	}

	if len(parentRelations) > 0 {
		// Parse parentRelations to extract hierarchy relations and target relations.
		// e.g., ["parent.owner", "parent.editor"] -> hierarchyRelations = ["parent"], targetRelations = ["owner", "editor"]
		// The hierarchy relation name is used to filter closure paths to only those
		// established through the correct relation (e.g., "parent" not "reference").
		type hierPair struct{ hierRel, targetRel string }
		var pairs []hierPair
		hierRelSet := make(map[string]bool)
		targetRelSet := make(map[string]bool)
		for _, pr := range parentRelations {
			parts := strings.SplitN(pr, ".", 2)
			if len(parts) == 2 {
				pairs = append(pairs, hierPair{parts[0], parts[1]})
				hierRelSet[parts[0]] = true
				targetRelSet[parts[1]] = true
			}
		}

		hierRelations := make([]string, 0, len(hierRelSet))
		for r := range hierRelSet {
			hierRelations = append(hierRelations, r)
		}
		targetRelations := make([]string, 0, len(targetRelSet))
		for r := range targetRelSet {
			targetRelations = append(targetRelations, r)
		}

		if len(targetRelations) > 0 {
			pHierRelations := addArg(pq.Array(hierRelations))
			pTargetRelations := addArg(pq.Array(targetRelations))
			pMaxDepth := addArg(maxDepth)

			// Sub-query 3: Hierarchical direct
			// Uses a recursive CTE on the relations table filtered by the hierarchy
			// relation name (e.g., "parent") instead of the unfiltered closure table,
			// ensuring only ancestors reachable through the correct relation are found.
			subQueries = append(subQueries, fmt.Sprintf(`
				SELECT DISTINCT hier.ancestor_id AS entity_id
				FROM (
				  WITH RECURSIVE hier_walk AS (
				    SELECT entity_id AS descendant_id, subject_type AS ancestor_type, subject_id AS ancestor_id, 1 AS depth
				    FROM relations
				    WHERE tenant_id = %s AND entity_type = %s
				      AND relation = ANY(%s)
				      AND COALESCE(subject_relation, '') = ''
				    UNION ALL
				    SELECT hw.descendant_id, r.subject_type, r.subject_id, hw.depth + 1
				    FROM hier_walk hw
				    INNER JOIN relations r
				      ON r.tenant_id = %s
				      AND r.entity_type = hw.ancestor_type
				      AND r.entity_id = hw.ancestor_id
				      AND r.relation = ANY(%s)
				      AND COALESCE(r.subject_relation, '') = ''
				    WHERE hw.depth < %s
				  )
				  SELECT descendant_id, ancestor_type, ancestor_id FROM hier_walk
				) hier
				INNER JOIN relations r
				  ON r.entity_type = hier.ancestor_type
				  AND r.entity_id = hier.ancestor_id
				  AND r.tenant_id = %s
				  AND r.relation = ANY(%s)
				  AND r.subject_type = %s AND r.subject_id = %s
				  AND COALESCE(r.subject_relation, '') = ''
			`, pTenantID, pEntityType, pHierRelations,
				pTenantID, pHierRelations, pMaxDepth,
				pTenantID, pTargetRelations, pSubjectType, pSubjectID))

			// Sub-query 4: Hierarchical computed usersets
			pHierUsersetDepth := addArg(maxUsersetDepth)
			subQueries = append(subQueries, fmt.Sprintf(`
				SELECT DISTINCT hier.descendant_id AS entity_id
				FROM (
				  WITH RECURSIVE hier_walk AS (
				    SELECT entity_id AS descendant_id, subject_type AS ancestor_type, subject_id AS ancestor_id, 1 AS depth
				    FROM relations
				    WHERE tenant_id = %s AND entity_type = %s
				      AND relation = ANY(%s)
				      AND COALESCE(subject_relation, '') = ''
				    UNION ALL
				    SELECT hw.descendant_id, r.subject_type, r.subject_id, hw.depth + 1
				    FROM hier_walk hw
				    INNER JOIN relations r
				      ON r.tenant_id = %s
				      AND r.entity_type = hw.ancestor_type
				      AND r.entity_id = hw.ancestor_id
				      AND r.relation = ANY(%s)
				      AND COALESCE(r.subject_relation, '') = ''
				    WHERE hw.depth < %s
				  )
				  SELECT descendant_id, ancestor_type, ancestor_id FROM hier_walk
				) hier
				INNER JOIN relations r
				  ON r.entity_type = hier.ancestor_type
				  AND r.entity_id = hier.ancestor_id
				  AND r.tenant_id = %s
				  AND r.relation = ANY(%s)
				  AND r.subject_relation IS NOT NULL AND r.subject_relation != ''
				INNER JOIN LATERAL (
				  WITH RECURSIVE userset_chain AS (
				    SELECT r.subject_type AS cur_type, r.subject_id AS cur_id,
				           r.subject_relation AS cur_rel, 1 AS depth
				    UNION ALL
				    SELECT nr.subject_type, nr.subject_id, nr.subject_relation, uc.depth + 1
				    FROM userset_chain uc
				    INNER JOIN relations nr
				      ON nr.tenant_id = %s
				      AND nr.entity_type = uc.cur_type
				      AND nr.entity_id = uc.cur_id
				      AND nr.relation = uc.cur_rel
				      AND nr.subject_relation IS NOT NULL AND nr.subject_relation != ''
				    WHERE uc.depth < %s
				  )
				  SELECT 1 FROM userset_chain uc2
				  INNER JOIN relations leaf
				    ON leaf.tenant_id = %s
				    AND leaf.entity_type = uc2.cur_type
				    AND leaf.entity_id = uc2.cur_id
				    AND leaf.relation = uc2.cur_rel
				    AND leaf.subject_type = %s AND leaf.subject_id = %s
				    AND COALESCE(leaf.subject_relation, '') = ''
				  LIMIT 1
				) userset_match ON true
			`, pTenantID, pEntityType, pHierRelations,
				pTenantID, pHierRelations, pMaxDepth,
				pTenantID, pTargetRelations,
				pTenantID, pHierUsersetDepth,
				pTenantID, pSubjectType, pSubjectID))
		}
	}

	if len(subQueries) == 0 {
		return nil, nil
	}

	// Build final query with cursor pagination
	combined := strings.Join(subQueries, " UNION ")
	query := fmt.Sprintf(`SELECT entity_id FROM (%s) combined`, combined)

	if cursor != "" {
		pCursor := addArg(cursor)
		query += fmt.Sprintf(` WHERE entity_id > %s`, pCursor)
	}

	query += " ORDER BY entity_id"
	pLimit := addArg(limit)
	query += fmt.Sprintf(` LIMIT %s`, pLimit)

	db := r.cluster.ReaderFor(tenantID)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup accessible entities: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan entity ID: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// LookupAccessibleSubjectsComplex finds subject IDs that can access an entity
// via direct relations, computed usersets, or hierarchical relations using closure table.
func (r *PostgresRelationRepository) LookupAccessibleSubjectsComplex(ctx context.Context, tenantID string,
	entityType string, entityID string, relations []string, parentRelations []string,
	subjectType string,
	maxDepth int, cursor string, limit int) ([]string, error) {

	var subQueries []string
	var args []interface{}
	argIdx := 1

	// Helper to add arg and return $N placeholder
	addArg := func(val interface{}) string {
		args = append(args, val)
		placeholder := fmt.Sprintf("$%d", argIdx)
		argIdx++
		return placeholder
	}

	pTenantID := addArg(tenantID)
	pEntityType := addArg(entityType)
	pEntityID := addArg(entityID)
	pSubjectType := addArg(subjectType)

	if len(relations) > 0 {
		pRelations := addArg(pq.Array(relations))

		// Sub-query 1: Direct relations
		subQueries = append(subQueries, fmt.Sprintf(`
			SELECT DISTINCT r.subject_id
			FROM relations r
			WHERE r.tenant_id = %s AND r.entity_type = %s AND r.entity_id = %s
			  AND r.relation = ANY(%s)
			  AND r.subject_type = %s
			  AND COALESCE(r.subject_relation, '') = ''
		`, pTenantID, pEntityType, pEntityID, pRelations, pSubjectType))

		// Sub-query 2: Computed usersets (recursive CTE for nested expansion)
		pUsersetDepth := addArg(maxUsersetDepth)
		subQueries = append(subQueries, fmt.Sprintf(`
			SELECT DISTINCT leaf.subject_id
			FROM relations outer_r
			INNER JOIN LATERAL (
			  WITH RECURSIVE userset_chain AS (
			    SELECT outer_r.subject_type AS cur_type, outer_r.subject_id AS cur_id,
			           outer_r.subject_relation AS cur_rel, 1 AS depth
			    UNION ALL
			    SELECT nr.subject_type, nr.subject_id, nr.subject_relation, uc.depth + 1
			    FROM userset_chain uc
			    INNER JOIN relations nr
			      ON nr.tenant_id = %s
			      AND nr.entity_type = uc.cur_type
			      AND nr.entity_id = uc.cur_id
			      AND nr.relation = uc.cur_rel
			      AND nr.subject_relation IS NOT NULL AND nr.subject_relation != ''
			    WHERE uc.depth < %s
			  )
			  SELECT leaf_r.subject_id FROM userset_chain uc2
			  INNER JOIN relations leaf_r
			    ON leaf_r.tenant_id = %s
			    AND leaf_r.entity_type = uc2.cur_type
			    AND leaf_r.entity_id = uc2.cur_id
			    AND leaf_r.relation = uc2.cur_rel
			    AND leaf_r.subject_type = %s
			    AND COALESCE(leaf_r.subject_relation, '') = ''
			) leaf ON true
			WHERE outer_r.tenant_id = %s AND outer_r.entity_type = %s AND outer_r.entity_id = %s
			  AND outer_r.relation = ANY(%s)
			  AND outer_r.subject_relation IS NOT NULL AND outer_r.subject_relation != ''
		`, pTenantID, pUsersetDepth, pTenantID, pSubjectType, pTenantID, pEntityType, pEntityID, pRelations))
	}

	if len(parentRelations) > 0 {
		// Parse parentRelations to extract hierarchy relations and target relations.
		hierRelSet := make(map[string]bool)
		targetRelSet := make(map[string]bool)
		for _, pr := range parentRelations {
			parts := strings.SplitN(pr, ".", 2)
			if len(parts) == 2 {
				hierRelSet[parts[0]] = true
				targetRelSet[parts[1]] = true
			}
		}

		hierRelations := make([]string, 0, len(hierRelSet))
		for r := range hierRelSet {
			hierRelations = append(hierRelations, r)
		}
		targetRelations := make([]string, 0, len(targetRelSet))
		for r := range targetRelSet {
			targetRelations = append(targetRelations, r)
		}

		if len(targetRelations) > 0 {
			pHierRelations := addArg(pq.Array(hierRelations))
			pTargetRelations := addArg(pq.Array(targetRelations))
			pMaxDepth := addArg(maxDepth)

			// Sub-query 3: Hierarchical direct (uses recursive CTE filtered by hierarchy relation)
			subQueries = append(subQueries, fmt.Sprintf(`
				SELECT DISTINCT r.subject_id
				FROM (
				  WITH RECURSIVE hier_walk AS (
				    SELECT subject_type AS ancestor_type, subject_id AS ancestor_id, 1 AS depth
				    FROM relations
				    WHERE tenant_id = %s AND entity_type = %s AND entity_id = %s
				      AND relation = ANY(%s)
				      AND COALESCE(subject_relation, '') = ''
				    UNION ALL
				    SELECT rel.subject_type, rel.subject_id, hw.depth + 1
				    FROM hier_walk hw
				    INNER JOIN relations rel
				      ON rel.tenant_id = %s
				      AND rel.entity_type = hw.ancestor_type
				      AND rel.entity_id = hw.ancestor_id
				      AND rel.relation = ANY(%s)
				      AND COALESCE(rel.subject_relation, '') = ''
				    WHERE hw.depth < %s
				  )
				  SELECT ancestor_type, ancestor_id FROM hier_walk
				) hier
				INNER JOIN relations r
				  ON r.entity_type = hier.ancestor_type
				  AND r.entity_id = hier.ancestor_id
				  AND r.tenant_id = %s
				  AND r.relation = ANY(%s)
				  AND r.subject_type = %s
				  AND COALESCE(r.subject_relation, '') = ''
			`, pTenantID, pEntityType, pEntityID, pHierRelations,
				pTenantID, pHierRelations, pMaxDepth,
				pTenantID, pTargetRelations, pSubjectType))

			// Sub-query 4: Hierarchical computed usersets
			pHierUsersetDepth := addArg(maxUsersetDepth)
			subQueries = append(subQueries, fmt.Sprintf(`
				SELECT DISTINCT leaf.subject_id
				FROM (
				  WITH RECURSIVE hier_walk AS (
				    SELECT subject_type AS ancestor_type, subject_id AS ancestor_id, 1 AS depth
				    FROM relations
				    WHERE tenant_id = %s AND entity_type = %s AND entity_id = %s
				      AND relation = ANY(%s)
				      AND COALESCE(subject_relation, '') = ''
				    UNION ALL
				    SELECT rel.subject_type, rel.subject_id, hw.depth + 1
				    FROM hier_walk hw
				    INNER JOIN relations rel
				      ON rel.tenant_id = %s
				      AND rel.entity_type = hw.ancestor_type
				      AND rel.entity_id = hw.ancestor_id
				      AND rel.relation = ANY(%s)
				      AND COALESCE(rel.subject_relation, '') = ''
				    WHERE hw.depth < %s
				  )
				  SELECT ancestor_type, ancestor_id FROM hier_walk
				) hier
				INNER JOIN relations r
				  ON r.entity_type = hier.ancestor_type
				  AND r.entity_id = hier.ancestor_id
				  AND r.tenant_id = %s
				  AND r.relation = ANY(%s)
				  AND r.subject_relation IS NOT NULL AND r.subject_relation != ''
				INNER JOIN LATERAL (
				  WITH RECURSIVE userset_chain AS (
				    SELECT r.subject_type AS cur_type, r.subject_id AS cur_id,
				           r.subject_relation AS cur_rel, 1 AS depth
				    UNION ALL
				    SELECT nr.subject_type, nr.subject_id, nr.subject_relation, uc.depth + 1
				    FROM userset_chain uc
				    INNER JOIN relations nr
				      ON nr.tenant_id = %s
				      AND nr.entity_type = uc.cur_type
				      AND nr.entity_id = uc.cur_id
				      AND nr.relation = uc.cur_rel
				      AND nr.subject_relation IS NOT NULL AND nr.subject_relation != ''
				    WHERE uc.depth < %s
				  )
				  SELECT leaf_r.subject_id FROM userset_chain uc2
				  INNER JOIN relations leaf_r
				    ON leaf_r.tenant_id = %s
				    AND leaf_r.entity_type = uc2.cur_type
				    AND leaf_r.entity_id = uc2.cur_id
				    AND leaf_r.relation = uc2.cur_rel
				    AND leaf_r.subject_type = %s
				    AND COALESCE(leaf_r.subject_relation, '') = ''
				) leaf ON true
			`, pTenantID, pEntityType, pEntityID, pHierRelations,
				pTenantID, pHierRelations, pMaxDepth,
				pTenantID, pTargetRelations,
				pTenantID, pHierUsersetDepth,
				pTenantID, pSubjectType))
		}
	}

	if len(subQueries) == 0 {
		return nil, nil
	}

	// Build final query with cursor pagination
	combined := strings.Join(subQueries, " UNION ")
	query := fmt.Sprintf(`SELECT subject_id FROM (%s) combined`, combined)

	if cursor != "" {
		pCursor := addArg(cursor)
		query += fmt.Sprintf(` WHERE subject_id > %s`, pCursor)
	}

	query += " ORDER BY subject_id"
	pLimit := addArg(limit)
	query += fmt.Sprintf(` LIMIT %s`, pLimit)

	db := r.cluster.ReaderFor(tenantID)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup accessible subjects: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan subject ID: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// scanTuples scans rows into RelationTuple slices.
func scanTuples(rows *sql.Rows) ([]*entities.RelationTuple, error) {
	var tuples []*entities.RelationTuple
	for rows.Next() {
		var tuple entities.RelationTuple
		var subjectRelation sql.NullString

		err := rows.Scan(
			&tuple.EntityType, &tuple.EntityID, &tuple.Relation,
			&tuple.SubjectType, &tuple.SubjectID, &subjectRelation, &tuple.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan relation: %w", err)
		}

		if subjectRelation.Valid {
			tuple.SubjectRelation = subjectRelation.String
		}

		tuples = append(tuples, &tuple)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating relations: %w", err)
	}

	return tuples, nil
}
