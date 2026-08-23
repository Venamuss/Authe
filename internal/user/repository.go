package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"authe/internal/platform/database"
)

type repository struct {
	db *database.Postgres
}

func NewRepository(db *database.Postgres) *repository {
	return &repository{
		db: db,
	}
}

func (repo *repository) Create(ctx context.Context, user *User) (int, error) {
	query := `INSERT INTO users (username, password_hash, email) VALUES ($1, $2, $3) RETURNING id`
	var id int

	err := repo.db.Pool.QueryRow(ctx, query, user.Username, user.PasswordHash, user.Email).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to exec query: %w", err)
	}

	return id, nil
}

func (repo *repository) Get(ctx context.Context, id int) (*User, error) {
	query := `SELECT id, username,
		password_hash, email, created_at, updated_at, deleted_at FROM users WHERE id = $1 AND deleted_at IS NULL`
	user := &User{}

	err := repo.db.Pool.QueryRow(ctx, query, id).Scan(&user.ID,
		&user.Username, &user.PasswordHash, &user.Email,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, UserNotFound
		}
		return nil, fmt.Errorf("failed to exec query: %w", err)
	}

	return user, nil
}

func (repo *repository) Update(ctx context.Context, id int, user *User) error {
	query := `
        UPDATE users 
        SET 
            username      = COALESCE($1, username),
            password_hash = COALESCE($2, password_hash),
            email         = COALESCE($3, email),
            updated_at    = NOW()
        WHERE id = $4 AND deleted_at IS NULL;
    `

	tag, err := repo.db.Pool.Exec(ctx, query, strToNull(user.Username), strToNull(user.PasswordHash), strToNull(user.Email), id)

	if err != nil {
		return fmt.Errorf("failed to exec update query: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return UserNotFound
	}
	return nil
}

func (repo *repository) Delete(ctx context.Context, id int) error {
	query := `UPDATE users SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`

	tag, err := repo.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to update query: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return UserNotFound
	}
	return nil
}

func (repo *repository) GetByUsername(ctx context.Context, username string) (*User, error) {
	query := `SELECT id, username,
		password_hash, email, created_at, updated_at, deleted_at FROM users WHERE username = $1 AND deleted_at IS NULL`
	user := &User{}

	err := repo.db.Pool.QueryRow(ctx, query, username).Scan(&user.ID,
		&user.Username, &user.PasswordHash, &user.Email,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, UserNotFound
		}
		return nil, fmt.Errorf("failed to exec query: %w", err)
	}

	return user, nil
}

func strToNull(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
