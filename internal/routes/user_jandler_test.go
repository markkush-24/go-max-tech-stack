package routes_test

import (
	"bytes"
	"github.com/go-playground/validator/v10"
	"net/http"
	"net/http/httptest"
	"pet-study/internal/health"
	"pet-study/internal/router"
	"pet-study/internal/routes"
	"pet-study/internal/service"
	"pet-study/internal/store/userrepo"
	"testing"
)

func TestCreateUser_Created(t *testing.T) {
	// Arrange: собираем стек
	repo := userrepo.NewMemoryUserRepository()
	svc := service.NewUserService(repo)
	v := validator.New()
	uh := routes.NewUserHandler(svc, v)
	app := router.NewRouter(uh)

	ready := health.NewReadiness()
	ready.SetReady()
	root := router.NewRoot(app, router.NewHealthRouter(ready))

	// Act: делаем запрос
	body := `{"name":"mark","age":33,"email":"mark_kush@mail.ru"}`
	req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	root.ServeHTTP(rr, req)

	// Assert: только статус
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusCreated, rr.Body.String())
	}
}

func TestCreateUser_ValidationError(t *testing.T) {
	// Arrange: собираем стек
	repo := userrepo.NewMemoryUserRepository()
	svc := service.NewUserService(repo)
	v := validator.New()
	uh := routes.NewUserHandler(svc, v)
	app := router.NewRouter(uh)

	ready := health.NewReadiness()
	ready.SetReady()
	root := router.NewRoot(app, router.NewHealthRouter(ready))

	// Act: делаем запрос
	body := `{"name":"mark","age":33}`
	req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	// 3. Recorder для ответа
	rr := httptest.NewRecorder()

	// 4. Пропускаем через роутер
	root.ServeHTTP(rr, req)

	// 5. Проверяем статус-код
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected %d, got %d", http.StatusUnprocessableEntity, rr.Code)
	}
}
