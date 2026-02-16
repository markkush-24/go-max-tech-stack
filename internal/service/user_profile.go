package service

import (
	"context"
	"pet-study/internal/entity"
	"pet-study/internal/outbound/profile"
)

type ProfileUser interface {
	GetUserWithProfile(ctx context.Context, userID int64, requestID string) (*entity.User, profile.Profile, error)
}
