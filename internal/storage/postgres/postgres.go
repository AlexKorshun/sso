package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/AlexKorshun/sso/domain/models"
	"github.com/AlexKorshun/sso/internal/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	uniqueViolation = "23505"
)

type Storage struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*Storage, error) {

	const op = "storage.postgres.New"

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{pool: pool}, nil
}

func (s *Storage) SaveUser(ctx context.Context, email string, passHash []byte) (int64, error) {
	const queryCreate = `INSERT INTO users (email, pass_hash) VALUES ($1, $2) RETURNING id`
	const op = "storage.postgres.SaveUser"

	var id int64

	row := s.pool.QueryRow(ctx, queryCreate, email, passHash)
	err := row.Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrUserExist)
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil

}

func (s *Storage) User(ctx context.Context, email string) (models.User, error) {
	const queryUser = `SELECT id, email, pass_hash FROM users WHERE email = $1`
	const op = "storage.postgres.User"

	user := models.User{}

	row := s.pool.QueryRow(ctx, queryUser, email)
	err := row.Scan(&user.ID, &user.Email, &user.PassHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
	}
	if err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}
	return user, nil

}

func (s *Storage) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	const queryIsAdmin = `SELECT is_admin FROM users WHERE id = $1`
	const op = "storage.postgres.IsAdmin"

	var isAdmin bool

	row := s.pool.QueryRow(ctx, queryIsAdmin, userID)
	err := row.Scan(&isAdmin)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
	}
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	return isAdmin, nil
}

func (s *Storage) App(ctx context.Context, appID int) (models.App, error) {
	const queryApp = `SELECT name, secret FROM apps WHERE id = $1`
	const op = "storage.postgres.App"

	app := models.App{ID: appID}

	row := s.pool.QueryRow(ctx, queryApp, appID)
	err := row.Scan(&app.Name, &app.Secret)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.App{}, fmt.Errorf("%s: %w", op, storage.ErrAppNotFound)
	}
	if err != nil {
		return models.App{}, fmt.Errorf("%s: %w", op, err)
	}
	return app, nil
}

func (s *Storage) Close() {
	s.pool.Close()
}
