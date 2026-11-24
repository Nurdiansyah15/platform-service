package persistence

import (
	"context"
	"database/sql"
	"errors"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/permission/constant"
	"platform-service/internal/domain/permission/entity"
	"platform-service/internal/domain/permission/valueobject"
)

// PostgresPermissionGroupRepository implements repository.PermissionGroupRepository.
type PostgresPermissionGroupRepository struct {
	db *sql.DB
}

func NewPostgresPermissionGroupRepository(db *sql.DB) *PostgresPermissionGroupRepository {
	return &PostgresPermissionGroupRepository{db: db}
}

func (r *PostgresPermissionGroupRepository) Save(ctx context.Context, g *entity.Group) error {
	const q = `
		INSERT INTO permission_groups (id, solution_id, code, name, description, sort_order, status, source, created_at, updated_at, parent_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			parent_id = EXCLUDED.parent_id,
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			sort_order = EXCLUDED.sort_order,
			status = EXCLUDED.status
		RETURNING created_at, updated_at`
	err := execFromContext(ctx, r.db).QueryRowContext(ctx, q,
		g.ID, g.SolutionID, g.Code.Value(), g.Name, g.Description, g.SortOrder, g.Status.String(), g.Source.String(),
		g.CreatedAt, g.UpdatedAt, g.ParentID,
	).Scan(&g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domainerr.New(constant.CodeGroupDuplicate)
		}
		if isForeignKeyViolation(err) {
			return domainerr.New(constant.CodeGroupParentUnknown)
		}
		if isCheckViolation(err) {
			return domainerr.New(constant.CodeGroupParentInvalid)
		}
		return domainerr.Wrap(constant.CodePermissionPersistenceFailed, err)
	}
	return nil
}

const permissionGroupColumns = `id, solution_id, code, name, description, sort_order, status, source, created_at, updated_at, parent_id`

func (r *PostgresPermissionGroupRepository) ListBySolution(ctx context.Context, solutionID string) ([]*entity.Group, error) {
	rows, err := execFromContext(ctx, r.db).QueryContext(ctx,
		`SELECT `+permissionGroupColumns+` FROM permission_groups WHERE solution_id = $1 ORDER BY sort_order, code`, solutionID)
	if err != nil {
		return nil, domainerr.Wrap(constant.CodePermissionQueryFailed, err)
	}
	defer rows.Close()

	var out []*entity.Group
	for rows.Next() {
		g, err := scanGroup(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	if err := rows.Err(); err != nil {
		return nil, domainerr.Wrap(constant.CodePermissionQueryFailed, err)
	}
	return out, nil
}

func (r *PostgresPermissionGroupRepository) FindByID(ctx context.Context, id string) (*entity.Group, error) {
	g, err := scanGroup(execFromContext(ctx, r.db).QueryRowContext(ctx,
		`SELECT `+permissionGroupColumns+` FROM permission_groups WHERE id = $1`, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainerr.New(constant.CodeGroupNotFound)
		}
		return nil, err
	}
	return g, nil
}

func scanGroup(row interface{ Scan(dest ...any) error }) (*entity.Group, error) {
	var (
		g                    entity.Group
		code, status, source string
	)
	if err := row.Scan(&g.ID, &g.SolutionID, &code, &g.Name, &g.Description, &g.SortOrder, &status, &source,
		&g.CreatedAt, &g.UpdatedAt, &g.ParentID); err != nil {
		return nil, domainerr.Wrap(constant.CodePermissionQueryFailed, err)
	}
	var err error
	if g.Code, err = valueobject.NewGroupCode(code); err != nil {
		return nil, domainerr.Wrap(constant.CodePermissionQueryFailed, err)
	}
	g.Status = valueobject.PermissionStatus(status)
	g.Source = valueobject.PermissionSource(source)
	return &g, nil
}

// PostgresScopeTypeRepository implements repository.ScopeTypeRepository.
type PostgresScopeTypeRepository struct {
	db *sql.DB
}

func NewPostgresScopeTypeRepository(db *sql.DB) *PostgresScopeTypeRepository {
	return &PostgresScopeTypeRepository{db: db}
}

const scopeTypeColumns = `id, solution_id, code, name, description, parent_type, sort_order, status, source, created_at, updated_at`

func (r *PostgresScopeTypeRepository) Save(ctx context.Context, s *entity.ScopeType) error {
	const q = `
		INSERT INTO solution_scope_types (` + scopeTypeColumns + `)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			parent_type = EXCLUDED.parent_type,
			sort_order = EXCLUDED.sort_order,
			status = EXCLUDED.status
		RETURNING created_at, updated_at`
	err := execFromContext(ctx, r.db).QueryRowContext(ctx, q,
		s.ID, s.SolutionID, s.Code.Value(), s.Name, s.Description, s.Parent.String(), s.SortOrder, s.Status.String(),
		s.Source.String(), s.CreatedAt, s.UpdatedAt,
	).Scan(&s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domainerr.New(constant.CodeScopeTypeDuplicate)
		}
		if isCheckViolation(err) {
			return domainerr.New(constant.CodeScopeTypeCodeInvalid)
		}
		return domainerr.Wrap(constant.CodePermissionPersistenceFailed, err)
	}
	return nil
}

func (r *PostgresScopeTypeRepository) ListBySolution(ctx context.Context, solutionID string) ([]*entity.ScopeType, error) {
	rows, err := execFromContext(ctx, r.db).QueryContext(ctx,
		`SELECT `+scopeTypeColumns+` FROM solution_scope_types WHERE solution_id = $1 ORDER BY sort_order, code`, solutionID)
	if err != nil {
		return nil, domainerr.Wrap(constant.CodePermissionQueryFailed, err)
	}
	defer rows.Close()

	var out []*entity.ScopeType
	for rows.Next() {
		s, err := scanScopeType(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, domainerr.Wrap(constant.CodePermissionQueryFailed, err)
	}
	return out, nil
}

func (r *PostgresScopeTypeRepository) FindByCode(ctx context.Context, solutionID, code string) (*entity.ScopeType, error) {
	row := execFromContext(ctx, r.db).QueryRowContext(ctx,
		`SELECT `+scopeTypeColumns+` FROM solution_scope_types WHERE solution_id = $1 AND code = $2`, solutionID, code)
	s, err := scanScopeType(row)
	if err != nil {
		var de *domainerr.DomainError
		if errors.As(err, &de) && errors.Is(de.Unwrap(), sql.ErrNoRows) {
			return nil, domainerr.New(constant.CodeScopeTypeNotFound)
		}
		return nil, err
	}
	return s, nil
}

func (r *PostgresScopeTypeRepository) FindByID(ctx context.Context, id string) (*entity.ScopeType, error) {
	s, err := scanScopeType(execFromContext(ctx, r.db).QueryRowContext(ctx,
		`SELECT `+scopeTypeColumns+` FROM solution_scope_types WHERE id = $1`, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainerr.New(constant.CodeScopeTypeNotFound)
		}
		return nil, err
	}
	return s, nil
}

func scanScopeType(row interface{ Scan(dest ...any) error }) (*entity.ScopeType, error) {
	var (
		s                            entity.ScopeType
		code, parent, status, source string
	)
	if err := row.Scan(&s.ID, &s.SolutionID, &code, &s.Name, &s.Description, &parent, &s.SortOrder, &status, &source,
		&s.CreatedAt, &s.UpdatedAt); err != nil {
		return nil, domainerr.Wrap(constant.CodePermissionQueryFailed, err)
	}
	var err error
	if s.Code, err = valueobject.NewScopeTypeCode(code); err != nil {
		return nil, domainerr.Wrap(constant.CodePermissionQueryFailed, err)
	}
	s.Parent = valueobject.ScopeParent(parent)
	s.Status = valueobject.PermissionStatus(status)
	s.Source = valueobject.PermissionSource(source)
	return &s, nil
}
