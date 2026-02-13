package profile

import (
	"context"
)

type Client interface {
	FetchProfile(ctx context.Context, userID int64, requestID string) (Profile, error)
}
