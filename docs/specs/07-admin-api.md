# Auth7 — Spec 07: Admin & Management API

> **Versi**: 1.0-draft | **Tanggal**: 2026-04-22 | **Fase**: Brainstorming

---

## 1. Admin API Overview

Admin API digunakan oleh:
- **super_admin** — full access semua org
- **org_admin** — access dalam satu org
- **branch_admin** — access terbatas dalam satu cabang

Semua admin endpoints di-prefix `/admin/v1/` dan memerlukan:
1. Valid access token dengan scope `admin:*` atau lebih spesifik
2. Role yang sesuai (enforced via RBAC)

### 1.1 Rate Limiting

- **Admin API**: 10 req/s (lebih ketat dari public API)
- **Public API**: 100 req/s

### 1.2 Access Control

- Semua admin API endpoint dilindungi oleh RBAC
- Request harus punya role admin yang sesuai
- Audit log mencatat semua admin actions

---

## 2. User Management

### 2.1 List Users

```
GET /admin/v1/users
Authorization: Bearer <token>

Query params:
  org_id=uuid         (required untuk org_admin)
  branch_id=uuid      (optional filter)
  status=active|inactive|locked|suspended
  search=john         (search by username/email/full_name)
  role=teller         (filter by role)
  page=1
  limit=20
  sort=created_at:desc

Response:
{
  "data": [
    {
      "id": "uuid",
      "username": "john.doe",
      "email": "john.doe@bank.co.id",
      "full_name": "John Doe",
      "status": "active",
      "mfa_enabled": true,
      "org_id": "uuid",
      "branch_id": "uuid",
      "roles": ["teller"],
      "last_login_at": "2026-04-22T08:00:00Z",
      "created_at": "2026-01-01T00:00:00Z"
    }
  ],
  "meta": {
    "total": 150,
    "page": 1,
    "limit": 20
  }
}
```

### 2.2 Create User

```
POST /admin/v1/users
Authorization: Bearer <token>

{
  "username": "jane.doe",
  "email": "jane.doe@bank.co.id",
  "full_name": "Jane Doe",
  "org_id": "uuid",
  "branch_id": "uuid",           // optional
  "roles": ["teller"],           // optional, tambah roles langsung
  "send_welcome_email": true,    // kirim email via auth7 internal SMTP mailer
  "temp_password": "Temp@1234",  // optional, jika tidak di-set → generated
  "require_password_change": true // user wajib ganti saat login pertama
}

Response:
{
  "user": { ... },
  "temp_password": "Temp@1234"  // hanya muncul jika generated/set
}
```

- Generate temporary password
- Kirim welcome email dengan link setup password
- Return user object (tanpa password)

### 2.3 Update User

```
PUT /admin/v1/users/:id
{
  "full_name": "Jane Doe Updated",
  "email": "jane.new@bank.co.id",  // trigger re-verification
  "branch_id": "new-branch-uuid",  // pindah cabang
  "status": "inactive"             // non-aktifkan
}
```

### 2.4 Lock / Unlock User

```
POST /admin/v1/users/:id/lock
{
  "reason": "Suspicious activity detected"  // wajib
}

POST /admin/v1/users/:id/unlock
{
  "reason": "Verified with user directly"   // wajib
}
```

Lock effects:
- Semua active sessions di-revoke
- User tidak bisa login sampai di-unlock
- Audit event: `admin.user_locked` + reason

### 2.5 Suspend User

Lebih permanen dari lock, butuh dual-approval (via workflow7 di v2.0):
```
POST /admin/v1/users/:id/suspend
{
  "reason": "...",
  "suspend_until": "2026-06-01"  // optional, auto-unsuspend
}
```

- Set status `suspended`
- Revoke semua session aktif
- User tidak bisa login

### 2.6 Delete User (Soft Delete)

```
DELETE /admin/v1/users/:id
{
  "reason": "..."   // wajib audit
}
```

Status: `deleted` — data tetap ada untuk audit trail, user tidak bisa login.
Username/email tidak bisa dipakai ulang.

### 2.7 Reset User Password

```
POST /admin/v1/users/:id/reset-password
{
  "method": "email",           // kirim recovery link via email
  "method": "temp_password",   // set temp password langsung
  "temp_password": "Temp@1234", // required jika method = temp_password
  "require_change": true       // user wajib ganti saat login
}
```

### 2.8 Impersonate User (v1.1)

```
POST /admin/v1/users/:id/impersonate
{
  "reason": "Debug: user reported issue with workflow",
  "duration_minutes": 15
}

Response:
{
  "impersonation_token": "eyJ...",   // short-lived, 15 menit max
  "audit_id": "uuid"                  // audit trail ID
}
```

Impersonation token: claims includes `act.sub = admin_user_id` (RFC 8693 token exchange).

### 2.9 Bulk Import (CSV)

```
POST /admin/v1/users/import
Content-Type: multipart/form-data

file: users.csv

Response:
{
  "created": 45,
  "failed": 3,
  "errors": [
    {"row": 12, "username": "duplicate", "error": "username already exists"},
    {"row": 25, "email": "invalid", "error": "invalid email format"},
    {"row": 30, "branch": "XYZ", "error": "branch not found"}
  ]
}
```

---

## 3. Role & Permission Management

### 3.1 Role CRUD

```
GET    /admin/v1/roles?org_id=uuid
POST   /admin/v1/roles
PUT    /admin/v1/roles/:id
DELETE /admin/v1/roles/:id
GET    /admin/v1/roles/:id/permissions
GET    /admin/v1/roles/:id/users     # users with this role
```

**Create Role:**
```
POST /admin/v1/roles
{
  "name": "senior_teller",
  "display_name": "Senior Teller",
  "description": "...",
  "org_id": "uuid",
  "inherits_from": ["teller"],    // inherit semua permissions dari teller
  "permissions": [                // tambahan permissions
    "workflow:bulk_approve"
  ]
}
```

### 3.2 Permission Management

```
GET    /admin/v1/permissions              # list all defined permissions
POST   /admin/v1/permissions              # create new permission
DELETE /admin/v1/permissions/:id          # delete (if not in use)

# Assign permissions to role
POST   /admin/v1/roles/:id/permissions
{
  "permissions": ["workflow:approve", "report:export"]
}

DELETE /admin/v1/roles/:id/permissions
{
  "permissions": ["workflow:approve"]
}
```

### 3.3 User Role Assignment

```
GET    /admin/v1/users/:id/roles
POST   /admin/v1/users/:id/roles
{
  "roles": ["teller", "supervisor"],
  "effective_from": "2026-04-22",   // optional
  "effective_until": "2026-12-31"   // optional (temporary role)
}

DELETE /admin/v1/users/:id/roles/:role_id
```

### 3.4 Delete Role

```
DELETE /admin/v1/roles/:id
```

- System roles tidak bisa dihapus
- Cek apakah role masih dipakai user

---

## 4. Branch Management

### 4.1 List Branches

```
GET /admin/v1/branches

Response:
{
  "branches": [
    {
      "id": "uuid",
      "code": "BDG",
      "name": "Bandung Branch",
      "branch_type": "cabang",
      "parent_id": null,
      "org_id": "org-uuid"
    }
  ]
}
```

### 4.2 Create Branch

```
POST /admin/v1/branches
{
  "code": "SBY",
  "name": "Surabaya Branch",
  "branch_type": "cabang",
  "parent_id": null
}
```

### 4.3 Branch Hierarchy

```
GET /admin/v1/branch-hierarchies

Response:
{
  "hierarchies": [
    {"parent_id": "kantor-pusat", "child_id": "BDG"},
    {"parent_id": "BDG", "child_id": "BDG-KPO"}
  ]
}
```

---

## 5. OAuth2 Client Management

### 5.1 Client CRUD

```
GET    /admin/v1/clients?org_id=uuid
POST   /admin/v1/clients
GET    /admin/v1/clients/:id
PUT    /admin/v1/clients/:id
DELETE /admin/v1/clients/:id
POST   /admin/v1/clients/:id/secret/rotate   # rotate client secret
```

**Create Client:**
```
POST /admin/v1/clients
{
  "name": "BOS7 Portal Production",
  "client_id": "bos7-portal-prod",    // optional, generated if not set
  "client_type": "public",            // public | confidential
  "org_id": "uuid",
  "redirect_uris": [
    "https://portal.bank.co.id/callback",
    "https://portal.bank.co.id/silent-refresh"
  ],
  "allowed_scopes": ["openid", "profile", "email", "roles", "workflow7:read"],
  "allowed_grant_types": ["authorization_code", "refresh_token"],
  "require_pkce": true,
  "token_lifetime": {
    "access_token_ttl": "15m",
    "refresh_token_ttl": "8h",
    "id_token_ttl": "15m"
  },
  "skip_consent": true               // untuk internal clients
}

Response:
{
  "client": {
    "id": "bos7-portal-prod",
    "client_type": "public",
    ...
  },
  "client_secret": "..."   // hanya untuk confidential clients, shown ONCE
}
```

### 5.2 Regenerate Client Secret

```
POST /admin/v1/clients/{id}/regenerate-secret

Response:
{
  "client_secret": "new-secret-plain-text"
}
```

- Secret lama tetap valid selama 1 jam (grace period)
- Secret baru return dalam response (hanya sekali)

---

## 6. Organization & Tenant Management

### 6.1 Organization (Bank)

```
GET    /admin/v1/organizations
POST   /admin/v1/organizations        # super_admin only
GET    /admin/v1/organizations/:id
PUT    /admin/v1/organizations/:id
DELETE /admin/v1/organizations/:id    # super_admin only (soft delete)
```

**Create Org:**
```
POST /admin/v1/organizations
{
  "code": "BJBS",
  "name": "Bank Jabar Banten Syariah",
  "domain": "bankbjbs.co.id",
  "settings": {
    "password_policy": {
      "min_length": 8,
      "require_uppercase": true,
      "require_number": true,
      "max_age_days": 90,
      "history_count": 5
    },
    "session_policy": {
      "max_concurrent_sessions": 3,
      "idle_timeout_minutes": 30,
      "absolute_timeout_hours": 8
    },
    "mfa_policy": "required_for_roles",
    "mfa_required_roles": ["supervisor", "manager", "org_admin"]
  }
}
```

### 6.2 Branch (Cabang)

```
GET    /admin/v1/organizations/:org_id/branches
POST   /admin/v1/organizations/:org_id/branches
GET    /admin/v1/branches/:id
PUT    /admin/v1/branches/:id
```

---

## 7. Session Management (Admin)

### 7.1 List All Sessions

```
GET /admin/v1/sessions?org_id=uuid&user_id=uuid&active=true

Response:
{
  "data": [
    {
      "id": "session-uuid",
      "user_id": "uuid",
      "username": "john.doe",
      "ip_address": "10.0.1.5",
      "device_info": "Chrome 120 / Windows 10",
      "created_at": "...",
      "last_used_at": "...",
      "expires_at": "..."
    }
  ]
}
```

### 7.2 Revoke Session

```
DELETE /admin/v1/sessions/:id
{
  "reason": "Security incident response"
}
```

### 7.3 Revoke All Sessions (User)

```
DELETE /admin/v1/users/:id/sessions
{
  "reason": "Password compromised, forcing re-login"
}
```

---

## 8. Audit Log

### 8.1 Query Audit Log

```
GET /admin/v1/audit-logs
Query params:
  org_id=uuid
  user_id=uuid
  event_type=user.login|user.failed_login|admin.user_locked|...
  from=2026-04-01T00:00:00Z
  to=2026-04-22T23:59:59Z
  ip=10.0.1.5
  page=1
  limit=50

Response:
{
  "data": [
    {
      "id": "uuid",
      "event_type": "user.login",
      "user_id": "uuid",
      "username": "john.doe",
      "org_id": "uuid",
      "ip_address": "10.0.1.5",
      "user_agent": "Mozilla/5.0...",
      "details": {
        "session_id": "uuid",
        "mfa_method": "totp"
      },
      "occurred_at": "2026-04-22T08:00:00Z"
    }
  ],
  "meta": { "total": 1234, "page": 1, "limit": 50 }
}
```

### 8.2 Audit Event Types

| Event | Trigger |
|---|---|
| `user.registered` | User baru dibuat |
| `user.email_verified` | Email diverifikasi |
| `user.login` | Login berhasil |
| `user.login_failed` | Login gagal |
| `user.login_mfa_failed` | MFA gagal |
| `user.logout` | Logout |
| `user.password_changed` | Ganti password (self) |
| `user.password_reset` | Reset password |
| `user.mfa_enrolled` | Daftar MFA |
| `user.mfa_disabled` | Nonaktifkan MFA |
| `user.session_revoked` | Session dicabut |
| `admin.user_created` | Admin buat user |
| `admin.user_locked` | Admin lock user |
| `admin.user_unlocked` | Admin unlock user |
| `admin.user_suspended` | Admin suspend user |
| `admin.user_mfa_reset` | Admin reset MFA user |
| `admin.role_created` | Buat role baru |
| `admin.role_assigned` | Assign role ke user |
| `admin.role_removed` | Hapus role dari user |
| `admin.permission_changed` | Ubah permission |
| `oauth2.token_issued` | Token berhasil di-issued |
| `oauth2.token_refreshed` | Token di-refresh |
| `oauth2.token_revoked` | Token dicabut |
| `oauth2.token_introspected` | Token di-introspect |
| `security.brute_force_detected` | Brute force terdeteksi |
| `security.suspicious_ip` | IP mencurigakan |
| `system.key_rotated` | Key pair di-rotate |

### 8.3 Audit Log Retention

- **Retention**: 5 tahun (sesuai regulasi perbankan Indonesia)
- **Immutable**: Tidak bisa dihapus atau diubah
- **Partitioning**: Monthly partition untuk performance

---

## 9. System Management (super_admin)

### 9.1 Key Rotation

```
POST /admin/v1/system/keys/rotate
{
  "grace_period_hours": 24   // berapa jam key lama masih di-serve di JWKS
}

Response:
{
  "old_key_id": "auth7-2026-01",
  "new_key_id": "auth7-2026-04",
  "rotation_at": "2026-04-22T10:00:00Z",
  "old_key_expires_at": "2026-04-23T10:00:00Z"
}
```

### 9.2 Health & Stats

```
GET /admin/v1/system/health
{
  "status": "healthy",
  "components": {
    "database": "healthy",
    "redis": "healthy",
    "token_signing": "healthy"
  }
}

GET /admin/v1/system/stats
{
  "active_sessions": 1250,
  "total_users": 5000,
  "tokens_issued_today": 3421,
  "failed_logins_today": 12
}
```

---

## 10. Admin Action Audit

Semua admin actions wajib dicatat:

| Action | Audit Fields |
|---|---|
| Create user | admin_id, user_id, timestamp |
| Update user | admin_id, user_id, changes, timestamp |
| Suspend user | admin_id, user_id, reason, timestamp |
| Reset MFA | admin_id, user_id, timestamp |
| Create role | admin_id, role_id, permissions, timestamp |
| Create client | admin_id, client_id, timestamp |
| Regenerate secret | admin_id, client_id, timestamp |

---

## 11. Dual Approval (v2.0)

- v1.0: Audit trail + reason wajib, tapi tidak perlu 4-eyes approval
- v2.0: Integrasi dengan workflow7 untuk approval flow (sensitive actions)

---

## 12. Open Questions

1. **Apakah admin API perlu rate limiting sendiri?**
   → Ya, lebih ketat dari public API (10 req/s untuk admin vs 100 req/s public)

2. **Apakah perlu webhook / event notification saat admin action?**
   → ✅ **KEPUTUSAN: v1.0** — auth7-svc sebagai producer ke notif7 untuk security alerts
   → Event types: `auth.account_locked`, `auth.mfa_reset`, `auth.login_new_device`, `auth.password_changed`
   → Lihat spec `06-mfa.md` Section 11 (Security Alerts via notif7)

3. **Bulk operations: import users dari CSV?**
   → v1.1: `POST /admin/v1/users/import` dengan file upload
   → Format: CSV dengan kolom username, email, full_name, branch_id, roles

4. **Admin audit log: harus real-time atau bisa async?**
   → Audit write: async (background goroutine, channel-based)
   → Audit read: real-time (SELECT dari DB)

5. **Apakah super_admin perlu MFA selalu?**
   → Ya, absolute requirement. Tidak bisa di-disable untuk super_admin.

6. **Apakah perlu bulk actions di user list?**
   → v1.0: Tidak (satu per satu)
   → v1.1: Ya (bulk suspend, bulk role assignment)

7. **Apakah perlu audit trail UI?**
   → v1.0: Tidak (audit log hanya via API)
   → v1.1: Ya (audit log viewer di auth7-ui)

---

*Prev: [06-mfa.md](./06-mfa.md) | Next: [08-data-model.md](./08-data-model.md)*
