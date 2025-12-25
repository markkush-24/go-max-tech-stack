package service

import (
	"context"
	"pet-study/internal/entity"
)

type UserRepository interface {
	GetAll(ctx context.Context) ([]*entity.User, error)
	GetByID(ctx context.Context, id int) (*entity.User, error)
	Save(ctx context.Context, user *entity.User) error
	Delete(ctx context.Context, id int) error
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}
