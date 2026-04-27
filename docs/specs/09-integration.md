# Auth7 — Spec 09: Integration dengan Core7

> **Versi**: 1.0-draft | **Tanggal**: 2026-04-22 | **Fase**: Brainstorming

---

## 1. Pola Integrasi

Auth7 terintegrasi dengan Core7 ecosystem melalui tiga mekanisme:

| Mekanisme | Kapan dipakai | Latency | Coupling |
|---|---|---|---|
| **JWT Verification** (stateless) | Setiap HTTP request ke service | < 1ms | Loose |
| **gRPC AuthCheck** | Permission check real-time | 1-5ms | Moderate |
| **Token Introspection** (HTTP) | Verifikasi detail token aktif | 5-10ms | Moderate |

---

## 2. JWT Verification (Stateless — Recommended)

### 2.1 Cara Kerja

Setiap Core7 service melakukan validasi JWT secara **lokal** tanpa memanggil auth7-svc:

```go
// Di core7 service (middleware)
func AuthMiddleware(jwksURL string) gin.HandlerFunc {
    // Cache JWKS keys, refresh tiap 1 jam
    keyFunc := jwks.NewCachingKeyFunc(jwksURL, time.Hour)
    
    return func(c *gin.Context) {
        tokenStr := extractBearerToken(c.Request)
        
        token, err := jwt.Parse(tokenStr, keyFunc, 
            jwt.WithValidMethods([]string{"RS256"}),
            jwt.WithIssuers("https://auth7.bank.co.id"),
            jwt.WithAudiences("workflow7"),   // service name sebagai audience
        )
        
        if err != nil {
            c.AbortWithStatus(401)
            return
        }
        
        // Extract claims dan inject ke context
        claims := token.Claims.(jwt.MapClaims)
        c.Set("user_id", claims["sub"])
        c.Set("org_id", claims["org_id"])
        c.Set("branch_id", claims["branch_id"])
        c.Set("roles", claims["roles"])
        c.Set("active_branch_id", claims["active_branch_id"])
        c.Set("assigned_branch_ids", claims["assigned_branch_ids"])
        c.Set("session_id", claims["sid"])
        
        c.Next()
    }
}
```

### 2.2 JWKS Caching Strategy

```go
type JWKSCache struct {
    keys       map[string]*rsa.PublicKey
    fetchedAt  time.Time
    refreshTTL time.Duration   // default: 1 hour
    mu         sync.RWMutex
}

// Saat key tidak ditemukan → refresh JWKS (untuk key rotation)
// Saat JWKS tidak tersedia → gunakan cache lama (graceful degradation)
```

### 2.3 Validasi Claims yang Wajib

Semua Core7 services wajib validate:
1. `exp` — token tidak expired
2. `iss` — issuer = `"https://auth7.bank.co.id"` (dari config)
3. `aud` — audience mencakup service name
4. `org_id` — sesuai dengan expected org (jika multi-org)

Optional tapi direkomendasikan:
5. `sid` — cek session masih active via Redis (untuk force-logout support)

### 2.4 Shared Library: `lib7-auth-go`

Auth7 menyediakan shared Go library untuk semua Core7 services:

```
libs/auth-go/          ← submodule lib7-auth-go
  ├── middleware/
  │     ├── gin.go     # Gin middleware
  │     └── grpc.go    # gRPC interceptor
  ├── jwks/
  │     └── client.go  # JWKS fetcher + cache
  ├── token/
  │     └── claims.go  # Claims extraction helpers
  └── authz/
        └── client.go  # gRPC authz client
```

**Usage di service:**
```go
import "github.com/ihsansolusi/lib7-auth-go/middleware"

// Gin
router.Use(middleware.Auth7JWT(cfg.Auth7.JWKSURL, cfg.Auth7.Issuer))

// gRPC
opts = append(opts, grpc.UnaryInterceptor(middleware.Auth7GRPCInterceptor(...)))
```

---

## 3. gRPC Permission Check

### 3.1 Protobuf Definition

```protobuf
syntax = "proto3";
package auth7.v1;

service AuthService {
  // Verify dan decode token
  rpc VerifyToken(VerifyTokenRequest) returns (VerifyTokenResponse);
  
  // Check single permission
  rpc CheckPermission(CheckPermRequest) returns (CheckPermResponse);
  
  // Check multiple permissions sekaligus
  rpc BatchCheckPermissions(BatchCheckRequest) returns (BatchCheckResponse);
  
  // Get full user info
  rpc GetUserInfo(GetUserInfoRequest) returns (UserInfo);
}

message VerifyTokenRequest {
  string token = 1;
  repeated string required_scopes = 2;   // optional
}

message VerifyTokenResponse {
  bool valid = 1;
  UserClaims claims = 2;
  string error_message = 3;
}

message UserClaims {
  string user_id = 1;
  string org_id = 2;
  string active_branch_id = 3;
  repeated string roles = 4;
  repeated string scopes = 5;
  string session_id = 6;
  string client_id = 7;
  int64 expires_at = 8;
  repeated string assigned_branch_ids = 9;
}

message CheckPermRequest {
  string user_id = 1;
  string org_id = 2;
  string resource = 3;
  string action = 4;
  map<string, string> context = 5;   // branch_id, ip, dll.
}

message CheckPermResponse {
  bool allowed = 1;
  repeated string reasons = 2;
}

message BatchCheckRequest {
  string user_id = 1;
  string org_id = 2;
  repeated PermCheck checks = 3;
  map<string, string> context = 4;
}

message PermCheck {
  string resource = 1;
  string action = 2;
}

message BatchCheckResponse {
  repeated PermCheckResult results = 1;
}

message PermCheckResult {
  string resource = 1;
  string action = 2;
  bool allowed = 3;
}
```

### 3.2 Penggunaan di Workflow7

```go
// workflow7-svc: cek permission sebelum approve task
func (s *TaskService) ApproveTask(ctx context.Context, taskID uuid.UUID) error {
    const op = "TaskService.ApproveTask"
    
    claims := auth7.ClaimsFromContext(ctx)
    
    // Check permission via gRPC
    resp, err := s.authzClient.CheckPermission(ctx, &authv1.CheckPermRequest{
        UserId:   claims.UserID,
        OrgId:    claims.OrgID,
        Resource: "workflow",
        Action:   "approve",
        Context: map[string]string{
            "active_branch_id": claims.ActiveBranchID,
        },
    })
    if err != nil {
        return fmt.Errorf("%s: authz check: %w", op, err)
    }
    if !resp.Allowed {
        return domain.ErrForbidden
    }
    
    // proceed with approval
    return s.store.ApproveTask(ctx, taskID, claims.UserID)
}
```

---

## 4. Integrasi per Service

### 4.1 workflow7

```
Auth7 → workflow7 integration:

1. JWT middleware: auth7-go middleware untuk semua endpoints
2. Claims → flow context: user_id, org_id, branch_id dipakai di flow instances
3. Permission checks:
   - workflow:create → buat instance baru
   - workflow:approve → approve user task
   - workflow:reject → reject user task
   - workflow:read → lihat task inbox
   - workflow:admin → kelola flow definitions
   - Menu/page visibility: menu:workflow:access → lihat menu workflow (4-layer auth model)
4. Branch scope: workflow instance dikategorikan per active_branch_id
5. Task assignment: workflow7 lookup user info via auth7 UserInfo endpoint
   (untuk display name, email di task inbox)
```

### 4.2 notif7

auth7-svc bertindak sebagai **producer** ke notif7 — bukan consumer. auth7 mengirim security alert events
setelah terjadi event keamanan penting (post-login). notif7 kemudian mengdeliver via in-app SSE + email.

```go
// internal/security/alert_dispatcher.go
// auth7-svc mengirim security alerts ke notif7 sebagai producer

notif7Client := notif7client.New(cfg.Notif7.BaseURL, cfg.Notif7.APIKey)

// Contoh: account locked event
_ = notif7Client.Send(ctx, notif7client.Event{
    Source:           "auth7",
    EventType:        "auth.account_locked",
    UserIDs:          []string{userID},
    EmailAddresses:   []string{userEmail},  // auth7 mengetahui email dari DB
    DeliveryChannels: []string{"in_app", "email"},
    Title:            "Akun Anda dikunci sementara",
    Body:             "Terdeteksi 5x percobaan login gagal. Akun dikunci 15 menit.",
    RefURL:           "/profile/security",
})
```

**Catatan arsitektur:**
- Email OTP / verification / recovery: tetap via auth7 internal SMTP (pre-login, tidak perlu notif7)
- Security alerts: via notif7 producer events (post-login, user_id tersedia)
- Dependency satu arah: auth7 → notif7 (tidak ada callback / circular dependency)

**EventType ke notif7 (v1.0):**

| EventType | Delivery |
|---|---|
| `auth.login_new_device` | in_app + email |
| `auth.account_locked` | in_app + email |
| `auth.mfa_reset` | in_app + email |
| `auth.password_changed` | in_app only |

### 4.3 bos7-portal / bos7-template (Next.js)

```
Browser Flow:
1. User akses bos7-portal
2. Redirect ke auth7-ui (login page)
3. Login berhasil → redirect kembali dengan authorization code
4. bos7-portal exchange code → tokens (OAuth2 PKCE flow)
5. Access token disimpan di memory (tidak di localStorage!)
6. Refresh token disimpan di HttpOnly cookie
7. Setiap API call: Authorization: Bearer <access_token>
8. Saat 401 → gunakan refresh token untuk renew
9. Saat refresh gagal → redirect ke login

SDK yang dipakai:
  - next-auth atau custom OAuth2 client
  - Atau: auth7 menyediakan official Next.js integration package (future)
```

### 4.4 service7-template

```
Penggunaan di service template:
1. middleware/auth.go: gunakan lib7-auth-go
2. domain/user.go: UserClaims dari context
3. store/*.go: multi-tenant query dengan org_id dari claims
4. Semua store method: terima branchID dari claims (bukan dari request body)
5. Permission check: gunakan 4-layer auth (page access, data access, branch scope, field masking)
```

---

## 5. Service-to-Service Auth (M2M)

### 5.1 Pattern

```
workflow7-svc perlu call notif7-svc:

1. workflow7-svc punya OAuth2 client_id + client_secret (confidential client)
2. Request token: POST /oauth2/token (client_credentials grant)
3. Dapatkan access token dengan scope "notif7:write"
4. Call notif7: Authorization: Bearer <m2m_token>
5. notif7 verify JWT (stateless) + check audience = "notif7"
```

### 5.2 M2M Client Registration

```
Setiap service yang perlu call service lain harus punya OAuth2 client:

- workflow7 → notif7:   client "workflow7-svc" dengan scope "notif7:write"
- auth7 → notif7:       client "auth7-svc" dengan scope "notif7:internal"
- (future) billing → core7: client "billing-svc" dengan scope "core7:read"
```

### 5.3 Token Caching (M2M)

```go
// M2M tokens bisa di-cache sampai mendekati expiry
type M2MTokenCache struct {
    token     string
    expiresAt time.Time
    mu        sync.Mutex
}

func (c *M2MTokenCache) Get(ctx context.Context, auth7Client Auth7Client) (string, error) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    // Buffer 60 detik sebelum expiry untuk renew
    if time.Now().Add(60 * time.Second).Before(c.expiresAt) {
        return c.token, nil
    }
    
    // Re-fetch
    token, exp, err := auth7Client.ClientCredentials(ctx, scopes)
    if err != nil {
        return "", err
    }
    c.token = token
    c.expiresAt = exp
    return token, nil
}
```

### 5.4 Setup auth7 sebagai notif7 producer

1. Dapatkan notif7 API key (producer JWT, issued by devops)
2. Set env: `NOTIF7_BASE_URL=http://notif7-svc:8082`, `NOTIF7_API_KEY=<jwt>`
3. Copy `pkg/notif7client/client.go` dari notif7 ke auth7 codebase
4. Wire `SecurityAlertDispatcher` di DI (cmd/)

Lihat detail implementasi di `06-mfa.md` Section 11.

---

## 6. Integration Testing

### 6.1 Mock Auth7 untuk Testing

```go
// lib7-auth-go menyediakan mock untuk testing
import "github.com/ihsansolusi/lib7-auth-go/mock"

// Dalam unit test
func TestApproveTask(t *testing.T) {
    mockAuthz := mock.NewAuthzClient()
    mockAuthz.SetPermission("user-id", "org-id", "workflow", "approve", true)
    
    svc := NewTaskService(store, mockAuthz)
    err := svc.ApproveTask(ctx, taskID)
    assert.NoError(t, err)
}
```

### 6.2 E2E Testing dengan Real Auth7

```yaml
# docker-compose.e2e.yml
services:
  auth7-svc:
    image: auth7:test
    environment:
      - DB_URL=postgres://...
      - REDIS_URL=redis://redis:6379
  
  workflow7-svc:
    environment:
      - AUTH7_JWKS_URL=http://auth7-svc:8080/.well-known/jwks.json
      - AUTH7_ISSUER=http://auth7-svc:8080
```

---

## 7. Auth7 SDK / Libraries

### 7.1 Go (lib7-auth-go)

```
Target consumers: workflow7-svc, notif7-svc, service7-template, dll.
Features:
  - JWT middleware (Gin + gRPC)
  - JWKS caching client
  - Permission check client (gRPC)
  - M2M token manager (client credentials)
  - Claims extraction helpers
  - Testing mock
```

### 7.2 TypeScript/Next.js (lib7-auth-ts) — future

```
Target consumers: bos7-portal, workflow7-web
Features:
  - OAuth2 PKCE client
  - Token storage (memory + HttpOnly cookie)
  - Automatic token refresh
  - Permission check hooks (usePermission)
  - Session management
```

---

> Semua open questions telah dijawab di [OPEN-QUESTIONS.md](../OPEN-QUESTIONS.md).

*Prev: [08-data-model.md](./08-data-model.md) | Next: [10-security.md](./10-security.md)*