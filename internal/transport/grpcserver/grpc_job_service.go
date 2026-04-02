package grpcserver

import (
	"context"
	"errors"
	"pet-study/internal/entity"
	"pet-study/internal/service"
	"pet-study/internal/transport/pb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type JobGRPCService struct {
	pb.UnimplementedJobsServiceServer
	jobService *service.JobService
}

func NewJobServer(jobService *service.JobService) *JobGRPCService {
	return &JobGRPCService{jobService: jobService}
}

// GetJob retrieves a job by ID. It returns a NotFound error if the job
// does not exist.
func (s *JobGRPCService) GetJob(ctx context.Context, req *pb.GetJobRequest) (*pb.Job, error) {
	// Validate the request
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request must not be nil")
	}
	if req.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "job ID must be positive")
	}

	job, err := s.jobService.GetByID(ctx, req.Id)
	if err != nil {
		// Return a gRPC status error with NotFound code
		if errors.Is(err, entity.ErrJobNotFound) {
			return nil, status.Errorf(codes.NotFound, "job with ID %d not found", req.Id)
		}
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.Job{
		Id:     job.ID,
		Status: mapJobStatus(job.Status),
	}, nil
}

func mapJobStatus(s entity.JobStatus) pb.JobStatus {
	switch s {
	case entity.JobQueued:
		return pb.JobStatus_JOB_STATUS_QUEUED
	case entity.JobRunning:
		return pb.JobStatus_JOB_STATUS_RUNNING
	case entity.JobSucceeded:
		return pb.JobStatus_JOB_STATUS_SUCCEEDED
	case entity.JobFailed:
		return pb.JobStatus_JOB_STATUS_FAILED
	default:
		return pb.JobStatus_JOB_STATUS_UNSPECIFIED
	}
}
