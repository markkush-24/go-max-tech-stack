package security

// Этот файл намеренно **декларативный**.
//
// В Step 6 мы реализуем Authentication/Authorization/CORS/Proxy/Security-Headers.
// Чтобы избежать “разъезда” между документацией, middleware и тестами,
// каноническая матрица доступа хранится здесь (и в docs/security-policy.md).

// Role — имя роли в RBAC.
//
// В Step 6 роли намеренно минимальные.
type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

// Имена JWT-клеймов, используемые в Step 6.
const (
	ClaimSubject = "sub"  // user_id
	ClaimRole    = "role" // admin|user
)

// AccessLevel описывает требуемый уровень аутентификации/авторизации.
type AccessLevel uint8

const (
	AccessPublic AccessLevel = iota
	AccessAuthenticated
	AccessAdminOnly
)

// ResourceConstraint описывает дополнительные resource-level правила.
type ResourceConstraint uint8

const (
	ResourceNone ResourceConstraint = iota

	// ResourceSelfOnly означает: роль=user может получить доступ только когда claim `sub`
	// совпадает со значением переменной пути `{id}`.
	//
	// Роль=admin обходит эту проверку.
	ResourceSelfOnly
)

// RouteRule — декларативное правило доступа.
//
// Pattern — это точная строка паттерна net/http ServeMux, ожидаемая в r.Pattern.
// Для method-specific паттернов (Go 1.22+), Pattern включает префикс метода,
// например: "GET /api/v1/users".
type RouteRule struct {
	Pattern            string
	Methods            []string // информационно; enforcement может опираться на Pattern для method-specific правил
	Access             AccessLevel
	ResourceConstraint ResourceConstraint
	Notes              string
}

// DefaultPolicy — матрица доступа Step 6.
//
// Deny-by-default: любой маршрут, не перечисленный здесь, считается закрытым,
// если он явно не public (например, путь, который даёт 404). Реализация может
// выбрать стратегию для “unknown pattern”, но тесты должны фиксировать нужное поведение.
var DefaultPolicy = []RouteRule{
	// Health probes (public)
	{Pattern: "/livez", Methods: []string{"GET"}, Access: AccessPublic},
	{Pattern: "/readyz", Methods: []string{"GET"}, Access: AccessPublic},

	// Debug subtree (admin only). Примечание: маршрут существует только при HTTP_DEBUG=true.
	{Pattern: "/debug/", Methods: []string{"GET"}, Access: AccessAdminOnly, Notes: "только когда HTTP_DEBUG=true"},

	// API v1 users collection
	{Pattern: "GET /api/v1/users", Methods: []string{"GET"}, Access: AccessAdminOnly},
	{Pattern: "POST /api/v1/users", Methods: []string{"POST"}, Access: AccessAdminOnly},
	// Fallback handler для хелпера 405. Не должен стать обходом auth.
	{Pattern: "/api/v1/users", Methods: []string{"*"}, Access: AccessAdminOnly, Notes: "fallback коллекции (хелпер для 405)"},

	// API v1 users item
	{Pattern: "/api/v1/users/{id}", Methods: []string{"GET"}, Access: AccessAuthenticated, ResourceConstraint: ResourceSelfOnly},
	{Pattern: "/api/v1/users/{id}/profile", Methods: []string{"GET"}, Access: AccessAuthenticated, ResourceConstraint: ResourceSelfOnly},

	// SSE Stream
	{Pattern: "/api/v1/jobs/{id}/events", Methods: []string{"GET"}, Access: AccessAuthenticated},

	// API v1 jobs item (admin only; ownership jobs не моделируется в Step 6)
	{Pattern: "/api/v1/jobs/{id}", Methods: []string{"GET"}, Access: AccessAdminOnly},

	{Pattern: "/api/v1/jobs/{id}/grpc", Methods: []string{"GET"}, Access: AccessAdminOnly},

	// API v2 users collection
	{Pattern: "GET /api/v2/users", Methods: []string{"GET"}, Access: AccessAdminOnly},
	{Pattern: "POST /api/v2/users", Methods: []string{"POST"}, Access: AccessAdminOnly},
	{Pattern: "/api/v2/users", Methods: []string{"*"}, Access: AccessAdminOnly, Notes: "fallback коллекции (хелпер для 405)"},
}

func CanReadUser(p Principal, targetID int64) bool {
	return p.Role == RoleAdmin || p.UserID == targetID
}

func CanReadJob(p Principal, ownerID int64) bool {
	return p.Role == RoleAdmin || p.UserID == ownerID
}
