package httputils

import (
	"errors"
	"fmt"
	"mime"
	"strings"
)

const (
	MediaTypeJSON      = "application/json"
	MediaTypeProtobuf  = "application/protobuf"   // canonical
	MediaTypeXProtobuf = "application/x-protobuf" // alias
)

var ErrNotAcceptable = errors.New("unsupported Accept header; supported: application/json, application/protobuf")

// AcceptHeader negotiates response Content-Type by the given Accept header value.
// Returns the chosen Content-Type (application/json or application/protobuf) or ErrNotAcceptable.
func AcceptHeader(accept string) (string, error) {
	accept = strings.TrimSpace(accept)
	if accept == "" {
		return MediaTypeJSON, nil
	}

	parts := strings.Split(accept, ",")

	jsonOK := false
	parsedAny := false

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		mediaType, _, err := mime.ParseMediaType(part)
		if err != nil {
			continue
		}
		parsedAny = true
		mediaType = strings.ToLower(mediaType)

		switch mediaType {
		case MediaTypeProtobuf, MediaTypeXProtobuf:
			return MediaTypeProtobuf, nil

		case MediaTypeJSON, "*/*", "application/*":
			jsonOK = true
		}
	}

	if jsonOK {
		return MediaTypeJSON, nil
	}

	if !parsedAny {
		return "", fmt.Errorf("%w: invalid Accept %q", ErrNotAcceptable, accept)
	}
	return "", fmt.Errorf("%w: %q", ErrNotAcceptable, accept)
}
