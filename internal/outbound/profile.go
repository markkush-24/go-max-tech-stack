package outbound

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"pet-study/internal/outbound/profile"
	"pet-study/internal/requestid"
)

type ClientImpl struct {
	baseURL string
	http    *http.Client
}

func NewClientImpl(baseURL string, http *http.Client) *ClientImpl {
	return &ClientImpl{baseURL: baseURL, http: http}
}

func (ci *ClientImpl) FetchProfile(ctx context.Context, userID int64, requestID string) (profile.Profile, error) {

	url := ci.baseURL + fmt.Sprintf("/profiles/%d", userID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return profile.Profile{}, &profile.Error{Kind: profile.ErrBadResponse, Cause: err}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set(requestid.HeaderName, requestID)

	resp, err := ci.http.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return profile.Profile{}, &profile.Error{Kind: profile.ErrTimeout, Cause: err}
		}
		if errors.Is(err, context.Canceled) {
			return profile.Profile{}, &profile.Error{Kind: profile.ErrCanceled, Cause: err}
		}
		return profile.Profile{}, &profile.Error{Kind: profile.ErrUpstream5xx, Cause: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		_, _ = io.Copy(io.Discard, resp.Body)
		code := resp.StatusCode

		if code >= 500 {
			return profile.Profile{}, &profile.Error{Kind: profile.ErrUpstream5xx, Status: code}
		}
		if code >= 400 && code <= 499 {
			return profile.Profile{}, &profile.Error{Kind: profile.ErrUpstream4xx, Status: code}
		}
		return profile.Profile{}, &profile.Error{Kind: profile.ErrBadResponse, Status: code}
	}

	var p profile.Profile

	dec := json.NewDecoder(resp.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&p); err != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return profile.Profile{}, &profile.Error{
			Kind:   profile.ErrBadResponse,
			Status: resp.StatusCode,
			Cause:  err,
		}
	}

	return p, nil
}
