package service

import (
	"context"
	"pet-study/internal/entity"
	"pet-study/internal/outbound/profile"
	"time"
)

type UserProfileService struct {
	userService    *UserService
	client         profile.Client
	profileTimeout time.Duration
}

func NewUserProfileService(
	userService *UserService,
	client profile.Client,
	profileTimeout time.Duration,
) *UserProfileService {
	return &UserProfileService{
		userService:    userService,
		client:         client,
		profileTimeout: profileTimeout,
	}
}

func (up *UserProfileService) GetUserWithProfile(
	ctx context.Context, userID int64, requestID string) (*entity.User, profile.Profile, error) {
	user, err := up.userService.GetByID(ctx, int(userID))
	if err != nil {
		return nil, profile.Profile{}, err
	}
	ctx2, cancel := context.WithTimeout(ctx, up.profileTimeout)
	defer cancel()
	prof, err1 := up.client.FetchProfile(ctx2, userID, requestID)
	if err1 != nil {
		return nil, profile.Profile{}, err1
	}
	return user, prof, nil
}
