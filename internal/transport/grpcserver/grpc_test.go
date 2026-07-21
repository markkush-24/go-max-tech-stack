package grpcserver_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net"
	"pet-study/internal/entity"
	"pet-study/internal/interceptors"
	"pet-study/internal/security"
	"pet-study/internal/service"
	"pet-study/internal/store/jobrepo"
	"pet-study/internal/transport/grpcserver"
	"pet-study/internal/transport/pb"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024
const testAdminToken = "admin-token"

func newBufconnJobsClient(
	t *testing.T,
	jobSvc *service.JobService,
	logger *slog.Logger,
	verifier security.Verifier,
) (pb.JobsServiceClient, func()) {
	t.Helper()

	authInterceptor, err := interceptors.UnaryAuthenticate(verifier)
	if err != nil {
		t.Fatalf("UnaryAuthenticate: %v", err)
	}

	lis := bufconn.Listen(bufSize)
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.UnaryRequestIDAndLogging(logger),
			authInterceptor,
		),
	)
	pb.RegisterJobsServiceServer(server, grpcserver.NewJobServer(jobSvc))

	go func() {
		if err := server.Serve(lis); err != nil {
			_ = err
		}
	}()

	dialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	cleanup := func() {
		_ = conn.Close()
		server.Stop()
		_ = lis.Close()
	}

	return pb.NewJobsServiceClient(conn), cleanup
}

func newDefaultBufconnJobsClient(t *testing.T, jobSvc *service.JobService, logger *slog.Logger) (pb.JobsServiceClient, func()) {
	t.Helper()

	return newBufconnJobsClient(t, jobSvc, logger, grpcTestVerifier{
		tokens: map[string]security.Principal{
			testAdminToken: {UserID: 999, Role: security.RoleAdmin},
		},
	})
}

func grpcAuthorizedContext(ctx context.Context, token string) context.Context {
	return metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+token))
}

type grpcTestVerifier struct {
	tokens map[string]security.Principal
}

func (v grpcTestVerifier) Verify(token string) (security.Principal, error) {
	if principal, ok := v.tokens[token]; ok {
		return principal, nil
	}
	return security.Principal{}, &security.AuthNError{Kind: security.AuthNInvalid}
}

func TestJobsServiceGetJob_Success_LogsRequestID(t *testing.T) {
	jobRepo := jobrepo.NewMemoryJobRepository()
	jobSvc := service.NewJobService(jobRepo)
	job := entity.Job{Status: entity.JobSucceeded}
	if err := jobSvc.Save(context.Background(), &job); err != nil {
		t.Fatalf("save job: %v", err)
	}

	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	client, cleanup := newDefaultBufconnJobsClient(t, jobSvc, logger)
	defer cleanup()

	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"request-id", "rid-test-1",
		"authorization", "Bearer "+testAdminToken,
	))
	var header metadata.MD
	resp, err := client.GetJob(ctx, &pb.GetJobRequest{Id: job.ID}, grpc.Header(&header))
	if err != nil {
		t.Fatalf("GetJob: %v", err)
	}
	if resp.GetId() != job.ID {
		t.Fatalf("id=%d want=%d", resp.GetId(), job.ID)
	}
	if resp.GetStatus() != pb.JobStatus_JOB_STATUS_SUCCEEDED {
		t.Fatalf("status=%v want=%v", resp.GetStatus(), pb.JobStatus_JOB_STATUS_SUCCEEDED)
	}
	if got := header.Get("request-id"); len(got) != 1 || got[0] != "rid-test-1" {
		t.Fatalf("response request-id metadata=%v want rid-test-1", got)
	}

	logText := logs.String()
	if !strings.Contains(logText, "request_id=rid-test-1") {
		t.Fatalf("log missing request_id: %s", logText)
	}
	if !strings.Contains(logText, "method=/pb.JobsService/GetJob") {
		t.Fatalf("log missing method: %s", logText)
	}
	if !strings.Contains(logText, "code=OK") {
		t.Fatalf("log missing code=OK: %s", logText)
	}
}

func TestJobsServiceGetJob_InvalidRequestIDIsSanitizedAndReturned(t *testing.T) {
	jobRepo := jobrepo.NewMemoryJobRepository()
	jobSvc := service.NewJobService(jobRepo)
	job := entity.Job{Status: entity.JobSucceeded}
	if err := jobSvc.Save(context.Background(), &job); err != nil {
		t.Fatalf("save job: %v", err)
	}

	client, cleanup := newDefaultBufconnJobsClient(t, jobSvc, slog.New(slog.NewTextHandler(ioDiscard{}, nil)))
	defer cleanup()

	invalidRID := strings.Repeat("a", 129)
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"request-id", invalidRID,
		"authorization", "Bearer "+testAdminToken,
	))
	var header metadata.MD
	_, err := client.GetJob(ctx, &pb.GetJobRequest{Id: job.ID}, grpc.Header(&header))
	if err != nil {
		t.Fatalf("GetJob: %v", err)
	}

	got := header.Get("request-id")
	if len(got) != 1 || got[0] == "" {
		t.Fatalf("response request-id metadata=%v", got)
	}
	if got[0] == invalidRID {
		t.Fatalf("invalid request-id was echoed")
	}
	if len(got[0]) > 128 {
		t.Fatalf("generated request-id too long: %q", got[0])
	}
}

func TestJobsServiceGetJob_InvalidArgument(t *testing.T) {
	jobSvc := service.NewJobService(jobrepo.NewMemoryJobRepository())
	logger := slog.New(slog.NewTextHandler(ioDiscard{}, nil))
	client, cleanup := newDefaultBufconnJobsClient(t, jobSvc, logger)
	defer cleanup()

	_, err := client.GetJob(grpcAuthorizedContext(context.Background(), testAdminToken), &pb.GetJobRequest{Id: 0})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code=%v want=%v err=%v", status.Code(err), codes.InvalidArgument, err)
	}
}

func TestJobsServiceGetJob_NotFound(t *testing.T) {
	jobSvc := service.NewJobService(jobrepo.NewMemoryJobRepository())
	logger := slog.New(slog.NewTextHandler(ioDiscard{}, nil))
	client, cleanup := newDefaultBufconnJobsClient(t, jobSvc, logger)
	defer cleanup()

	_, err := client.GetJob(grpcAuthorizedContext(context.Background(), testAdminToken), &pb.GetJobRequest{Id: 999999})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("code=%v want=%v err=%v", status.Code(err), codes.NotFound, err)
	}
}

func TestJobsServiceGetJob_ContextErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code codes.Code
	}{
		{
			name: "canceled",
			err:  context.Canceled,
			code: codes.Canceled,
		},
		{
			name: "deadline exceeded",
			err:  context.DeadlineExceeded,
			code: codes.DeadlineExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobSvc := service.NewJobService(failingJobRepository{getErr: tt.err})
			logger := slog.New(slog.NewTextHandler(ioDiscard{}, nil))
			client, cleanup := newDefaultBufconnJobsClient(t, jobSvc, logger)
			defer cleanup()

			_, err := client.GetJob(grpcAuthorizedContext(context.Background(), testAdminToken), &pb.GetJobRequest{Id: 1})
			if status.Code(err) != tt.code {
				t.Fatalf("code=%v want=%v err=%v", status.Code(err), tt.code, err)
			}
		})
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

type failingJobRepository struct {
	getErr error
}

func (r failingJobRepository) GetAll(context.Context) ([]*entity.Job, error) {
	return nil, errors.New("not implemented")
}

func (r failingJobRepository) GetByID(context.Context, int64) (*entity.Job, error) {
	return nil, r.getErr
}

func (r failingJobRepository) Save(context.Context, *entity.Job) error {
	return errors.New("not implemented")
}

func (r failingJobRepository) Delete(context.Context, int64) error {
	return errors.New("not implemented")
}

func (r failingJobRepository) SetRunning(context.Context, int64) error {
	return errors.New("not implemented")
}

func (r failingJobRepository) SetSucceeded(context.Context, int64, entity.JobResult) error {
	return errors.New("not implemented")
}

func (r failingJobRepository) SetFailed(context.Context, int64, entity.JobProblem) error {
	return errors.New("not implemented")
}

func (r failingJobRepository) FailActive(context.Context, entity.JobProblem) (int, error) {
	return 0, errors.New("not implemented")
}
