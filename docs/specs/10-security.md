# Auth7 — Spec 10: Security Posture & Banking Compliance

> **Versi**: 1.0-draft | **Tanggal**: 2026-04-22 | **Fase**: Brainstorming

---

## 1. Security Overview

Auth7 harus memenuhi standar keamanan untuk sistem perbankan Indonesia:
- **OJK POJK No. 11/POJK.03/2022** — Tata kelola teknologi informasi bank umum
- **BI PBI No. 23/6/PBI/2021** — Penyelenggaraan transfer dana
- **ISO/IEC 27001:2022** — Information security management
- **OWASP Top 10** — Web application security

---

## 2. Cryptography Standards

### 2.1 Password Hashing

**Argon2id** (winner dari Password Hashing Competition 2015):

```go
const (
    Argon2Memory      = 65536     // 64 MB
    Argon2Iterations  = 3         // time cost
    Argon2Parallelism = 4         // threads
    Argon2SaltLength  = 16        // bytes
    Argon2KeyLength   = 32        // bytes output
)

// Hash
hash, err := argon2.CreateHash(password, &argon2.Params{
    Memory:      Argon2Memory,
    Iterations:  Argon2Iterations,
    Parallelism: Argon2Parallelism,
    SaltLength:  Argon2SaltLength,
    KeyLength:   Argon2KeyLength,
})

// Format output: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
```

Mengapa Argon2id vs bcrypt:
- Argon2id: memory-hard (tahan GPU attack) + side-channel resistant
- bcrypt: masih acceptable tapi lebih lama tergantikan; tidak memory-hard

| Property | Value |
|---|---|
| Algorithm | Argon2id |
| Memory | 64 MB |
| Iterations | 3 |
| Parallelism | 4 |
| Key length | 32 bytes |
| Salt length | 16 bytes |

### 2.2 JWT Signing

```
Algorithm:    RS256 (RSASSA-PKCS1-v1_5 + SHA-256)
Key size:     RSA 2048-bit minimum (4096-bit untuk produksi sensitif)
Alternative:  ES256 (ECDSA P-256) — lebih kecil signature, sama amannya

v1.0: RS256 (compatibility lebih luas)
v2.0: pertimbangkan ES256
```

| Property | Value |
|---|---|
| Algorithm | RS256 (RSA 2048-bit) |
| Key rotation | 90 hari |
| Private key | Stored encrypted at-rest |
| Public key | JWKS endpoint |

### 2.3 Data Encryption at Rest

**TOTP Secrets & JWT Private Keys:**
```
Algorithm: AES-256-GCM (authenticated encryption)
KEK:       32-byte key dari environment variable (atau Vault)
Format:    nonce (12 bytes) || ciphertext || tag (16 bytes)
```

**Database-level:**
- PostgreSQL TDE (Transparent Data Encryption) — bergantung pada infrastructure
- Minimal: enkripsi full-disk di server PostgreSQL

| Data | Method |
|---|---|
| TOTP secret | AES-256-GCM |
| Refresh token hash | SHA-256 |
| Client secret | Argon2id (same as password) |
| Backup codes | SHA-256 (hashed) |

### 2.4 Key Encryption Key (KEK)

- v1.0: Software encryption (KEK dari env var)
- v2.0: HSM/Vault jika ada requirement regulator OJK

```env
AUTH7_ENCRYPTION_KEY=${AUTH7_ENCRYPTION_KEY}  # 256-bit key
```

### 2.5 Transport Security

```
HTTPS mandatory: TLS 1.2 minimum, TLS 1.3 preferred
Certificate: harus valid (bukan self-signed di production)
HSTS: Strict-Transport-Security: max-age=31536000; includeSubDomains
```

---

## 3. Secure Development Practices

### 3.1 Input Validation

```go
// Semua input di-validate sebelum diproses
type LoginRequest struct {
    Username string `json:"username" validate:"required,min=3,max=100,alphanum_underscore_dot"`
    Password string `json:"password" validate:"required,min=8,max=128"`
    OrgID    string `json:"org_id"   validate:"required,uuid4"`
}

// Custom validators
validate.RegisterValidation("alphanum_underscore_dot", func(fl validator.FieldLevel) bool {
    return regexp.MustCompile(`^[a-zA-Z0-9._-]+$`).MatchString(fl.Field().String())
})
```

### 3.2 SQL Injection Prevention

```go
// Selalu gunakan parameterized queries (sqlc generate ini)
// TIDAK PERNAH string concatenation untuk SQL

// ✅ Benar (sqlc generated)
func (q *Queries) GetUserByUsername(ctx context.Context, arg GetUserByUsernameParams) (User, error) {
    row := q.db.QueryRowContext(ctx, getUserByUsername, arg.OrgID, arg.Username)
    ...
}

// ❌ Salah
query := "SELECT * FROM users WHERE username = '" + username + "'"
```

### 3.3 XSS Prevention

```go
// Semua output di-escape
// JSON response: Go's encoding/json auto-escape
// Error messages: tidak boleh reflect user input langsung
func writeError(c *gin.Context, code int, msg string) {
    c.JSON(code, gin.H{
        "error": html.EscapeString(msg),  // extra safety
    })
}
```

### 3.4 CSRF Protection

```
Cookie auth:
- SameSite=Strict pada session cookie → CSRF prevention
- Double submit cookie pattern untuk form POST

API calls:
- Stateless JWT → tidak rentan CSRF (tidak ada cookie-based auth untuk API)
```

### 3.5 Rate Limiting

```
Auth endpoints (per IP + per username/org):
  POST /auth/login:         5 req/min per username, 100 req/min per IP
  POST /auth/register:      3 req/hour per IP
  POST /auth/recover:       3 req/hour per email
  POST /oauth2/token:       60 req/min per client_id

Admin endpoints (per IP):
  GET  /admin/v1/*:         60 req/min
  POST /admin/v1/*:         30 req/min

Public endpoints (per IP):
  GET  /.well-known/*:      100 req/min
  GET  /oauth2/userinfo:    60 req/min
```

**Implementasi:**
```go
// Redis-based sliding window rate limiter
type RateLimiter struct {
    redis  *redis.Client
    limits map[string]RateLimit
}

type RateLimit struct {
    Requests int
    Window   time.Duration
}

// Key: "rate:{endpoint}:{identifier}"
// Value: sliding window counter
```

---

## 4. Brute Force Protection

### 4.1 Login Protection

```
Progressive delays:
  1-3 failures:   Proceed normally (slow hash provides natural delay ~300ms)
  4-5 failures:   429 Too Many Requests + 1 min cooldown
  6-9 failures:   429 + 5 min cooldown
  10+ failures:   Account locked (manual admin unlock required)

Lockout scope: per (org_id, username) — bukan per IP (prevent DoS via lockout)
IP-based:      Suspicious flag setelah 50 failures dari satu IP (log + alert)

Storage: Redis counter dengan TTL = window
  Key: "rate:login:{org_id}:{username}"
  TTL: 15 menit (reset setelah berhasil login)
```

### 4.2 TOTP Protection

```
3 failures: 429 + 1 min cooldown
5 failures: Force re-login (session terminated)
Scope: per user_id (bukan IP)
Key: "rate:totp:{user_id}"
```

### 4.3 Password Recovery Protection

```
3 requests per email per hour
After limit: tetap return 200 (jangan bocorkan email existence)
Alert: kirim notif ke admin jika banyak recovery request
```

---

## 5. Security Headers

```go
// Middleware untuk security headers
func SecurityHeaders() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        c.Header("X-Content-Type-Options", "nosniff")
        c.Header("X-Frame-Options", "DENY")
        c.Header("X-XSS-Protection", "1; mode=block")
        c.Header("Content-Security-Policy", "default-src 'none'")
        c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
        c.Header("Cache-Control", "no-store")  // penting untuk auth responses
        c.Header("Pragma", "no-cache")
        c.Next()
    }
}
```

| Header | Value |
|---|---|
| `Strict-Transport-Security` | `max-age=31536000; includeSubDomains` |
| `X-Content-Type-Options` | `nosniff` |
| `X-Frame-Options` | `DENY` |
| `X-XSS-Protection` | `1; mode=block` |
| `Content-Security-Policy` | `default-src 'none'` |
| `Referrer-Policy` | `strict-origin-when-cross-origin` |
| `Permissions-Policy` | `camera=(), microphone=(), geolocation=()` |
| `Cache-Control` | `no-store` |

---

## 6. Audit & Logging Security

### 6.1 What Must Be Logged (Banking Requirement)

```
✅ Semua authentication events (login, logout, failed, MFA)
✅ Semua authorization changes (role assign/revoke, permission change)
✅ Semua admin actions (user create/lock/delete, client create)
✅ Semua token operations (issue, revoke, introspect)
✅ All security events (brute force, suspicious IP, account lock)

❌ TIDAK PERNAH log:
  - Password (plaintext atau hash)
  - TOTP secrets
  - Access tokens / refresh tokens
  - Client secrets
  - Personal data (lebih dari yang diperlukan untuk audit)
```

### 6.2 Log Format

```json
{
  "timestamp": "2026-04-22T08:00:00.123Z",
  "level": "info",
  "service": "auth7",
  "trace_id": "uuid",
  "event_type": "user.login",
  "user_id": "uuid",
  "org_id": "uuid",
  "ip": "10.0.1.5",
  "user_agent": "Mozilla/5.0...",
  "success": true
}
```

### 6.3 Audit Log Immutability

```
Requirement: Audit log TIDAK BOLEH dimodifikasi atau dihapus.
Implementation:
  - PostgreSQL: row-level security, REVOKE DELETE/UPDATE pada audit_logs
  - Application: tidak ada code path yang update/delete audit_logs
  - Backup: daily backup audit logs ke cold storage (object storage)
  - Retention: minimum 5 tahun (banking regulation)
```

### 6.4 Events Logged

| Event | Details |
|---|---|
| Login success/fail | IP, user agent, method |
| Logout | IP, session ID |
| MFA enroll/verify | Method, success/fail |
| Password change | User ID, admin ID (if admin-initiated) |
| Role assignment | Admin ID, user ID, role |
| Client create/update | Admin ID, client ID |
| Token revoke | Reason, user ID |
| Admin actions | All CRUD operations |

---

## 7. Vulnerability Management

### 7.1 Dependency Scanning

```bash
# Dalam CI/CD pipeline:
go list -json -m all | nancy sleuth      # OSS vulnerability scanner
govulncheck ./...                        # Go official vuln checker
```

### 7.2 Static Analysis

```bash
golangci-lint run --enable=gosec        # Go security linter
# Rules yang wajib enable:
#   G101: hardcoded credentials
#   G401: weak crypto (MD5, SHA1 untuk password)
#   G501: import of dangerous packages
#   G601: implicit memory aliasing
```

### 7.3 OWASP Top 10 Mapping

| OWASP | Mitigation di Auth7 |
|---|---|
| A01: Broken Access Control | RBAC + ABAC, default deny, tenant-scoped |
| A02: Cryptographic Failures | Argon2id, AES-256-GCM, RS256 JWT |
| A03: Injection | sqlc parameterized queries, input validation |
| A04: Insecure Design | Security by design, audit requirements |
| A05: Security Misconfiguration | Strict defaults, no default credentials |
| A06: Vulnerable Components | govulncheck, dependency updates |
| A07: Authentication Failures | Brute force protection, MFA, secure sessions |
| A08: Integrity Failures | JWT signature verification, token rotation |
| A09: Logging Failures | Immutable audit log, all auth events |
| A10: SSRF | No outbound requests to user-provided URLs |

---

## 8. Secrets Management

### 8.1 What Are Secrets

```
- Database connection string (password)
- Redis connection string (password)
- JWT private key encryption key (KEK)
- OAuth2 client secrets
- Notification service API key (untuk notif7)
- SMTP credentials (jika direct email, bukan via notif7)
```

### 8.2 Secret Storage

```
v1.0:
  - Semua secrets via environment variables
  - Tidak ada secrets di config files (consistent dengan service7-template)
  - Docker secrets atau Kubernetes Secrets untuk deployment

v2.0:
  - HashiCorp Vault atau AWS Secrets Manager
  - Dynamic credentials (DB credentials rotate otomatis)
```

### 8.3 No Secrets in Config Files

```yaml
# ✅ GOOD
database:
  password: "${AUTH7_DB_PASSWORD}"

# ❌ BAD
database:
  password: "supersecret123"
```

### 8.4 Environment Variables

| Variable | Deskripsi |
|---|---|
| `AUTH7_DB_PASSWORD` | Database password |
| `AUTH7_REDIS_PASSWORD` | Redis password |
| `AUTH7_ENCRYPTION_KEY` | KEK for encryption at-rest |
| `AUTH7_JWT_PRIVATE_KEY_PATH` | Path to JWT private key |
| `AUTH7_JWT_KEK` | KEK for JWT private key encryption |
| `AUTH7_TOTP_EK` | Encryption key for TOTP secrets |
| `AUTH7_ISSUER` | JWT issuer URL |
| `NOTIF7_GRPC_ADDR` | notif7 gRPC address |

```bash
# auth7-svc/.env.example (TIDAK ada nilai asli)
DATABASE_URL="${AUTH7_DATABASE_URL}"
REDIS_URL="${AUTH7_REDIS_URL}"
JWT_KEY_ENCRYPTION_KEY="${AUTH7_JWT_KEK}"   # 32 bytes, base64
TOTP_ENCRYPTION_KEY="${AUTH7_TOTP_EK}"       # 32 bytes, base64
AUTH7_ISSUER="https://auth7.bank.co.id"
NOTIF7_GRPC_ADDR="${NOTIF7_GRPC_ADDR}"
```

---

## 9. Incident Response

### 9.1 Security Incident Types

| Insiden | Response Otomatis | Response Manual |
|---|---|---|
| Brute force login | Account lock setelah N failures | Admin review + unlock |
| Credential stuffing | IP rate limit + alert | Block IP range |
| Token theft detected | Revoke token family + alert | User notification |
| Admin account compromise | Force logout semua session | Emergency key rotation |
| Data breach | (audit log untuk forensics) | CSIRT procedure |

### 9.2 Emergency Procedures

```
Emergency Token Revocation:
  POST /admin/v1/system/emergency/revoke-all-tokens
  Body: { "reason": "security_incident", "org_id": "uuid" }
  Effect: invalidate semua access tokens + sessions untuk org
  Requires: super_admin + MFA

Emergency Key Rotation:
  POST /admin/v1/system/keys/emergency-rotate
  Effect: generate new key pair immediately, grace period = 0
  Requires: super_admin + MFA

Force All Logout:
  POST /admin/v1/system/emergency/force-logout-all
  Body: { "org_id": "uuid", "reason": "..." }
  Effect: revoke semua sessions + Redis flush untuk org
```

---

## 10. WAF (Web Application Firewall)

- **Level**: Infrastructure (bukan aplikasi)
- **Stack**: Nginx + ModSecurity di depan auth7-svc
- **Rules**: OWASP ModSecurity Core Rule Set (CRS)

---

## 11. Compliance Checklist

### 11.1 OJK Requirements

| Requirement | Status | Implementasi |
|---|---|---|
| Autentikasi kuat (min 2FA untuk privileged) | ✅ Plan | MFA required untuk supervisor+ |
| Log audit semua akses | ✅ Plan | audit_logs table, append-only |
| Password expiry (max 90 hari) | ✅ Plan | password_changed_at + policy |
| Session timeout (idle) | ✅ Plan | 30 menit idle timeout |
| Account lockout (brute force) | ✅ Plan | Progressive delays + lock at 10 |
| Enkripsi data sensitif | ✅ Plan | Argon2id + AES-256-GCM |

### 11.2 Penetration Testing Plan

Pre-production checklist:
- [ ] SQL injection testing semua input points
- [ ] JWT forgery / algorithm confusion attacks
- [ ] CSRF testing pada form-based flows
- [ ] Session fixation testing
- [ ] Token hijacking simulation
- [ ] Rate limit bypass testing
- [ ] Authorization bypass testing (IDOR, privilege escalation)
- [ ] Cryptographic implementation review

---

> Semua open questions telah dijawab di [OPEN-QUESTIONS.md](../OPEN-QUESTIONS.md).

*Prev: [09-integration.md](./09-integration.md) | Back to: [README.md](./README.md)*