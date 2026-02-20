package outbound

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"pet-study/internal/outbound/profile"
	"pet-study/internal/requestid"
)

type ClientImpl struct {
	base *url.URL
	http *http.Client
}

func NewClientImpl(baseURL string, http *http.Client) *ClientImpl {
	u, _ := url.Parse(baseURL)
	return &ClientImpl{base: u, http: http}
}

func (ci *ClientImpl) FetchProfile(ctx context.Context, userID int64, requestID string) (profile.Profile, error) {
	if ci.base == nil {
		return profile.Profile{}, &profile.Error{Kind: profile.ErrBadResponse, Cause: errors.New("base url is nil")}
	}

	urlStr := ci.base.JoinPath("profiles", strconv.FormatInt(userID, 10)).String()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
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
		return profile.Profile{}, &profile.Error{Kind: profile.ErrNetwork, Cause: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
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
		return profile.Profile{}, &profile.Error{Kind: profile.ErrParse, Status: resp.StatusCode, Cause: err}
	}

	return p, nil
}
