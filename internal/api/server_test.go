package api_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"pet-study/internal/api"
	"pet-study/internal/config"
	"pet-study/internal/entity"
	"pet-study/internal/health"
	"pet-study/internal/metrics"
	"pet-study/internal/queue"
	"pet-study/internal/service"
	"pet-study/internal/store/jobrepo"
	"pet-study/internal/store/userrepo"
	"pet-study/internal/stream"
	"pet-study/internal/workerpool"
	"testing"
	"time"
)

func TestRunDoesNotSetReadyOnBindFailure(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	jobSvc := service.NewJobService(jobrepo.NewMemoryJobRepository())
	userSvc := service.NewUserService(userrepo.NewMemoryUserRepository())
	q := queue.New(1)

	hub := stream.NewHub(16)
	pool := workerpool.NewWorkerPool(q, jobSvc, userSvc, metrics.DefaultHTTP(), hub)
	if err := pool.Start(context.Background(), 0); err != nil {
		t.Fatalf("start worker pool: %v", err)
	}
	defer stopPool(t, pool)

	readiness := health.NewReadiness()
	server := api.NewAPIServer(testConfig(ln.Addr().String()), http.NewServeMux(), readiness, pool, q)

	if err := server.Run(context.Background()); err == nil {
		t.Fatal("Run() error = nil, want bind failure")
	}
	if readiness.IsReady() {
		t.Fatal("readiness=true after bind failure")
	}
}

func TestRunFailsQueuedJobsOnShutdown(t *testing.T) {
	jobRepo := jobrepo.NewMemoryJobRepository()
	jobSvc := service.NewJobService(jobRepo)
	userSvc := service.NewUserService(userrepo.NewMemoryUserRepository())
	q := queue.New(1)
	hub := stream.NewHub(16)
	pool := workerpool.NewWorkerPool(q, jobSvc, userSvc, metrics.DefaultHTTP(), hub)
	if err := pool.Start(context.Background(), 0); err != nil {
		t.Fatalf("start worker pool: %v", err)
	}

	job := entity.Job{Status: entity.JobQueued}
	if err := jobSvc.Save(context.Background(), &job); err != nil {
		t.Fatalf("save job: %v", err)
	}
	if err := q.Enqueue(context.Background(), queue.WorkItem{
		JobID: job.ID,
		Payload: entity.CreateUserInput{
			Name:  "queued-user",
			Email: "queued@example.com",
			Age:   21,
		},
	}); err != nil {
		t.Fatalf("enqueue job: %v", err)
	}

	readiness := health.NewReadiness()
	server := api.NewAPIServer(testConfig("127.0.0.1:0"), http.NewServeMux(), readiness, pool, q)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Run(ctx)
	}()

	waitReady(t, readiness)
	cancel()

	if err := <-errCh; err != nil {
		t.Fatalf("Run(): %v", err)
	}

	assertJobFailedOnShutdown(t, jobSvc, job.ID)
}

func TestRunFailsRunningJobsOnShutdown(t *testing.T) {
	jobRepo := jobrepo.NewMemoryJobRepository()
	jobSvc := service.NewJobService(jobRepo)
	userSvc := service.NewUserService(&blockingUserRepo{
		saveStarted: make(chan struct{}),
	})
	q := queue.New(1)
	hub := stream.NewHub(16)
	pool := workerpool.NewWorkerPool(q, jobSvc, userSvc, metrics.DefaultHTTP(), hub)
	if err := pool.Start(context.Background(), 1); err != nil {
		t.Fatalf("start worker pool: %v", err)
	}

	job := entity.Job{Status: entity.JobQueued}
	if err := jobSvc.Save(context.Background(), &job); err != nil {
		t.Fatalf("save job: %v", err)
	}
	if err := q.Enqueue(context.Background(), queue.WorkItem{
		JobID: job.ID,
		Payload: entity.CreateUserInput{
			Name:  "running-user",
			Email: "running@example.com",
			Age:   21,
		},
	}); err != nil {
		t.Fatalf("enqueue job: %v", err)
	}

	readiness := health.NewReadiness()
	server := api.NewAPIServer(testConfig("127.0.0.1:0"), http.NewServeMux(), readiness, pool, q)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Run(ctx)
	}()

	waitReady(t, readiness)
	waitJobStatus(t, jobSvc, job.ID, entity.JobRunning)
	cancel()

	if err := <-errCh; err != nil {
		t.Fatalf("Run(): %v", err)
	}

	assertJobFailedOnShutdown(t, jobSvc, job.ID)
}

func TestRunWithTLSEnabledServesHTTPS(t *testing.T) {
	jobSvc := service.NewJobService(jobrepo.NewMemoryJobRepository())
	userSvc := service.NewUserService(userrepo.NewMemoryUserRepository())
	q := queue.New(1)
	hub := stream.NewHub(16)
	pool := workerpool.NewWorkerPool(q, jobSvc, userSvc, metrics.DefaultHTTP(), hub)
	if err := pool.Start(context.Background(), 0); err != nil {
		t.Fatalf("start worker pool: %v", err)
	}

	httpAddr := reserveAddr(t)
	httpsAddr := reserveAddr(t)
	certFile, keyFile := writeSelfSignedKeyPair(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/livez", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := testConfig(httpAddr)
	cfg.HTTP.TLS = config.TLSConfig{
		Enable:   true,
		Addr:     httpsAddr,
		CertFile: certFile,
		KeyFile:  keyFile,
	}

	readiness := health.NewReadiness()
	server := api.NewAPIServer(cfg, mux, readiness, pool, q)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Run(ctx)
	}()

	waitReady(t, readiness)

	client := &http.Client{
		Timeout: time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := client.Get("https://" + httpsAddr + "/livez")
	if err != nil {
		cancel()
		<-errCh
		t.Fatalf("https GET /livez: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		cancel()
		<-errCh
		t.Fatalf("https status=%d want=%d", resp.StatusCode, http.StatusOK)
	}

	cancel()
	if err := <-errCh; err != nil {
		t.Fatalf("Run(): %v", err)
	}
}

func TestRunWithTLSEnabledMissingKeyPairDoesNotSetReady(t *testing.T) {
	jobSvc := service.NewJobService(jobrepo.NewMemoryJobRepository())
	userSvc := service.NewUserService(userrepo.NewMemoryUserRepository())
	q := queue.New(1)
	hub := stream.NewHub(16)
	pool := workerpool.NewWorkerPool(q, jobSvc, userSvc, metrics.DefaultHTTP(), hub)
	if err := pool.Start(context.Background(), 0); err != nil {
		t.Fatalf("start worker pool: %v", err)
	}
	defer stopPool(t, pool)

	cfg := testConfig(reserveAddr(t))
	cfg.HTTP.TLS = config.TLSConfig{
		Enable:   true,
		Addr:     reserveAddr(t),
		CertFile: filepath.Join(t.TempDir(), "missing.crt"),
		KeyFile:  filepath.Join(t.TempDir(), "missing.key"),
	}

	readiness := health.NewReadiness()
	server := api.NewAPIServer(cfg, http.NewServeMux(), readiness, pool, q)

	if err := server.Run(context.Background()); err == nil {
		t.Fatal("Run() error = nil, want tls key pair error")
	}
	if readiness.IsReady() {
		t.Fatal("readiness=true after tls key pair error")
	}
}

type blockingUserRepo struct {
	saveStarted chan struct{}
}

func (r *blockingUserRepo) GetAll(ctx context.Context) ([]*entity.User, error) {
	return nil, nil
}

func (r *blockingUserRepo) GetByID(ctx context.Context, id int) (*entity.User, error) {
	return nil, entity.ErrUserNotFound
}

func (r *blockingUserRepo) Save(ctx context.Context, user *entity.User) error {
	select {
	case <-r.saveStarted:
	default:
		close(r.saveStarted)
	}

	<-ctx.Done()
	return ctx.Err()
}

func (r *blockingUserRepo) Delete(ctx context.Context, id int) error {
	return entity.ErrUserNotFound
}

func (r *blockingUserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return false, nil
}

func testConfig(addr string) config.Config {
	return config.Config{
		HTTP: config.HTTPConfig{
			Addr:            addr,
			ShutdownTimeout: time.Second,
		},
	}
}

func waitReady(t *testing.T, readiness *health.Readiness) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for !readiness.IsReady() {
		if time.Now().After(deadline) {
			t.Fatal("timeout waiting for readiness=true")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func waitJobStatus(t *testing.T, jobSvc *service.JobService, id int64, want entity.JobStatus) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for {
		job, err := jobSvc.GetByID(context.Background(), id)
		if err != nil {
			t.Fatalf("GetByID(%d): %v", id, err)
		}
		if job.Status == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for job %d status=%s, last=%s", id, want, job.Status)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func assertJobFailedOnShutdown(t *testing.T, jobSvc *service.JobService, id int64) {
	t.Helper()

	job, err := jobSvc.GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("GetByID(%d): %v", id, err)
	}
	if job.Status != entity.JobFailed {
		t.Fatalf("job %d status=%s want=%s", id, job.Status, entity.JobFailed)
	}
	if job.Error == nil {
		t.Fatalf("job %d error=nil", id)
	}
	if job.Error.Status != http.StatusServiceUnavailable {
		t.Fatalf("job %d error.status=%d want=%d", id, job.Error.Status, http.StatusServiceUnavailable)
	}
	if job.Error.Detail != "job canceled: server shutting down" {
		t.Fatalf("job %d error.detail=%q", id, job.Error.Detail)
	}
}

func stopPool(t *testing.T, pool *workerpool.WorkerPool) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := pool.Stop(ctx)
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("stop worker pool: %v", err)
	}
}

func reserveAddr(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	return addr
}

func writeSelfSignedKeyPair(t *testing.T) (string, string) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "127.0.0.1",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	certPath := filepath.Join(t.TempDir(), "server.crt")
	keyPath := filepath.Join(t.TempDir(), "server.key")

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}

	return certPath, keyPath
}
