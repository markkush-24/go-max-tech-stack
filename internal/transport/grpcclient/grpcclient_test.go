package grpcclient

import "testing"

func TestNewJobsClientWithConfigRejectsNegativeCallTimeout(t *testing.T) {
	client, conn, err := NewJobsClientWithConfig(Config{
		Addr:        "127.0.0.1:0",
		CallTimeout: -1,
	})
	if err == nil {
		if conn != nil {
			_ = conn.Close()
		}
		t.Fatalf("NewJobsClientWithConfig returned client=%v, want error", client)
	}
}
