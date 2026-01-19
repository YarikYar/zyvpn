package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/zyvpn/backend/internal/model"
)

var ErrUserNotFound = errors.New("user not found")

func (r *Repository) GetUser(ctx context.Context, id int64) (*model.User, error) {
	var user model.User
	err := r.db.GetContext(ctx, &user, "SELECT * FROM users WHERE id = $1", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *Repository) CreateUser(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (id, username, first_name, last_name, language_code, referral_code, referred_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			username = EXCLUDED.username,
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			language_code = EXCLUDED.language_code,
			updated_at = NOW()
		RETURNING created_at, updated_at`

	return r.db.QueryRowContext(ctx, query,
		user.ID,
		user.Username,
		user.FirstName,
		user.LastName,
		user.LanguageCode,
		user.ReferralCode,
		user.ReferredBy,
	).Scan(&user.CreatedAt, &user.UpdatedAt)
}

func (r *Repository) UpdateUser(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users SET
			username = $2,
			first_name = $3,
			last_name = $4,
			language_code = $5,
			updated_at = NOW()
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Username,
		user.FirstName,
		user.LastName,
		user.LanguageCode,
	)
	return err
}

func (r *Repository) GetUserByReferralCode(ctx context.Context, code string) (*model.User, error) {
	var user model.User
	err := r.db.GetContext(ctx, &user, "SELECT * FROM users WHERE referral_code = $1", code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *Repository) GetUserWithSubscription(ctx context.Context, id int64) (*model.UserWithSubscription, error) {
	user, err := r.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}

	result := &model.UserWithSubscription{User: *user}

	sub, err := r.GetActiveSubscription(ctx, id)
	if err != nil && !errors.Is(err, ErrSubscriptionNotFound) {
		return nil, err
	}
	result.Subscription = sub

	return result, nil
}
