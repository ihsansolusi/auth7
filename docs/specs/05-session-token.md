# Auth7 — Spec 05: Session & Token Lifecycle

> **Versi**: 1.0-draft | **Tanggal**: 2026-04-22 | **Fase**: Brainstorming

---

## 1. Session Management

### 1.1 Session Storage (Redis)

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

### 1.2 Session Lifecycle

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

### 1.3 Max Concurrent Sessions

- **Configurable per org**: `organizations.settings.session_policy.max_concurrent`
- Default: 3 sessions
- Jika exceed → revoke session paling lama tidak aktif

### 1.4 IP Binding

- **Soft binding**: Warning saja jika IP berubah, tidak force logout
- Mobile/VPN friendlier
- Audit log mencatat IP change event

---

## 2. Token Lifecycle

### 2.1 Access Token

```
[Issued] → [Valid] → [Expired]
                ↓
          [Revoked]
```

- **TTL**: 15 menit
- **Verification**: Stateless via JWKS (zero latency)
- **Revocation**: Tidak bisa revoke individual JWT (TTL pendek mitigasi risk)

### 2.2 Refresh Token

```
[Issued] → [Valid] → [Rotated] → [Revoked]
                ↓                    ↑
          [Expired] ─────────────────┘
```

- **TTL**: 8 jam
- **Rotation**: Ya (issue new refresh token setiap refresh)
- **Reuse detection**: Jika refresh token dipakai 2x → revoke semua session (security breach indicator)

### 2.3 Token Revocation

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

## 3. Cookie-Based Session (Browser)

### 3.1 Cookie Properties

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

### 3.2 No Tokens in localStorage

**TIDAK BOLEH** menyimpan token di localStorage atau sessionStorage.

---

## 4. Session API

### 4.1 List Active Sessions

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

### 4.2 Revoke Session

```
DELETE /api/v1/me/sessions/{session_id}
```

- Revoke session dari Redis
- Invalidate refresh token

### 4.3 Revoke All Other Sessions

```
DELETE /api/v1/me/sessions
```

- Revoke semua session kecuali current session
- Dipanggil saat change password

---

## 5. Session Timeout Handling

### 5.1 Warning Before Expire

- Warning 5 menit sebelum session expire
- User bisa extend session (jika masih dalam jam kerja)

### 5.2 Auto-Redirect on 401

- Client app redirect ke login page saat dapat 401
- Return URL disimpan untuk redirect setelah login

---

## 6. Open Questions

1. **Apakah perlu persistent session (remember me)?**
   → Banking: Tidak (session expire di akhir jam kerja)

2. **Bagaimana handling jika user membuka multiple tabs?**
   → Session shared via cookies (same domain)
   → Logout di satu tab → tab lain redirect ke login saat next request

---

*Prev: [04-authorization.md](./04-authorization.md) | Next: [06-mfa.md](./06-mfa.md)*
