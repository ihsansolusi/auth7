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
  "branch_id": "branch-uuid",
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
    ID          string    `json:"id"`
    UserID      string    `json:"user_id"`
    OrgID       string    `json:"org_id"`
    BranchID    string    `json:"branch_id,omitempty"`
    Roles       []string  `json:"roles"`
    Permissions []string  `json:"permissions,omitempty"`  // cached subset
    IPAddress   string    `json:"ip_address"`
    UserAgent   string    `json:"user_agent"`
    DeviceInfo  string    `json:"device_info"`
    CreatedAt   int64     `json:"created_at"`  // unix timestamp
    ExpiresAt   int64     `json:"expires_at"`
    LastUsedAt  int64     `json:"last_used_at"`
    MFAVerified bool      `json:"mfa_verified"`
}
```

### 2.4 Session Renewal (Sliding Window)

Setiap request yang berhasil memperpanjang session TTL:
- Jika `remaining TTL < 30%` dari total TTL → renew
- Max session lifetime: 12 jam (hard limit)
- Absolute expiry: tidak peduli ada aktivitas, session expires di jam X

**Banking requirement**: Session WAJIB expired saat:
- Jam kerja selesai (16:00)
- User logout manual
- Admin force-logout
- Password berubah
- Account di-lock

### 2.5 Max Concurrent Sessions

- **Configurable per org**: `organizations.settings.session_policy.max_concurrent`
- Default: 3 sessions (3 device/browser)
- Saat batas terlampaui: oldest session di-revoke (atau error, configurable per org)
- Admin dapat force-revoke sessions via admin API

### 2.6 IP Binding (Soft)

Auth7 menerapkan **soft IP binding** untuk session security:
- IP address disimpan saat session dibuat
- Setiap request, IP saat ini dibandingkan dengan IP awal
- Jika IP berubah drastis:
  - **Warn** dicatat di audit log
  - User **tidak** di-logout (mobile/VPN friendly)
  - Admin mendapat notifikasi di dashboard
- Hard binding (force logout) dapat dikonfigurasi per org jika diperlukan

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
| Working hours session | 8 jam |
| Standard session | 24 jam |
| Remember me (future) | 30 hari |
| M2M / service account | Tidak ada refresh (re-authenticate) |

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

## 12. Open Questions

1. **Apakah access token harus memuat roles/permissions langsung?**
   → Pro: zero-latency permission check di resource server
   → Con: stale permissions (kalau role berubah, token lama masih valid)
   → Decision: roles in token (short TTL mitigates staleness), tapi ABAC conditions tetap di-check real-time

2. **Session storage: Redis saja atau juga PostgreSQL?**
   → Redis untuk active sessions (fast lookup)
   → PostgreSQL untuk audit/history sessions (setelah expired, pindah ke audit log)

3. **Apakah perlu "remember me" feature untuk banking?**
   → Banking standard: NO. Session harus expire setelah jam kerja.
   → Tapi: mungkin perlu untuk admin panel?

4. **IP binding untuk sessions?**
   → ✅ **KEPUTUSAN: Soft binding**
   → Warn jika IP berubah drastis, bukan force logout
   → Mobile/VPN friendly, tetap maintain security audit trail

5. **Token family (Refresh Token family) approach?**
   → Satu user bisa punya N refresh token families (N = N sessions)
   → Reuse detection per family, bukan per user
   → ✅ **KEPUTUSAN: Implement ini**

6. **Max concurrent sessions: hardcoded atau configurable?**
   → ✅ **KEPUTUSAN: Configurable per org**
   → Diatur di `organizations.settings.session_policy.max_concurrent`
   → Default: 3 sessions

7. **Apakah perlu persistent session (remember me)?**
   → Banking: Tidak (session expire di akhir jam kerja)

8. **Bagaimana handling jika user membuka multiple tabs?**
   → Session shared via cookies (same domain)
   → Logout di satu tab → tab lain redirect ke login saat next request

---

*Prev: [04-authorization.md](./04-authorization.md) | Next: [06-mfa.md](./06-mfa.md)*
