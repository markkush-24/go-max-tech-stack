package service

import (
	"errors"
	"pet-study/internal/entity"
)

var (
	ErrNotFound  = entity.ErrUserNotFound
	ErrConflict  = errors.New("conflict")
	ErrForbidden = errors.New("forbidden")
)
