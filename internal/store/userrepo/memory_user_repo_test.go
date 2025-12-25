package userrepo

import (
	"context"
	"errors"
	"pet-study/internal/entity"
	"testing"
)

func TestSave_AssignsID(t *testing.T) {
	repo := NewMemoryUserRepository()
	u := entity.User{
		Age:   23,
		Name:  "Sergei",
		Email: "sergo@mail.ru",
	}
	err := repo.Save(context.Background(), &u)
	if err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	if u.ID <= 0 {
		t.Fatalf("expected assigned ID > 0, got %d", u.ID)
	}
}

func TestGetByID_ReturnsCopy(t *testing.T) {
	repo := NewMemoryUserRepository()
	ctx := context.Background()

	orig := entity.User{Name: "Sergei", Age: 23, Email: "sergo@mail.ru"}
	if err := repo.Save(ctx, &orig); err != nil {
		t.Fatalf("Save(): %v", err)
	}

	got1, err := repo.GetByID(ctx, orig.ID)
	if err != nil {
		t.Fatalf("GetByID(first): %v", err)
	}

	got1.Name = "hacked"

	got2, err := repo.GetByID(ctx, orig.ID)
	if err != nil {
		t.Fatalf("GetByID(second): %v", err)
	}

	if got2.Name == "hacked" {
		t.Fatalf("expected repo to return a copy; mutation leaked back into repo")
	}

}

func TestGetAll_ReturnsCopies(t *testing.T) {
	repo := NewMemoryUserRepository()
	ctx := context.Background()

	orig1 := entity.User{Name: "Sergei", Age: 23, Email: "sergo@mail.ru"}
	if err := repo.Save(ctx, &orig1); err != nil {
		t.Fatalf("Save(orig1): %v", err)
	}
	orig2 := entity.User{Name: "Aleksey", Age: 25, Email: "alex@mail.ru"}
	if err := repo.Save(ctx, &orig2); err != nil {
		t.Fatalf("Save(orig2): %v", err)
	}

	before, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll(before): %v", err)
	}
	if len(before) == 0 {
		t.Fatalf("expected non-empty list")
	}

	// хакнем конкретный ID
	hackedID := before[0].ID
	before[0].Name = "hacked"

	after, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll(after): %v", err)
	}

	for _, u := range after {
		if u.ID == hackedID && u.Name == "hacked" {
			t.Fatalf("expected repo to return copies; mutation leaked for id=%d", hackedID)
		}
	}
}

func TestDelete_NotFound(t *testing.T) {
	repo := NewMemoryUserRepository()
	ctx := context.Background()

	err := repo.Delete(ctx, 999)

	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !errors.Is(err, entity.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestExistsByEmail_CaseInsensitive(t *testing.T) {
	repo := NewMemoryUserRepository()
	ctx := context.Background()

	orig := entity.User{Name: "Sergei", Age: 23, Email: "Sergo@mail.ru"}
	if err := repo.Save(ctx, &orig); err != nil {
		t.Fatalf("Save(): %v", err)
	}

	exists, err := repo.ExistsByEmail(ctx, "sergo@mail.ru")
	if err != nil {
		t.Fatalf("ExistsByEmail(): %v", err)
	}
	if !exists {
		t.Fatalf("expected ExistsByEmail to be case-insensitive")
	}
}
