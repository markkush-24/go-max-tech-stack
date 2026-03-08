package router_test

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"pet-study/internal/entity"
	"pet-study/internal/testkit"
	"strings"
	"testing"
)

func TestUsersGetByID_ETag_IfNoneMatch(t *testing.T) {
	server, app := testkit.NewServer(t)

	u, err := app.UserSvc.CreateUser(context.Background(), &entity.CreateUserInput{
		Name:  "IVAN",
		Age:   21,
		Email: "ivan@gmail.com",
	})
	if err != nil {
		t.Fatalf("Error when try to create User: %v", err)
	}

	url := fmt.Sprintf("%s/api/v1/users/%d", server.URL, u.ID)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/json")
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("get #1: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get #1 status=%d want=%d", resp.StatusCode, http.StatusOK)
	}

	rawCT := resp.Header.Get("Content-Type")
	mt, _, err := mime.ParseMediaType(rawCT)
	if err != nil {
		t.Fatalf("bad Content-Type %q: %v", rawCT, err)
	}
	if mt != "application/json" {
		t.Fatalf("Content-Type=%q want=%q", mt, "application/json")
	}

	etag := resp.Header.Get("ETag")
	if etag == "" {
		t.Fatalf("%s empty", "ETag")
	}

	vary := resp.Header.Get("Vary")
	if !strings.Contains(vary, "Accept") {
		t.Fatalf("Vary=%q want to contain %q", vary, "Accept")
	}

	req2, _ := http.NewRequest("GET", url, nil)
	req2.Header.Set("Accept", "application/json")
	req2.Header.Set("If-None-Match", etag)

	resp2, err := server.Client().Do(req2)
	if err != nil {
		t.Fatalf("get #2: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusNotModified {
		t.Fatalf("get #2 status=%d want=%d", resp2.StatusCode, http.StatusNotModified)
	}

	// body must be empty
	b, err := io.ReadAll(resp2.Body)
	if err != nil {
		t.Fatalf("read body #2: %v", err)
	}
	if len(b) != 0 {
		t.Fatalf("get #2 body len=%d want=0; body=%q", len(b), string(b))
	}

	vary2 := resp2.Header.Get("Vary")
	if !strings.Contains(vary2, "Accept") {
		t.Fatalf("get #2 Vary=%q want contain %q", vary2, "Accept")
	}

	etag2 := resp2.Header.Get("ETag")
	if etag2 != "" && etag2 != etag {
		t.Fatalf("get #2 ETag=%q want %q", etag2, etag)
	}

	req3, _ := http.NewRequest("GET", url, nil)
	req3.Header.Set("Accept", "application/json")
	req3.Header.Set("If-None-Match", `W/"nope-etag"`)

	resp3, err := server.Client().Do(req3)
	if err != nil {
		t.Fatalf("get #3: %v", err)
	}
	defer resp3.Body.Close()

	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("get #3 status=%d want=%d", resp3.StatusCode, http.StatusOK)
	}

	rawCT3 := resp3.Header.Get("Content-Type")
	mt3, _, err := mime.ParseMediaType(rawCT3)
	if err != nil {
		t.Fatalf("bad Content-Type #3 %q: %v", rawCT3, err)
	}
	if mt3 != "application/json" {
		t.Fatalf("Content-Type #3=%q want=%q", mt3, "application/json")
	}

	b3, err := io.ReadAll(resp3.Body)
	if err != nil {
		t.Fatalf("read body #3: %v", err)
	}
	if len(b3) == 0 {
		t.Fatalf("get #3 body empty; want non-empty")
	}

	// опционально: ETag должен быть тем же, что и в #1 (ресурс не менялся)
	etag3 := resp3.Header.Get("ETag")
	if etag3 == "" {
		t.Fatalf("ETag #3 empty")
	}
	if etag3 != etag {
		t.Fatalf("ETag #3=%q want %q", etag3, etag)
	}

	vary3 := resp3.Header.Get("Vary")
	if !strings.Contains(vary3, "Accept") {
		t.Fatalf("get #3 Vary=%q want contain %q", vary3, "Accept")
	}
}
