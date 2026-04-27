# Auth7 — Spec 05: Session & Token Lifecycle

> **Versi**: 1.0-draft | **Tanggal**: 2026-04-22 | **Fase**: Brainstorming

---

## 1. Dua Mode Autentikasi

Auth7 mendukung dua mode yang digunakan secara bersamaan:

| Mode | Untuk | Storage | Revocation |
|---|---|---|---|
| **Session-based** | Browser apps (bos7-portal, auth7-ui) | Redis | Instant |
| **Token-based** | API clients, mobile, M2M | Stateless JWT | Blacklist (JWT) |

---

## 2. Session Management

### 2.1 Session Storage (Redis)

```
Key: session:{session_id}
Value: {
  "user_id": "user-uuid",
  "org_id": "org-uuid",
  "active_branch_id": "branch-uuid",     // branch yang sedang aktif
  "assigned_branch_ids": ["branch-uuid-1", "branch-uuid-2"],  // semua branch yang bisa diakses
  "client_id": "client-id",
  "ip_address": "192.168.1.1",
  "user_agent": "Mozilla/5.0...",
  "created_at": "2026-04-22T08:00:00Z",
  "expires_at": "2026-04-22T16:00:00Z",
  "last_active_at": "2026-04-22T10:30:00Z"
}
TTL: 8 jam (sesuai refresh token TTL)
```

### 2.2 Session Lifecycle

```
Login
  │
  ├── Create session entry di Redis
  │     Key: "session:{session_id}"
  │     Value: JSON (user_id, org_id, branch_id, roles, ...)
  │     TTL: 8 jam (configurable, banking: hari kerja)
  │
  ├── Set HTTP-only cookie: "auth7_session={session_id}"
  │     Flags: HttpOnly, Secure, SameSite=Strict
  │     Domain: .bank.co.id
  │
  └── Return access_token (short-lived, untuk API calls)
```

```
[Created] → [Active] → [Expired] → [Revoked]
                ↓
          [Idle Timeout]
```

| Event | Action |
|---|---|
| Login sukses | Create session di Redis |
| Request dengan valid token | Update `last_active_at` |
| Refresh token expire | Session expire |
| User logout | Revoke session |
| Admin revoke | Revoke session |
| Password change | Revoke semua session lain |

### 2.3 Session Data Structure

```go
type SessionData struct {
    ID                  string    `json:"id"`
    UserID              string    `json:"user_id"`
    OrgID               string    `json:"org_id"`
    ActiveBranchID      string    `json:"active_branch_id"`              // branch sedang aktif
    AssignedBranchIDs   []string  `json:"assigned_branch_ids,omitempty"`  // semua branch yang bisa diakses
    Roles               []string  `json:"roles"`                          // roles untuk active branch
    Permissions         []string  `json:"permissions,omitempty"`          // cached subset
    IPAddress           string    `json:"ip_address"`
    UserAgent           string    `json:"user_agent"`
    DeviceInfo          string    `json:"device_info"`
    CreatedAt           int64     `json:"created_at"`       // unix timestamp
    ExpiresAt           int64     `json:"expires_at"`
    LastUsedAt          int64     `json:"last_used_at"`
    MFAVerified         bool      `json:"mfa_verified"`
}
```

### 2.4 Session Renewal (Sliding Window)

Setiap request yang berhasil memperpanjang session TTL:
- Jika `remaining TTL < 30%` dari total TTL → renew
- Max session lifetime: 12 jam (hard limit)
- Absolute expiry: session expires sesuai `session_end_time` untuk role tersebut

**Banking requirement**: Session expired saat:
- Jam kerja selesai sesuai role (bukan hardcoded 16:00)
- User logout manual
- Admin force-logout
- Password berubah
- Account di-lock

### 2.5 Session Timeout by Role (Configurable via Policy7)

Session expiry time **bukan hardcoded 16:00** — melainkan **configurable** dengan hierarchy override:

```
Override priority (paling spesifik menang):
1. User-specific override  → user_id = "john"         (prioritas tertinggi)
2. Role default            → role = "accounting"       (prioritas kedua)
3. Organization default    → org_id = "bank-uuid"      (fallback terakhir)
```

**Contoh data di policy7:**

```
# Organization default (fallback)
category: operational_hours, applies_to: global, applies_to_id: null
value: { "session_start": "08:00", "session_end": "16:00", "days": ["mon","tue","wed","thu","fri"] }

# Role override — teller
category: operational_hours, applies_to: role, applies_to_id: "teller"
value: { "session_start": "08:00", "session_end": "16:00", "days": ["mon","tue","wed","thu","fri"] }

# Role override — accounting
category: operational_hours, applies_to: role, applies_to_id: "accounting"
value: { "session_start": "08:00", "session_end": "21:00", "days": ["mon","tue","wed","thu","fri","sat"] }

# Role override — branch manager (24/7)
category: operational_hours, applies_to: role, applies_to_id: "branch_manager"
value: { "session_start": "00:00", "session_end": "23:59", "days": ["*"] }

# User-specific override — audit special (lembur sampai malam)
category: operational_hours, applies_to: user, applies_to_id: "user-audit-uuid"
value: { "session_start": "00:00", "session_end": "23:59", "days": ["*"] }

# User-specific override — teller yang disuspend jam kerjanya
category: operational_hours, applies_to: user, applies_to_id: "user-maria-uuid"
value: { "session_start": "09:00", "session_end": "15:00", "days": ["mon","tue","wed","thu","fri"] }
```

**Saat login, auth7 query policy7 dengan fallback:**

```
Login flow:
1. User authenticate → success
2. Auth7 query policy7 (dengan override hierarchy):
   a. GET /v1/params/operational-hours?user_id=john&role=accounting&org_id=bank-uuid
   b. Policy7 return: value dari user-specific (jika ada), else role, else org default
3. Auth7 calculate session expires_at:
   - Jika sekarang 07:30 → expires_at = hari ini sesuai session_end
   - Jika sekarang 22:00 dan role=accounting (end=21:00) → DENY login (di luar jam kerja)
   - Jika sekarang 22:00 dan user punya override 24/7 → expires_at = 23:59
4. Set session TTL = expires_at - now
```

**Kasus khusus — diluar jam kerja:**
- Emergency access: admin bisa override via admin API `POST /admin/v1/system/emergency/extend-session`
- User override di policy7: admin set user-specific operational hours tanpa code changes
- Query ke policy7 di-cache di Redis (TTL 5 menit) agar tidak setiap request hit policy7

### 2.6 Max Concurrent Sessions

- **Configurable per org**: `organizations.settings.session_policy.max_concurrent`
- Default: 3 sessions (3 device/browser)
- Saat batas terlampaui: oldest session di-revoke (atau error, configurable per org)
- Admin dapat force-revoke sessions via admin API

### 2.6 IP Binding (Hard)

Auth7 menerapkan **hard IP binding** untuk session security (banking-grade):
- IP address disimpan saat session dibuat (dalam Redis session data)
- Setiap request, IP saat ini dibandingkan dengan IP awal
- Jika IP berubah → **session langsung di-revoke**, user harus login ulang
- Audit event dicatat: `session.ip_changed` dengan IP lama dan IP baru
- Notifikasi ke admin dashboard

**Alasan hard binding untuk banking:**
- Mencegah session hijacking — jika token dicuri, attacker dari IP berbeda langsung ditolak
- Regulasi perbankan Indonesia (OJK) mewajibkan mekanisme deteksi anomali akses
- Konsisten dengan prinsip zero-trust

**Penanganan IP change yang legitimate:**
- VPN switch / mobile network change → user harus re-login (1x login ulang, bukan blocked permanent)
- Proxy / load balancer: gunakan `X-Forwarded-For` atau `X-Real-IP` header
- Branch office WiFi → IP mungkin sama (corporate NAT)

**Perlu diperhatikan di deployment:**
- Reverse proxy (Nginx) harus forward IP asli client
- Auth7 membaca IP dari header `X-Forwarded-For` (pertama) atau `X-Real-IP`, fallback ke `RemoteAddr`
- Jika deployment di belakang load balancer, pastikan header IP di-trust hanya dari proxy internal

### 2.7 Branch Switching in Session

Saat user switch branch (via `POST /api/v1/auth/switch-branch`):
1. Validasi: `target_branch_id` ada di `assigned_branch_ids`
2. Re-authenticate: user harus masukkan password lagi (banking security)
3. Update session: `active_branch_id` = target, `roles` & `permissions` dari `user_roles` untuk branch baru
4. Issue new access token dengan claims yang di-update
5. Invalidate cached permissions (role berubah per branch)
6. Insert audit: `user.switch_branch`

SessionRedis setelah branch switch:
```
session:{id} → {
  "active_branch_id": "kcp-dago-uuid",    // berubah
  "assigned_branch_ids": ["kc-bdg", "kcp-dago", "kc-jkt"],  // tetap sama
  "roles": ["teller"],                     // berubah (dari supervisor → teller)
  "permissions": ["account:read", "transaction:create"],  // berubah
  ...
}
```

---

## 3. Token Lifecycle

### 3.1 Access Token

```
[Issued] → [Valid] → [Expired]
                ↓
          [Revoked]
```

- **TTL**: 15 menit (banking-grade)
- **Verification**: Stateless via JWKS (zero latency)
- **Revocation**: Session-bound (lihat Section 3.3)

### 3.2 Refresh Token

```
[Issued] → [Valid] → [Rotated] → [Revoked]
                ↓                    ↑
          [Expired] ─────────────────┘
```

- **TTL**: 8 jam
- **Rotation**: Ya (issue new refresh token setiap refresh)
- **Reuse detection**: Jika refresh token dipakai 2x → revoke semua session (security breach indicator)

### 3.3 Token Revocation Strategies

Karena JWT adalah stateless, revocation perlu strategi:

```
Option A: Blacklist by JTI (token ID)
  - Saat revoke: tambah jti ke Redis SET dengan TTL = sisa expiry
  - Saat verify: cek jti tidak ada di blacklist
  - Overhead: 1 Redis lookup per request


Option B: Session-bound tokens
  - Token embeds session_id
  - Saat session revoked: semua token dengan session_id tersebut invalid
  - Cek: session masih active? (Redis lookup anyway)


Decision v1.0: Option B (session-bound)
  - Lebih natural untuk banking (session = login event)
  - Single Redis lookup per request untuk session check
  - Sekaligus verify session valid + token valid
```

**Refresh Token Revocation:**
- Database update: `revoked_at = now()`
- Reuse detection: jika refresh token yang sudah dipakai di-submit lagi → revoke semua

### 3.4 Token Revocation API

```
POST /oauth2/revoke
Content-Type: application/x-www-form-urlencoded
Authorization: Basic base64(client_id:client_secret)

token=refresh_token
&token_type_hint=refresh_token
```

- Revoke refresh token → hapus dari Redis
- Revoke semua access token (via session revocation)
- RFC 7009 compliant

---

## 4. Token Refresh Flow Detail

```
Client                           auth7-svc                    Redis    PostgreSQL
  │                                    │                        │          │
  │── POST /oauth2/token               │                        │          │
  │   grant_type=refresh_token         │                        │          │
  │   refresh_token=<opaque_token>     │                        │          │
  │   ─────────────────────────────────►                        │          │
  │                                    │                        │          │
  │                                    │── SELECT refresh_token ──────────►│
  │                                    │◄── token record ─────────────────│
  │                                    │                        │          │
  │                                    │ [check: expired? revoked? reused?]│
  │                                    │                        │          │
  │                                    │── [if reuse detected]  │          │
  │                                    │   REVOKE all session tokens       │
  │                                    │   UPDATE all tokens revoked ─────►│
  │                                    │   Return 401 Unauthorized         │
  │                                    │                        │          │
  │                                    │── [if valid]           │          │
  │                                    │   generate new access_token       │
  │                                    │   generate new refresh_token      │
  │                                    │   INSERT new refresh_token ──────►│
  │                                    │   UPDATE old: used_at=now ───────►│
  │                                    │                        │          │
  │                                    │── check session still active ────►│
  │                                    │◄── session data ──────────────────│
  │                                    │                        │          │
  │◄── {access_token, refresh_token}  │                        │          │
```

---

## 5. Token Claims & Scopes Mapping

### 5.1 Core Claims (always present)
```
iss     = issuer (auth7 URL)
sub     = user_id (UUID)
aud     = resource server(s)
exp     = expiry timestamp
iat     = issued at
jti     = unique token ID
sid     = session_id (untuk revocation)
client_id = OAuth2 client yang request token
org_id  = bank/organization ID
```

### 5.2 Scope-Conditional Claims
```
scope: profile  → name, preferred_username
scope: email    → email, email_verified
scope: roles    → roles: [...]
scope: permissions → permissions: [...]  (careful: bisa besar!)
```

### 5.3 Claims di M2M (Client Credentials)
```
sub     = client_id (bukan user)
aud     = target resource servers
scopes  = allowed scopes untuk client ini
org_id  = org yang client ini milik
```

---

## 6. Cookie-Based Session (Browser)

### 6.1 Cookie Properties

```
Set-Cookie: auth7_session=session_value;
  HttpOnly;
  Secure;
  SameSite=Strict;
  Path=/;
  Domain=.bank.co.id;
  Max-Age=28800
```

| Property | Value | Alasan |
|---|---|---|
| `HttpOnly` | true | Tidak bisa diakses JavaScript |
| `Secure` | true | HTTPS only |
| `SameSite` | Strict | CSRF protection |
| `Domain` | `.bank.co.id` | Shared across subdomains |
| `Max-Age` | 28800 | 8 jam (jam kerja) |

### 6.2 No Tokens in localStorage

**TIDAK BOLEH** menyimpan token di localStorage atau sessionStorage.

---

## 7. Token Security Best Practices

### 7.1 Access Token TTL Recommendation

| Konteks | TTL |
|---|---|
| Banking web app (bos7-portal) | 15 menit |
| Internal service-to-service | 1 jam |
| Mobile app (future) | 30 menit |
| Admin operations | 15 menit |

### 7.2 Refresh Token TTL Recommendation

| Konteks | TTL |
|---|---|
| Working hours session | 8 jam (default, configurable via policy7) |
| Extended hours (accounting) | Sampai sesi berakhir (configurable per role) |
| Standard session | 8 jam |
| Remember me (future) | 30 hari |
| M2M / service account | Tidak ada refresh (re-authenticate) |

> **Catatan**: TTL refresh token di-derive dari `session_end_time` yang didapat dari policy7.
> Teller (08:00-16:00) mendapat TTL 8 jam. Accounting (08:00-21:00) mendapat TTL 13 jam.
> Admin 24/7 mendapat TTL 24 jam.

### 7.3 Security Measures

**JWT Signing:**
- Algorithm: RS256 (asymmetric) — bukan HS256
- Key size: RSA 2048-bit minimum
- Key rotation: setiap 90 hari (banking standard)

**Refresh Token Storage:**
- Hashed di database (SHA-256)
- Tidak pernah di-log
- Hanya dikirim sekali ke client, tidak bisa di-retrieve ulang

---

## 8. Concurrent Request Handling

**Problem**: Ketika access token expired, multiple concurrent requests bisa semua mencoba refresh sekaligus (thundering herd).

**Solution**: Refresh token locking
```
1. Client mencoba refresh
2. Cek: ada lock untuk refresh_token ini?
   - Ya → tunggu (poll dengan backoff) → token baru sudah ada, gunakan
   - Tidak → acquire lock (Redis SET NX, TTL 5s)
3. Do refresh
4. Release lock
5. Return token baru
```

---

## 9. Offline Access & Background Services

Untuk scenario di mana service perlu akses lama tanpa user interaction:
- Gunakan `offline_access` scope → refresh token dengan TTL panjang
- Atau: M2M Client Credentials (tidak perlu user sama sekali)

---

## 10. Session API

### 10.1 List Active Sessions

```
GET /api/v1/me/sessions

Response:
{
  "sessions": [
    {
      "id": "session-uuid",
      "client_id": "bos7-portal-prod",
      "ip_address": "192.168.1.1",
      "user_agent": "Mozilla/5.0...",
      "created_at": "2026-04-22T08:00:00Z",
      "last_active_at": "2026-04-22T10:30:00Z",
      "expires_at": "2026-04-22T16:00:00Z",
      "is_current": true
    }
  ]
}
```

### 10.2 Revoke Session

```
DELETE /api/v1/me/sessions/{session_id}
```

- Revoke session dari Redis
- Invalidate refresh token

### 10.3 Revoke All Other Sessions

```
DELETE /api/v1/me/sessions
```

- Revoke semua session kecuali current session
- Dipanggil saat change password

---

## 11. Session Timeout Handling

### 11.1 Warning Before Expire

- Warning 5 menit sebelum session expire
- User bisa extend session (jika masih dalam jam kerja)

### 11.2 Auto-Redirect on 401

- Client app redirect ke login page saat dapat 401
- Return URL disimpan untuk redirect setelah login

---

## 12. NATS Integration (v1.0)

**Part of Hybrid Messaging Model** — Redis untuk cache, NATS untuk event streaming.

### 12.1 Events Published

Auth7 publishes events ke NATS untuk konsumsi service lain:

| Event | Subject | Payload | Subscribers |
|-------|---------|---------|-------------|
| **Token Revoked** | `auth7.tokens.revoked` | `{token_id, org_id, user_id, revoked_by, reason, revoked_at}` | workflow7, core7-enterprise, notif7 |
| **Token Refreshed** | `auth7.tokens.refreshed` | `{token_id, org_id, user_id, refreshed_at}` | audit |
| **Session Created** | `auth7.sessions.created` | `{session_id, org_id, user_id, ip_address, user_agent, created_at}` | audit |
| **Session Terminated** | `auth7.sessions.terminated` | `{session_id, org_id, user_id, reason, terminated_at}` | audit, notif7 |
| **Session Revoked All** | `auth7.sessions.revoked_all` | `{org_id, revoked_by, revoked_at}` | All services |
| **Security Alert** | `auth7.security.alert` | `{type, severity, org_id, user_id, details, occurred_at}` | notif7, audit |

**Alert Types**:
- `brute_force_detected` — Multiple failed login attempts
- `new_device_login` — Login from unrecognized device
- `suspicious_activity` — Anomalous behavior detected
- `mfa_disabled` — User disabled MFA
- `password_changed` — User changed password

### 12.2 Events Subscribed

Auth7 subscribes ke events dari service lain:

| Subject | Publisher | Handler |
|---------|-----------|---------|
| `policy7.params.updated` | policy7 | Invalidate OPA cache untuk parameter yang berubah |
| `policy7.params.deleted` | policy7 | Invalidate OPA cache |

**Use Case**: Saat admin ubah operational hours di policy7, auth7 OPA perlu invalidate cache untuk pickup new value.

### 12.3 Configuration

```yaml
# configs/nats.yaml
messaging:
  nats:
    url: "${NATS_URL}"              # nats://localhost:4222
    name: "auth7"
    reconnect_wait: 2s
    max_reconnects: 10
    
    publish:
      timeout: 5s
      retry: 3                    # Retry 3x dengan exponential backoff
      
    subscribe:
      queue_group: "auth7-opa"     # Load balancing
```

### 12.4 Fail-Safe Design

**Publishing Failures**:
- ❌ **Non-blocking**: NATS failures tidak menghentikan business logic
- ✅ **Logged**: Warning log dengan correlation ID
- ✅ **Retry**: 3 attempts dengan exponential backoff
- ✅ **Source of Truth**: Database tetap authoritative

**Subscription Failures**:
- ✅ **Auto-reconnect**: Dengan exponential backoff
- ⚠️ **Graceful Degradation**: Continue tanpa real-time updates (cache stale briefly)
- ✅ **Health Check**: Endpoint `/health/nats` untuk monitoring

### 12.5 Example Flow: Token Revocation

```
Admin revoke token → Auth7
                        │
                        ├── 1. Update DB (revoke token)
                        │
                        ├── 2. Invalidate Redis cache
                        │
                        └── 3. Publish NATS event
                                Subject: auth7.tokens.revoked
                                │
                                ▼
                        ┌───────────────┐
                        │     NATS      │
                        └───────┬───────┘
                                │
            ┌───────────────────┼───────────────────┐
            │                   │                   │
            ▼                   ▼                   ▼
      ┌──────────┐       ┌──────────┐       ┌──────────┐
      │ workflow7│       │  core7   │       │  notif7  │
      │          │       │enterprise│       │          │
      │ Invalidate│       │Invalidate│       │  Notify  │
      │  cache   │       │  cache   │       │  admin   │
      └──────────┘       └──────────┘       └──────────┘
```

### 12.6 Why NATS (not Redis Pub/Sub)?

| Feature | Redis Pub/Sub | NATS |
|---------|---------------|------|
| Request-Reply | ❌ No | ✅ Yes |
| Queue Groups (load balancing) | ❌ No | ✅ Yes |
| Durable Subscriptions | ❌ No | ✅ Yes |
| Reconnection Handling | Basic | Advanced |
| Service Discovery | ❌ No | ✅ Built-in |

**Decision**: Redis untuk cache, NATS untuk service communication.

---

> Semua open questions telah dijawab di [OPEN-QUESTIONS.md](../OPEN-QUESTIONS.md).

*Prev: [04-authorization.md](./04-authorization.md) | Next: [06-mfa.md](./06-mfa.md)*
