package grpcserver_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"log/slog"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"pet-study/internal/entity"
	"pet-study/internal/security"
	"pet-study/internal/service"
	"pet-study/internal/store/jobrepo"
	"pet-study/internal/transport/grpcclient"
	"pet-study/internal/transport/grpcserver"
	"pet-study/internal/transport/pb"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	reflectionpb "google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/grpc/status"
)

const testGRPCServerName = "grpc.pet-study.internal"

func TestRuntimeMTLSAcceptsConfiguredClientAndRejectsPlaintext(t *testing.T) {
	files := writeMTLSFiles(t)
	jobSvc := service.NewJobService(jobrepo.NewMemoryJobRepository())
	job := entity.Job{Status: entity.JobSucceeded}
	if err := jobSvc.Save(context.Background(), &job); err != nil {
		t.Fatalf("Save job: %v", err)
	}

	runtime := startSecureRuntime(t, jobSvc, files, false)

	client, conn, err := grpcclient.NewJobsClientWithConfig(grpcclient.Config{
		Addr: runtime.Addr(),
		TLS: grpcclient.TLSConfig{
			Enable:     true,
			CertFile:   files.clientCert,
			KeyFile:    files.clientKey,
			CAFile:     files.serverCA,
			ServerName: testGRPCServerName,
		},
	})
	if err != nil {
		t.Fatalf("NewJobsClientWithConfig: %v", err)
	}
	defer conn.Close()

	unauthCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	_, err = client.GetJob(unauthCtx, &pb.GetJobRequest{Id: job.ID})
	cancel()
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("mTLS without bearer code=%v err=%v want %v", status.Code(err), err, codes.Unauthenticated)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ctx = grpcAuthorizedContext(ctx, testAdminToken)
	resp, err := client.GetJob(ctx, &pb.GetJobRequest{Id: job.ID})
	if err != nil {
		t.Fatalf("mTLS authenticated GetJob: %v", err)
	}
	if resp.GetId() != job.ID {
		t.Fatalf("job id=%d want %d", resp.GetId(), job.ID)
	}

	plaintextConn, err := grpc.NewClient(
		runtime.Addr(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("NewClient plaintext: %v", err)
	}
	defer plaintextConn.Close()

	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = pb.NewJobsServiceClient(plaintextConn).GetJob(ctx, &pb.GetJobRequest{Id: job.ID})
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("plaintext GetJob code=%v err=%v want %v", status.Code(err), err, codes.Unavailable)
	}
}

func TestRuntimeMTLSValidatesServerName(t *testing.T) {
	files := writeMTLSFiles(t)
	jobSvc := service.NewJobService(jobrepo.NewMemoryJobRepository())
	job := entity.Job{Status: entity.JobSucceeded}
	if err := jobSvc.Save(context.Background(), &job); err != nil {
		t.Fatalf("Save job: %v", err)
	}

	runtime := startSecureRuntime(t, jobSvc, files, false)

	client, conn, err := grpcclient.NewJobsClientWithConfig(grpcclient.Config{
		Addr: runtime.Addr(),
		TLS: grpcclient.TLSConfig{
			Enable:     true,
			CertFile:   files.clientCert,
			KeyFile:    files.clientKey,
			CAFile:     files.serverCA,
			ServerName: "wrong.pet-study.internal",
		},
	})
	if err != nil {
		t.Fatalf("NewJobsClientWithConfig: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = client.GetJob(ctx, &pb.GetJobRequest{Id: job.ID})
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("wrong server name code=%v err=%v want %v", status.Code(err), err, codes.Unavailable)
	}
}

func TestRuntimeReflectionDisabledByDefaultAndExplicitlyEnabledOnLoopback(t *testing.T) {
	files := writeMTLSFiles(t)
	jobSvc := service.NewJobService(jobrepo.NewMemoryJobRepository())

	secureRuntime := startSecureRuntime(t, jobSvc, files, false)
	secureConn := newMTLSConn(t, secureRuntime.Addr(), files, testGRPCServerName)
	defer secureConn.Close()

	if services, err := listReflectionServices(context.Background(), secureConn); err == nil {
		t.Fatalf("reflection default-off returned services=%v, want unavailable reflection service", services)
	}

	devRuntime, err := grpcserver.NewRuntimeWithConfig(grpcserver.Config{
		Addr:              "127.0.0.1:0",
		ReflectionEnabled: true,
		Auth:              grpcRuntimeAuth(),
	}, jobSvc, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewRuntimeWithConfig dev reflection: %v", err)
	}
	if err := devRuntime.Start(nil); err != nil {
		t.Fatalf("Start dev runtime: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = devRuntime.Shutdown(ctx)
	})

	devConn, err := grpc.NewClient(
		devRuntime.Addr(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("NewClient dev reflection: %v", err)
	}
	defer devConn.Close()

	services, err := listReflectionServices(context.Background(), devConn)
	if err != nil {
		t.Fatalf("list reflection services: %v", err)
	}
	if !containsService(services, "pb.JobsService") {
		t.Fatalf("reflection services=%v missing pb.JobsService", services)
	}
}

func TestRuntimeRejectsInsecureProtectedAndMisconfiguredTLS(t *testing.T) {
	jobSvc := service.NewJobService(jobrepo.NewMemoryJobRepository())
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	_, err := grpcserver.NewRuntimeWithConfig(grpcserver.Config{
		Addr: ":0",
		Auth: grpcRuntimeAuth(),
	}, jobSvc, logger)
	if err == nil || !strings.Contains(err.Error(), "plaintext grpc requires") {
		t.Fatalf("plaintext wildcard error=%v want plaintext loopback validation", err)
	}

	_, err = grpcserver.NewRuntimeWithConfig(grpcserver.Config{
		Addr: "127.0.0.1:0",
		Auth: grpcRuntimeAuth(),
		TLS:  grpcserver.TLSConfig{Enable: true},
	}, jobSvc, logger)
	if err == nil || !strings.Contains(err.Error(), "GRPC_TLS_CERT_FILE") {
		t.Fatalf("misconfigured TLS error=%v want GRPC_TLS_CERT_FILE", err)
	}

	_, err = grpcserver.NewRuntimeWithConfig(grpcserver.Config{
		Addr:              ":0",
		ReflectionEnabled: true,
		Auth:              grpcRuntimeAuth(),
		TLS:               grpcserver.TLSConfig{Enable: true},
	}, jobSvc, logger)
	if err == nil || !strings.Contains(err.Error(), "grpc reflection requires") {
		t.Fatalf("reflection non-loopback error=%v want reflection loopback validation", err)
	}

	_, err = grpcserver.NewRuntimeWithConfig(grpcserver.Config{
		Addr: "127.0.0.1:0",
	}, jobSvc, logger)
	if err == nil || !strings.Contains(err.Error(), "grpc auth") {
		t.Fatalf("missing auth error=%v want grpc auth validation", err)
	}
}

func startSecureRuntime(t *testing.T, jobSvc *service.JobService, files mtlsFiles, reflection bool) *grpcserver.Runtime {
	t.Helper()

	runtime, err := grpcserver.NewRuntimeWithConfig(grpcserver.Config{
		Addr:              "127.0.0.1:0",
		ReflectionEnabled: reflection,
		Auth:              grpcRuntimeAuth(),
		TLS: grpcserver.TLSConfig{
			Enable:       true,
			CertFile:     files.serverCert,
			KeyFile:      files.serverKey,
			ClientCAFile: files.clientCA,
		},
	}, jobSvc, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewRuntimeWithConfig secure: %v", err)
	}
	if err := runtime.Start(nil); err != nil {
		t.Fatalf("Start secure runtime: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = runtime.Shutdown(ctx)
	})
	return runtime
}

func grpcRuntimeAuth() grpcserver.AuthConfig {
	return grpcserver.AuthConfig{
		Verifier: grpcTestVerifier{
			tokens: map[string]security.Principal{
				testAdminToken: {UserID: 999, Role: security.RoleAdmin},
			},
		},
	}
}

func newMTLSConn(t *testing.T, addr string, files mtlsFiles, serverName string) *grpc.ClientConn {
	t.Helper()

	_, conn, err := grpcclient.NewJobsClientWithConfig(grpcclient.Config{
		Addr: addr,
		TLS: grpcclient.TLSConfig{
			Enable:     true,
			CertFile:   files.clientCert,
			KeyFile:    files.clientKey,
			CAFile:     files.serverCA,
			ServerName: serverName,
		},
	})
	if err != nil {
		t.Fatalf("NewJobsClientWithConfig: %v", err)
	}
	return conn
}

func listReflectionServices(ctx context.Context, conn *grpc.ClientConn) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	stream, err := reflectionpb.NewServerReflectionClient(conn).ServerReflectionInfo(ctx)
	if err != nil {
		return nil, err
	}
	if err := stream.Send(&reflectionpb.ServerReflectionRequest{
		MessageRequest: &reflectionpb.ServerReflectionRequest_ListServices{ListServices: ""},
	}); err != nil {
		return nil, err
	}

	resp, err := stream.Recv()
	if err != nil {
		return nil, err
	}

	list := resp.GetListServicesResponse()
	if list == nil {
		return nil, nil
	}

	services := make([]string, 0, len(list.Service))
	for _, service := range list.Service {
		services = append(services, service.GetName())
	}
	return services, nil
}

func containsService(services []string, want string) bool {
	for _, service := range services {
		if service == want {
			return true
		}
	}
	return false
}

type mtlsFiles struct {
	serverCA   string
	serverCert string
	serverKey  string
	clientCA   string
	clientCert string
	clientKey  string
}

func writeMTLSFiles(t *testing.T) mtlsFiles {
	t.Helper()

	dir := t.TempDir()
	serverCA, serverCAKey, serverCAPEM := newCertificateAuthority(t, "pet-study test server ca")
	clientCA, clientCAKey, clientCAPEM := newCertificateAuthority(t, "pet-study test client ca")

	serverCertPEM, serverKeyPEM := newLeafCertificate(t, leafCertificate{
		commonName: "pet-study grpc server",
		ca:         serverCA,
		caKey:      serverCAKey,
		usage:      x509.ExtKeyUsageServerAuth,
		dnsNames:   []string{testGRPCServerName, "localhost"},
		ipAddresses: []net.IP{
			net.ParseIP("127.0.0.1"),
		},
	})
	clientCertPEM, clientKeyPEM := newLeafCertificate(t, leafCertificate{
		commonName: "pet-study grpc bridge client",
		ca:         clientCA,
		caKey:      clientCAKey,
		usage:      x509.ExtKeyUsageClientAuth,
	})

	files := mtlsFiles{
		serverCA:   filepath.Join(dir, "server-ca.pem"),
		serverCert: filepath.Join(dir, "server.pem"),
		serverKey:  filepath.Join(dir, "server-key.pem"),
		clientCA:   filepath.Join(dir, "client-ca.pem"),
		clientCert: filepath.Join(dir, "client.pem"),
		clientKey:  filepath.Join(dir, "client-key.pem"),
	}
	writePEMFile(t, files.serverCA, serverCAPEM)
	writePEMFile(t, files.serverCert, serverCertPEM)
	writePEMFile(t, files.serverKey, serverKeyPEM)
	writePEMFile(t, files.clientCA, clientCAPEM)
	writePEMFile(t, files.clientCert, clientCertPEM)
	writePEMFile(t, files.clientKey, clientKeyPEM)
	return files
}

func newCertificateAuthority(t *testing.T, commonName string) (*x509.Certificate, *ecdsa.PrivateKey, []byte) {
	t.Helper()

	key := newPrivateKey(t)
	template := &x509.Certificate{
		SerialNumber:          randomSerial(t),
		Subject:               pkix.Name{CommonName: commonName},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate CA: %v", err)
	}
	return template, key, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

type leafCertificate struct {
	commonName  string
	ca          *x509.Certificate
	caKey       *ecdsa.PrivateKey
	usage       x509.ExtKeyUsage
	dnsNames    []string
	ipAddresses []net.IP
}

func newLeafCertificate(t *testing.T, cfg leafCertificate) ([]byte, []byte) {
	t.Helper()

	key := newPrivateKey(t)
	template := &x509.Certificate{
		SerialNumber: randomSerial(t),
		Subject:      pkix.Name{CommonName: cfg.commonName},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{cfg.usage},
		DNSNames:     cfg.dnsNames,
		IPAddresses:  cfg.ipAddresses,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, cfg.ca, &key.PublicKey, cfg.caKey)
	if err != nil {
		t.Fatalf("CreateCertificate leaf: %v", err)
	}

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("MarshalECPrivateKey: %v", err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
}

func newPrivateKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	return key
}

func randomSerial(t *testing.T) *big.Int {
	t.Helper()

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatalf("rand.Int serial: %v", err)
	}
	return serial
}

func writePEMFile(t *testing.T, path string, data []byte) {
	t.Helper()

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile %s: %v", path, err)
	}
}
