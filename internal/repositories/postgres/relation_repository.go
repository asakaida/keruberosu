package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/asakaida/keruberosu/internal/entities"
	"github.com/asakaida/keruberosu/internal/repositories"
	"github.com/lib/pq"
)

// PostgresRelationRepository implements RelationRepository using PostgreSQL
type PostgresRelationRepository struct {
	db *sql.DB
}

// NewPostgresRelationRepository creates a new PostgreSQL relation repository
func NewPostgresRelationRepository(db *sql.DB) repositories.RelationRepository {
	return &PostgresRelationRepository{db: db}
}

// Write creates a new relation tuple and updates the closure table.
func (r *PostgresRelationRepository) Write(ctx context.Context, tenantID string, tuple *entities.RelationTuple) error {
	if err := tuple.Validate(); err != nil {
		return fmt.Errorf("invalid relation tuple: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
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

	// Update closure table for O(1) ancestor lookups
	if err := r.updateClosureOnAdd(ctx, tx, tenantID, tuple.EntityType, tuple.EntityID, tuple.SubjectType, tuple.SubjectID); err != nil {
		// Log warning but don't fail - closure table is an optimization
		// The closure table might not exist in older database schemas
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Delete removes a relation tuple and updates the closure table.
func (r *PostgresRelationRepository) Delete(ctx context.Context, tenantID string, tuple *entities.RelationTuple) error {
	if err := tuple.Validate(); err != nil {
		return fmt.Errorf("invalid relation tuple: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
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

	// Update closure table
	if err := r.updateClosureOnDelete(ctx, tx, tenantID, tuple.EntityType, tuple.EntityID, tuple.SubjectType, tuple.SubjectID); err != nil {
		// Log warning but don't fail - closure table is an optimization
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

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

	// Build dynamic WHERE clause based on filter
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

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to read relations: %w", err)
	}
	defer rows.Close()

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

// CheckExists checks if a specific relation tuple exists
func (r *PostgresRelationRepository) CheckExists(ctx context.Context, tenantID string, tuple *entities.RelationTuple) (bool, error) {
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
	var exists bool
	err := r.db.QueryRowContext(ctx, query,
		tenantID, tuple.EntityType, tuple.EntityID, tuple.Relation,
		tuple.SubjectType, tuple.SubjectID, tuple.SubjectRelation,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check relation existence: %w", err)
	}

	return exists, nil
}

// BatchWrite creates multiple relation tuples in a single transaction and updates the closure table.
func (r *PostgresRelationRepository) BatchWrite(ctx context.Context, tenantID string, tuples []*entities.RelationTuple) error {
	if len(tuples) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
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

		// Update closure table for each tuple
		if err := r.updateClosureOnAdd(ctx, tx, tenantID, tuple.EntityType, tuple.EntityID, tuple.SubjectType, tuple.SubjectID); err != nil {
			// Log warning but don't fail - closure table is an optimization
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// BatchDelete removes multiple relation tuples in a single transaction and updates the closure table.
func (r *PostgresRelationRepository) BatchDelete(ctx context.Context, tenantID string, tuples []*entities.RelationTuple) error {
	if len(tuples) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
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

		// Update closure table for each tuple
		if err := r.updateClosureOnDelete(ctx, tx, tenantID, tuple.EntityType, tuple.EntityID, tuple.SubjectType, tuple.SubjectID); err != nil {
			// Log warning but don't fail - closure table is an optimization
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// DeleteByFilter removes relation tuples matching the filter (Permify互換)
func (r *PostgresRelationRepository) DeleteByFilter(ctx context.Context, tenantID string, filter *repositories.RelationFilter) error {
	if filter == nil {
		return fmt.Errorf("filter is required")
	}

	query := `DELETE FROM relations WHERE tenant_id = $1`
	args := []interface{}{tenantID}
	argIdx := 2

	// Build dynamic WHERE clause based on filter
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

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete relations by filter: %w", err)
	}

	return nil
}

// ReadByFilter retrieves relation tuples matching filter with pagination (Permify互換)
func (r *PostgresRelationRepository) ReadByFilter(ctx context.Context, tenantID string, filter *repositories.RelationFilter, pageSize int, pageToken string) ([]*entities.RelationTuple, string, error) {
	if pageSize <= 0 {
		pageSize = 100
	}

	query := `
		SELECT entity_type, entity_id, relation, subject_type, subject_id, subject_relation, created_at
		FROM relations
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}
	argIdx := 2

	// Build dynamic WHERE clause based on filter
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

	// Handle pagination token (use created_at for cursor-based pagination)
	if pageToken != "" {
		query += fmt.Sprintf(" AND created_at > $%d", argIdx)
		args = append(args, pageToken)
		argIdx++
	}

	// Order by created_at for consistent pagination
	query += " ORDER BY created_at"

	// Fetch one extra to determine if there's a next page
	query += fmt.Sprintf(" LIMIT $%d", argIdx)
	args = append(args, pageSize+1)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read relations by filter: %w", err)
	}
	defer rows.Close()

	var tuples []*entities.RelationTuple
	var lastCreatedAt time.Time

	for rows.Next() {
		var tuple entities.RelationTuple
		var subjectRelation sql.NullString

		err := rows.Scan(
			&tuple.EntityType, &tuple.EntityID, &tuple.Relation,
			&tuple.SubjectType, &tuple.SubjectID, &subjectRelation, &tuple.CreatedAt,
		)
		if err != nil {
			return nil, "", fmt.Errorf("failed to scan relation: %w", err)
		}

		if subjectRelation.Valid {
			tuple.SubjectRelation = subjectRelation.String
		}

		tuples = append(tuples, &tuple)
		lastCreatedAt = tuple.CreatedAt
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("error iterating relations: %w", err)
	}

	// Determine next token
	var nextToken string
	if len(tuples) > pageSize {
		tuples = tuples[:pageSize]
		nextToken = lastCreatedAt.Format(time.RFC3339Nano)
	}

	return tuples, nextToken, nil
}

// updateClosureOnAdd updates the entity_closure table when a new relation is added.
// This inserts the direct relationship and all transitive relationships.
func (r *PostgresRelationRepository) updateClosureOnAdd(
	ctx context.Context,
	tx *sql.Tx,
	tenantID, entityType, entityID, subjectType, subjectID string,
) error {
	// Insert direct relationship (depth=1)
	directQuery := `
		INSERT INTO entity_closure (tenant_id, descendant_type, descendant_id, ancestor_type, ancestor_id, depth)
		VALUES ($1, $2, $3, $4, $5, 1)
		ON CONFLICT DO NOTHING
	`
	_, err := tx.ExecContext(ctx, directQuery, tenantID, entityType, entityID, subjectType, subjectID)
	if err != nil {
		return fmt.Errorf("failed to insert direct closure: %w", err)
	}

	// Insert transitive relationships:
	// All ancestors of the subject become ancestors of the entity
	transitiveQuery := `
		INSERT INTO entity_closure (tenant_id, descendant_type, descendant_id, ancestor_type, ancestor_id, depth)
		SELECT $1, $2, $3, ancestor_type, ancestor_id, depth + 1
		FROM entity_closure
		WHERE tenant_id = $1 AND descendant_type = $4 AND descendant_id = $5
		ON CONFLICT DO NOTHING
	`
	_, err = tx.ExecContext(ctx, transitiveQuery, tenantID, entityType, entityID, subjectType, subjectID)
	if err != nil {
		return fmt.Errorf("failed to insert transitive closures: %w", err)
	}

	return nil
}

// updateClosureOnDelete updates the entity_closure table when a relation is deleted.
// This removes the direct relationship between entity and subject.
func (r *PostgresRelationRepository) updateClosureOnDelete(
	ctx context.Context,
	tx *sql.Tx,
	tenantID, entityType, entityID, subjectType, subjectID string,
) error {
	// Delete the direct relationship
	query := `
		DELETE FROM entity_closure
		WHERE tenant_id = $1
		  AND descendant_type = $2 AND descendant_id = $3
		  AND ancestor_type = $4 AND ancestor_id = $5
		  AND depth = 1
	`
	_, err := tx.ExecContext(ctx, query, tenantID, entityType, entityID, subjectType, subjectID)
	if err != nil {
		return fmt.Errorf("failed to delete closure: %w", err)
	}

	return nil
}

// WriteWithClosure creates a new relation tuple and updates the closure table.
// Returns a snapshot token for cache consistency.
func (r *PostgresRelationRepository) WriteWithClosure(ctx context.Context, tenantID string, tuple *entities.RelationTuple) (string, error) {
	if err := tuple.Validate(); err != nil {
		return "", fmt.Errorf("invalid relation tuple: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert the relation
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

	// Update closure table
	err = r.updateClosureOnAdd(ctx, tx, tenantID, tuple.EntityType, tuple.EntityID, tuple.SubjectType, tuple.SubjectID)
	if err != nil {
		return "", fmt.Errorf("failed to update closure: %w", err)
	}

	// Generate snapshot token
	snapshotMgr := NewSnapshotManager(r.db)
	token, err := snapshotMgr.GenerateWriteToken(ctx, tx)
	if err != nil {
		return "", fmt.Errorf("failed to generate snapshot token: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	return token, nil
}

// DeleteWithClosure removes a relation tuple and updates the closure table.
// Returns a snapshot token for cache consistency.
func (r *PostgresRelationRepository) DeleteWithClosure(ctx context.Context, tenantID string, tuple *entities.RelationTuple) (string, error) {
	if err := tuple.Validate(); err != nil {
		return "", fmt.Errorf("invalid relation tuple: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete the relation
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

	// Update closure table
	err = r.updateClosureOnDelete(ctx, tx, tenantID, tuple.EntityType, tuple.EntityID, tuple.SubjectType, tuple.SubjectID)
	if err != nil {
		return "", fmt.Errorf("failed to update closure: %w", err)
	}

	// Generate snapshot token
	snapshotMgr := NewSnapshotManager(r.db)
	token, err := snapshotMgr.GenerateWriteToken(ctx, tx)
	if err != nil {
		return "", fmt.Errorf("failed to generate snapshot token: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	return token, nil
}

// LookupAncestors finds all ancestors of an entity using the closure table.
// This provides O(1) ancestor lookups instead of recursive CTE traversal.
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

	rows, err := r.db.QueryContext(ctx, query, args...)
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
