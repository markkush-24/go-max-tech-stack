package userrepo

import (
	"context"
	"pet-study/internal/apperr"
	"pet-study/internal/entity"
	"pet-study/internal/service"
	"strings"
	"sync"
)

var _ service.UserRepository = (*MemoryUserRepository)(nil)

// MemoryUserRepository — in-memory реализация репозитория пользователей.
// Так как здесь нет блокирующего I/O, методы почти не используют ctx;
// это нормально для тестовой/учебной реализации, но не заменяет
// дисциплину cancel/timeout для будущего DB-backed слоя.
type MemoryUserRepository struct {
	users  map[int]*entity.User
	mux    sync.RWMutex
	nextID int
}

func NewMemoryUserRepository() *MemoryUserRepository {
	return &MemoryUserRepository{
		users:  make(map[int]*entity.User),
		nextID: 1,
	}
}

func (r *MemoryUserRepository) Ping(ctx context.Context) error {
	return ctx.Err()
}

func (r *MemoryUserRepository) GetAll(ctx context.Context) ([]*entity.User, error) {
	r.mux.RLock()
	defer r.mux.RUnlock()

	arrUsers := make([]*entity.User, 0, len(r.users))
	for _, u := range r.users {
		cp := *u
		arrUsers = append(arrUsers, &cp)
	}
	return arrUsers, nil
}

func (r *MemoryUserRepository) GetByID(ctx context.Context, id int) (*entity.User, error) {
	r.mux.RLock()
	defer r.mux.RUnlock()

	u, ok := r.users[id]
	if !ok {
		return nil, entity.ErrUserNotFound
	}
	cp := *u
	return &cp, nil
}

func (r *MemoryUserRepository) Save(ctx context.Context, user *entity.User) error {
	r.mux.Lock()
	defer r.mux.Unlock()

	for _, u := range r.users {
		if strings.EqualFold(u.Email, user.Email) {
			return apperr.ErrConflict
		}
	}

	if user.ID == 0 {
		user.ID = r.nextID
		r.nextID++
	} else {
		if _, ok := r.users[user.ID]; ok {
			return entity.ErrUserAlreadyExists
		}
	}

	if user.Version == 0 {
		user.Version = 1
	}

	stored := *user
	r.users[stored.ID] = &stored
	return nil
}

func (r *MemoryUserRepository) Delete(ctx context.Context, id int) error {
	r.mux.Lock()
	defer r.mux.Unlock()

	if _, ok := r.users[id]; !ok {
		return entity.ErrUserNotFound
	}
	delete(r.users, id)
	return nil
}

func (r *MemoryUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	r.mux.RLock()
	defer r.mux.RUnlock()

	for _, u := range r.users {
		if strings.EqualFold(u.Email, email) {
			return true, nil
		}
	}
	return false, nil
}

func (r *MemoryUserRepository) Update(ctx context.Context, id int, u *entity.User) (*entity.User, error) {
	r.mux.Lock()
	defer r.mux.Unlock()

	prev, ok := r.users[id]
	if !ok {
		return nil, entity.ErrUserNotFound
	}

	next := entity.User{
		ID:      id,
		Name:    u.Name,
		Age:     u.Age,
		Email:   u.Email,
		Version: prev.Version + 1,
	}

	stored := next
	r.users[id] = &stored

	out := stored
	return &out, nil
}
