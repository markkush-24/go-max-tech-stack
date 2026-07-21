package grpcserver

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/netip"
	"os"
	"strings"

	"google.golang.org/grpc/credentials"
)

func newServerTLSCredentials(cfg TLSConfig) (credentials.TransportCredentials, error) {
	if strings.TrimSpace(cfg.CertFile) == "" {
		return nil, fmt.Errorf("GRPC_TLS_CERT_FILE is required")
	}
	if strings.TrimSpace(cfg.KeyFile) == "" {
		return nil, fmt.Errorf("GRPC_TLS_KEY_FILE is required")
	}
	if strings.TrimSpace(cfg.ClientCAFile) == "" {
		return nil, fmt.Errorf("GRPC_TLS_CLIENT_CA_FILE is required")
	}

	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load gRPC server certificate: %w", err)
	}

	clientCA, err := certPoolFromFile("GRPC_TLS_CLIENT_CA_FILE", cfg.ClientCAFile)
	if err != nil {
		return nil, err
	}

	return credentials.NewTLS(&tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    clientCA,
	}), nil
}

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

func isLoopbackListenAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}

	host = strings.TrimSpace(host)
	if strings.EqualFold(host, "localhost") {
		return true
	}
	if host == "" {
		return false
	}

	ip, err := netip.ParseAddr(host)
	return err == nil && ip.IsLoopback()
}
