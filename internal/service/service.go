package service

import (
	"context"
	"fmt"
	"pet-study/internal/entity"
)

type UserService struct {
	repo UserRepository
}

func NewUserService(repo UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) GetAllUsers(ctx context.Context) ([]*entity.User, error) {
	return s.repo.GetAll(ctx)
}

func (s *UserService) GetByID(ctx context.Context, id int) (*entity.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *UserService) Save(ctx context.Context, user *entity.User) error {
	return s.repo.Save(ctx, user)
}

func (s *UserService) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}

func (s *UserService) CreateUser(ctx context.Context, in *entity.CreateUserInput) (*entity.UserDTO, error) {

	exists, err := s.repo.ExistsByEmail(ctx, in.Email)
	if err != nil {
		return nil, fmt.Errorf("check email: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("email already exists: %w", ErrConflict)
	}

	u := entity.User{Name: in.Name, Age: in.Age, Email: in.Email}
	if err := s.repo.Save(ctx, &u); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	return &entity.UserDTO{ID: u.ID, Name: u.Name, Email: u.Email}, nil
}
