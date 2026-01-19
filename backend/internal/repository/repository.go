package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Repository struct {
	db *sqlx.DB
}

func New(dsn string) (*Repository, error) {
	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	return &Repository{db: db}, nil
}

func (r *Repository) Close() error {
	return r.db.Close()
}

func (r *Repository) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

func (r *Repository) DB() *sqlx.DB {
	return r.db
}

func (r *Repository) QueryRow(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	return r.db.QueryRowxContext(ctx, query, args...)
}
