package security

type Verifier interface {
	Verify(tokenString string) (Principal, error)
}
