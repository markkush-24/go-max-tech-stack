package userrepo

import (
	"context"
	"database/sql"
	"errors"
	"pet-study/internal/apperr"
	"pet-study/internal/db"
	"pet-study/internal/entity"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	qGetAllUsers = `
SELECT id, name, email, age
FROM users
ORDER BY id
`

	qGetUserByID = `
SELECT id, name, email, age
FROM users
WHERE id = $1
`

	qInsertUser = `
INSERT INTO users (name, email, age)
VALUES ($1, $2, $3)
RETURNING id
`

	qDeleteUser = `
DELETE FROM users
WHERE id = $1
`

	qExistsByEmail = `
SELECT EXISTS(
	SELECT 1
	FROM users
	WHERE email = $1
)
`
)

type SQLXUserRepository struct {
	sqlDB        *db.DB
	queryTimeout time.Duration
}

func NewSQLX(sqlDB *db.DB, queryTimeout time.Duration) *SQLXUserRepository {
	return &SQLXUserRepository{
		sqlDB:        sqlDB,
		queryTimeout: queryTimeout,
	}
}

func (r *SQLXUserRepository) GetAll(ctx context.Context) ([]*entity.User, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	rows, err := r.sqlDB.QueryContext(ctx, qGetAllUsers)
	if err != nil {
		return nil, mapUserRepoErr(err)
	}
	defer rows.Close()

	users := make([]*entity.User, 0)

	for rows.Next() {
		var user entity.User
		if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.Age); err != nil {
			return nil, err
		}
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, mapUserRepoErr(err)
	}

	return users, nil
}

func (r *SQLXUserRepository) GetByID(ctx context.Context, id int) (*entity.User, error) {
	var user entity.User
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	err := r.sqlDB.GetContext(ctx, &user, qGetUserByID, id)

	if err != nil {
		return nil, mapUserRepoErr(err)
	}

	return &user, nil
}

func (r *SQLXUserRepository) Save(ctx context.Context, user *entity.User) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	err := r.sqlDB.QueryRowContext(ctx, qInsertUser,
		user.Name, user.Email, user.Age).Scan(&user.ID)
	if err != nil {
		return mapUserRepoErr(err)
	}

	return nil
}

func (r *SQLXUserRepository) Delete(ctx context.Context, id int) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	res, err := r.sqlDB.ExecContext(ctx, qDeleteUser, id)
	if err != nil {
		return mapUserRepoErr(err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return entity.ErrUserNotFound
	}
	return nil
}

func (r *SQLXUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	var exists bool
	if err := r.sqlDB.GetContext(ctx, &exists, qExistsByEmail, email); err != nil {
		return false, mapUserRepoErr(err)
	}

	return exists, nil
}

func mapUserRepoErr(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return entity.ErrUserNotFound
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return apperr.ErrConflict
		}
	}

	return err
}
