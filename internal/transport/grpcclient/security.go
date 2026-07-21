package grpcclient

import (
	"crypto/x509"
	"fmt"
	"os"
)

func certPoolFromFile(name, path string) (*x509.CertPool, error) {
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pemBytes) {
		return nil, fmt.Errorf("%s does not contain a valid PEM certificate", name)
	}
	return pool, nil
}
