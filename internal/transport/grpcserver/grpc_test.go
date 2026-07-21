package grpcserver_test

import (
	"bytes"
	"context"
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
	resp, err := client.GetJob(ctx, &pb.GetJobRequest{Id: job.ID})
	if err != nil {
		t.Fatalf("GetJob: %v", err)
	}
	if resp.GetId() != job.ID {
		t.Fatalf("id=%d want=%d", resp.GetId(), job.ID)
	}
	if resp.GetStatus() != pb.JobStatus_JOB_STATUS_SUCCEEDED {
		t.Fatalf("status=%v want=%v", resp.GetStatus(), pb.JobStatus_JOB_STATUS_SUCCEEDED)
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

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
