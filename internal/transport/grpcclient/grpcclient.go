package grpcclient

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/netip"
	"pet-study/internal/interceptors"
	"pet-study/internal/transport/pb"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const DefaultCallTimeout = 2 * time.Second

type Config struct {
	Addr        string
	CallTimeout time.Duration
	TLS         TLSConfig
}

type TLSConfig struct {
	Enable     bool
	CertFile   string
	KeyFile    string
	CAFile     string
	ServerName string
}

func NewJobsClient(listenAddr string) (pb.JobsServiceClient, *grpc.ClientConn, error) {
	return NewJobsClientWithConfig(Config{Addr: listenAddr})
}

func NewJobsClientWithConfig(cfg Config) (pb.JobsServiceClient, *grpc.ClientConn, error) {
	target, err := loopbackTarget(cfg.Addr)
	if err != nil {
		return nil, nil, err
	}

	transportCredentials := insecure.NewCredentials()
	if cfg.TLS.Enable {
		transportCredentials, err = newClientTLSCredentials(cfg.TLS)
		if err != nil {
			return nil, nil, err
		}
	} else if !isLoopbackTarget(target) {
		return nil, nil, fmt.Errorf("plaintext grpc client requires a loopback target")
	}

	callTimeout := cfg.CallTimeout
	if callTimeout == 0 {
		callTimeout = DefaultCallTimeout
	}
	if callTimeout < 0 {
		return nil, nil, fmt.Errorf("grpc client call timeout must be >= 0")
	}

	conn, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(transportCredentials),
		grpc.WithChainUnaryInterceptor(
			interceptors.UnaryClientRequestIDAndTimeout(callTimeout),
		),
	)
	if err != nil {
		return nil, nil, err
	}

	return pb.NewJobsServiceClient(conn), conn, nil
}

func newClientTLSCredentials(cfg TLSConfig) (credentials.TransportCredentials, error) {
	if strings.TrimSpace(cfg.CertFile) == "" {
		return nil, fmt.Errorf("GRPC_TLS_CLIENT_CERT_FILE is required")
	}
	if strings.TrimSpace(cfg.KeyFile) == "" {
		return nil, fmt.Errorf("GRPC_TLS_CLIENT_KEY_FILE is required")
	}
	if strings.TrimSpace(cfg.CAFile) == "" {
		return nil, fmt.Errorf("GRPC_TLS_SERVER_CA_FILE is required")
	}
	if strings.TrimSpace(cfg.ServerName) == "" {
		return nil, fmt.Errorf("GRPC_TLS_SERVER_NAME is required")
	}

	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load gRPC client certificate: %w", err)
	}

	serverCA, err := certPoolFromFile("GRPC_TLS_SERVER_CA_FILE", cfg.CAFile)
	if err != nil {
		return nil, err
	}

	return credentials.NewTLS(&tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
		RootCAs:      serverCA,
		ServerName:   cfg.ServerName,
	}), nil
}

func loopbackTarget(addr string) (string, error) {
	if strings.HasPrefix(addr, ":") {
		return "127.0.0.1" + addr, nil
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", err
	}

	switch host {
	case "", "0.0.0.0", "::":
		host = "127.0.0.1"
	}

	return net.JoinHostPort(host, port), nil
}

func isLoopbackTarget(target string) bool {
	host, _, err := net.SplitHostPort(target)
	if err != nil {
		return false
	}

	host = strings.TrimSpace(host)
	if strings.EqualFold(host, "localhost") {
		return true
	}

	ip, err := netip.ParseAddr(host)
	return err == nil && ip.IsLoopback()
}
