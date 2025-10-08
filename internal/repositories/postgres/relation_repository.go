package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/asakaida/keruberosu/internal/entities"
	"github.com/asakaida/keruberosu/internal/repositories"
)

// PostgresRelationRepository implements RelationRepository using PostgreSQL
type PostgresRelationRepository struct {
	db *sql.DB
}

// NewPostgresRelationRepository creates a new PostgreSQL relation repository
func NewPostgresRelationRepository(db *sql.DB) repositories.RelationRepository {
	return &PostgresRelationRepository{db: db}
}

// Write creates a new relation tuple
func (r *PostgresRelationRepository) Write(ctx context.Context, tenantID string, tuple *entities.RelationTuple) error {
	if err := tuple.Validate(); err != nil {
		return fmt.Errorf("invalid relation tuple: %w", err)
	}

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
	_, err := r.db.ExecContext(ctx, query,
		tenantID, tuple.EntityType, tuple.EntityID, tuple.Relation,
		tuple.SubjectType, tuple.SubjectID, sql.NullString{String: tuple.SubjectRelation, Valid: tuple.SubjectRelation != ""}, now,
	)
	if err != nil {
		return fmt.Errorf("failed to write relation: %w", err)
	}

	return nil
}

// Delete removes a relation tuple
func (r *PostgresRelationRepository) Delete(ctx context.Context, tenantID string, tuple *entities.RelationTuple) error {
	if err := tuple.Validate(); err != nil {
		return fmt.Errorf("invalid relation tuple: %w", err)
	}

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
	_, err := r.db.ExecContext(ctx, query,
		tenantID, tuple.EntityType, tuple.EntityID, tuple.Relation,
		tuple.SubjectType, tuple.SubjectID, tuple.SubjectRelation,
	)
	if err != nil {
		return fmt.Errorf("failed to delete relation: %w", err)
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

// BatchWrite creates multiple relation tuples in a single transaction
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
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// BatchDelete removes multiple relation tuples in a single transaction
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
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
