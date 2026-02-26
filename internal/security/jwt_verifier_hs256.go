package security

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// HMACKey — ключ для HS256.
// KID используется для выбора ключа при ротации.
type HMACKey struct {
	KID    string
	Secret []byte
}

type jwtClaims struct {
	jwt.RegisteredClaims
	Role string `json:"role"`
}

type JWTVerifierHS256 struct {
	parser     *jwt.Parser
	allowedAlg string

	keys      []HMACKey
	keysByKID map[string][]byte
}

var errNoKID = errors.New("no kid in header")

func NewJWTVerifierHS256(
	allowedAlg string,
	issuer string,
	audience string,
	clockSkew time.Duration,
	keys []HMACKey,
) (*JWTVerifierHS256, error) {
	if allowedAlg != "HS256" {
		return nil, fmt.Errorf("jwt verifier: unsupported alg=%q (expected HS256)", allowedAlg)
	}
	if clockSkew < 0 {
		return nil, fmt.Errorf("jwt verifier: clockSkew must be >= 0")
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("jwt verifier: at least 1 key is required")
	}

	keysByKID := make(map[string][]byte, len(keys))
	for _, k := range keys {
		if k.KID == "" {
			return nil, fmt.Errorf("jwt verifier: kid is empty")
		}
		if len(k.Secret) == 0 {
			return nil, fmt.Errorf("jwt verifier: secret is empty for kid=%q", k.KID)
		}
		if _, ok := keysByKID[k.KID]; ok {
			return nil, fmt.Errorf("jwt verifier: duplicate kid=%q", k.KID)
		}
		keysByKID[k.KID] = k.Secret
	}

	opts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{allowedAlg}), // прибиваем алгоритм
		jwt.WithLeeway(clockSkew),                  // clock skew
		jwt.WithExpirationRequired(),               // exp обязателен
	}

	// Issuer/audience проверяем только если задано в конфиге.
	if issuer != "" {
		opts = append(opts, jwt.WithIssuer(issuer))
	}
	if audience != "" {
		opts = append(opts, jwt.WithAudience(audience))
	}

	return &JWTVerifierHS256{
		parser:     jwt.NewParser(opts...),
		allowedAlg: allowedAlg,
		keys:       append([]HMACKey(nil), keys...),
		keysByKID:  keysByKID,
	}, nil
}

func (v *JWTVerifierHS256) Verify(tokenString string) (Principal, error) {
	// 1) Сначала пытаемся валидировать по kid (если он есть).
	p, ok, err := v.tryVerifyByKID(tokenString)
	if ok {
		return p, nil
	}
	if err != nil && !errors.Is(err, errNoKID) {
		return Principal{}, mapJWTParseError(err)
	}

	// 2) kid отсутствует → пробуем ключи по очереди (rotation without kid).
	var lastErr error
	for _, k := range v.keys {
		p, err := v.tryVerifyWithKey(tokenString, k.Secret)
		if err == nil {
			return p, nil
		}

		lastErr = err

		// Если ошибка только из-за подписи — возможно, ключ не тот → пробуем следующий.
		// Важно: в некоторых случаях ошибка может “содержать” несколько причин
		// (например, signature invalid + expired). Поэтому сигнатуру проверяем первой.
		if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			continue
		}

		// Остальные ошибки (формат, issuer/aud, required claims, nbf/exp при верном ключе)
		// не исправятся сменой ключа → сразу возвращаем.
		return Principal{}, mapJWTParseError(err)
	}

	if lastErr == nil {
		lastErr = jwt.ErrTokenSignatureInvalid
	}
	return Principal{}, &AuthNError{Kind: AuthNInvalid, Cause: lastErr}
}

func (v *JWTVerifierHS256) tryVerifyByKID(tokenString string) (Principal, bool, error) {
	claims := &jwtClaims{}

	keyFunc := func(t *jwt.Token) (any, error) {
		if t.Method == nil || t.Method.Alg() != v.allowedAlg {
			return nil, fmt.Errorf("unexpected alg=%q", safeAlg(t))
		}

		kidRaw, ok := t.Header["kid"]
		if !ok {
			return nil, errNoKID
		}
		kid, ok := kidRaw.(string)
		if !ok || kid == "" {
			return nil, fmt.Errorf("invalid kid header")
		}
		sec, ok := v.keysByKID[kid]
		if !ok {
			return nil, fmt.Errorf("unknown kid=%q", kid)
		}
		return sec, nil
	}

	tok, err := v.parser.ParseWithClaims(tokenString, claims, keyFunc)
	if err != nil {
		return Principal{}, false, err
	}
	if tok == nil || !tok.Valid {
		return Principal{}, false, errors.New("token is not valid")
	}

	p, err := principalFromClaims(claims)
	if err != nil {
		return Principal{}, false, err
	}
	return p, true, nil
}

func (v *JWTVerifierHS256) tryVerifyWithKey(tokenString string, secret []byte) (Principal, error) {
	claims := &jwtClaims{}

	keyFunc := func(t *jwt.Token) (any, error) {
		if t.Method == nil || t.Method.Alg() != v.allowedAlg {
			return nil, fmt.Errorf("unexpected alg=%q", safeAlg(t))
		}
		return secret, nil
	}

	tok, err := v.parser.ParseWithClaims(tokenString, claims, keyFunc)
	if err != nil {
		return Principal{}, err
	}
	if tok == nil || !tok.Valid {
		return Principal{}, errors.New("token is not valid")
	}

	return principalFromClaims(claims)
}

func principalFromClaims(claims *jwtClaims) (Principal, error) {
	sub := claims.Subject
	if sub == "" {
		return Principal{}, &AuthNError{Kind: AuthNInvalid, Cause: errors.New("missing sub")}
	}
	uid, err := strconv.ParseInt(sub, 10, 64)
	if err != nil || uid <= 0 {
		return Principal{}, &AuthNError{Kind: AuthNInvalid, Cause: fmt.Errorf("invalid sub=%q", sub)}
	}

	role := RoleUser
	if claims.Role != "" {
		switch Role(claims.Role) {
		case RoleUser, RoleAdmin:
			role = Role(claims.Role)
		default:
			return Principal{}, &AuthNError{Kind: AuthNInvalid, Cause: fmt.Errorf("invalid role=%q", claims.Role)}
		}
	}

	return Principal{UserID: uid, Role: role}, nil
}

func mapJWTParseError(err error) error {
	// Важно: проверяем signature invalid раньше expired/not-yet,
	// потому что jwt может возвращать “составные” ошибки. :contentReference[oaicite:2]{index=2}
	switch {
	case errors.Is(err, jwt.ErrTokenSignatureInvalid):
		return &AuthNError{Kind: AuthNInvalid, Cause: err}
	case errors.Is(err, jwt.ErrTokenExpired):
		return &AuthNError{Kind: AuthNExpired, Cause: err}
	case errors.Is(err, jwt.ErrTokenNotValidYet):
		return &AuthNError{Kind: AuthNNotYetValid, Cause: err}
	default:
		return &AuthNError{Kind: AuthNInvalid, Cause: err}
	}
}

func safeAlg(t *jwt.Token) string {
	if t == nil || t.Method == nil {
		return ""
	}
	return t.Method.Alg()
}
