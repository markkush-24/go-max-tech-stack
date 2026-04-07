package grpcclient

import (
	"net"
	"pet-study/internal/transport/pb"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewJobsClient(listenAddr string) (pb.JobsServiceClient, *grpc.ClientConn, error) {
	target, err := loopbackTarget(listenAddr)
	if err != nil {
		return nil, nil, err
	}

	conn, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, err
	}

	return pb.NewJobsServiceClient(conn), conn, nil
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
