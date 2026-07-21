package grpcserver_test

import (
	"context"
	"log/slog"
	"testing"

	"pet-study/internal/entity"
	"pet-study/internal/security"
	"pet-study/internal/service"
	"pet-study/internal/store/jobrepo"
	"pet-study/internal/transport/pb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestJobsServiceGetJob_AuthnAndAuthz(t *testing.T) {
	jobSvc := service.NewJobService(jobrepo.NewMemoryJobRepository())
	job := entity.Job{Status: entity.JobSucceeded, OwnerUserID: 2}
	if err := jobSvc.Save(context.Background(), &job); err != nil {
		t.Fatalf("Save job: %v", err)
	}

	client, cleanup := newBufconnJobsClient(
		t,
		jobSvc,
		slog.New(slog.NewTextHandler(ioDiscard{}, nil)),
		grpcTestVerifier{
			tokens: map[string]security.Principal{
				"user-1": {UserID: 1, Role: security.RoleUser},
				"owner":  {UserID: 2, Role: security.RoleUser},
				"admin":  {UserID: 999, Role: security.RoleAdmin},
			},
		},
	)
	defer cleanup()

	t.Run("missing bearer is rejected", func(t *testing.T) {
		_, err := client.GetJob(context.Background(), &pb.GetJobRequest{Id: job.ID})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code=%v want=%v err=%v", status.Code(err), codes.Unauthenticated, err)
		}
	})

	t.Run("invalid bearer is rejected", func(t *testing.T) {
		_, err := client.GetJob(grpcAuthorizedContext(context.Background(), "invalid"), &pb.GetJobRequest{Id: job.ID})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code=%v want=%v err=%v", status.Code(err), codes.Unauthenticated, err)
		}
	})

	t.Run("user cannot read foreign job", func(t *testing.T) {
		_, err := client.GetJob(grpcAuthorizedContext(context.Background(), "user-1"), &pb.GetJobRequest{Id: job.ID})
		if status.Code(err) != codes.PermissionDenied {
			t.Fatalf("code=%v want=%v err=%v", status.Code(err), codes.PermissionDenied, err)
		}
	})

	t.Run("owner can read own job", func(t *testing.T) {
		resp, err := client.GetJob(grpcAuthorizedContext(context.Background(), "owner"), &pb.GetJobRequest{Id: job.ID})
		if err != nil {
			t.Fatalf("GetJob owner: %v", err)
		}
		if resp.GetId() != job.ID {
			t.Fatalf("id=%d want=%d", resp.GetId(), job.ID)
		}
	})

	t.Run("admin can read any job", func(t *testing.T) {
		resp, err := client.GetJob(grpcAuthorizedContext(context.Background(), "admin"), &pb.GetJobRequest{Id: job.ID})
		if err != nil {
			t.Fatalf("GetJob admin: %v", err)
		}
		if resp.GetId() != job.ID {
			t.Fatalf("id=%d want=%d", resp.GetId(), job.ID)
		}
	})
}
