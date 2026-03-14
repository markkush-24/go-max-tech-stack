package routes_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"pet-study/internal/entity"
	"pet-study/internal/testkit"
	"pet-study/internal/transport/pb"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
)

func TestUsersGetByID_Negotiate(t *testing.T) {
	server, app := testkit.NewServer(t)

	u, err := app.UserSvc.CreateUser(context.Background(), &entity.CreateUserInput{
		Name:  "IVAN",
		Age:   21,
		Email: "ivan@gmail.com",
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	url := fmt.Sprintf("%s/api/v1/users/%d", server.URL, u.ID)

	// --- #1 JSON ---
	req1, _ := http.NewRequest("GET", url, nil)
	req1.Header.Set("Accept", "application/json")

	resp1, err := server.Client().Do(req1)
	if err != nil {
		t.Fatalf("get #1: %v", err)
	}
	defer resp1.Body.Close()

	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("get #1 status=%d want=%d", resp1.StatusCode, http.StatusOK)
	}

	mt1, _, err := mime.ParseMediaType(resp1.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("bad Content-Type #1: %v", err)
	}
	if mt1 != "application/json" {
		t.Fatalf("Content-Type #1=%q want=%q", mt1, "application/json")
	}

	if vary := resp1.Header.Get("Vary"); !strings.Contains(vary, "Accept") {
		t.Fatalf("Vary #1=%q want contain %q", vary, "Accept")
	}

	body1, err := io.ReadAll(resp1.Body)
	if err != nil {
		t.Fatalf("read body #1: %v", err)
	}
	var dto entity.UserDTO
	if err := json.Unmarshal(body1, &dto); err != nil {
		t.Fatalf("unmarshal json #1: %v; body=%q", err, string(body1))
	}
	if dto.ID != u.ID || dto.Name != "IVAN" || dto.Email != "ivan@gmail.com" {
		t.Fatalf("json #1 dto=%+v want id=%d name=%q email=%q", dto, u.ID, "IVAN", "ivan@gmail.com")
	}

	// --- #2 Protobuf ---
	req2, _ := http.NewRequest("GET", url, nil)
	req2.Header.Set("Accept", "application/x-protobuf") // alias, ответ должен быть application/protobuf

	resp2, err := server.Client().Do(req2)
	if err != nil {
		t.Fatalf("get #2: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("get #2 status=%d want=%d", resp2.StatusCode, http.StatusOK)
	}

	mt2, _, err := mime.ParseMediaType(resp2.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("bad Content-Type #2: %v", err)
	}
	if mt2 != "application/protobuf" {
		t.Fatalf("Content-Type #2=%q want=%q", mt2, "application/protobuf")
	}

	if vary := resp2.Header.Get("Vary"); !strings.Contains(vary, "Accept") {
		t.Fatalf("Vary #2=%q want contain %q", vary, "Accept")
	}

	b2, err := io.ReadAll(resp2.Body)
	if err != nil {
		t.Fatalf("read body #2: %v", err)
	}

	var pu pb.User
	if err := proto.Unmarshal(b2, &pu); err != nil {
		t.Fatalf("unmarshal protobuf #2: %v", err)
	}
	if pu.Id != int64(u.ID) || pu.Name != "IVAN" || pu.Email != "ivan@gmail.com" {
		t.Fatalf("protobuf #2 user=%+v want id=%d name=%q email=%q", &pu, u.ID, "IVAN", "ivan@gmail.com")
	}

	// --- #3 Unsupported Accept -> 406 + Problem+JSON ---
	req3, _ := http.NewRequest("GET", url, nil)
	req3.Header.Set("Accept", "text/plain")

	resp3, err := server.Client().Do(req3)
	if err != nil {
		t.Fatalf("get #3: %v", err)
	}
	defer resp3.Body.Close()

	if resp3.StatusCode != http.StatusNotAcceptable {
		t.Fatalf("get #3 status=%d want=%d", resp3.StatusCode, http.StatusNotAcceptable)
	}

	mt3, _, err := mime.ParseMediaType(resp3.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("bad Content-Type #3: %v", err)
	}
	if mt3 != "application/problem+json" {
		t.Fatalf("Content-Type #3=%q want=%q", mt3, "application/problem+json")
	}
}
