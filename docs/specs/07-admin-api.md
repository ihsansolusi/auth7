# Auth7 — Spec 07: Admin API

> **Versi**: 1.0-draft | **Tanggal**: 2026-04-22 | **Fase**: Brainstorming

---

## 1. Overview

Admin API menyediakan endpoint untuk manajemen user, role, branch, OAuth2 client, dan audit log. Hanya bisa diakses oleh user dengan role `org_admin`, `super_admin`, atau `branch_admin`.

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
GET /admin/v1/users?search=john&status=active&branch_id=uuid&role=teller&page=1&limit=20

Response:
{
  "users": [...],
  "total": 150,
  "page": 1,
  "limit": 20
}
```

### 2.2 Create User

```
POST /admin/v1/users
{
  "username": "john.doe",
  "email": "john@bank.co.id",
  "full_name": "John Doe",
  "branch_id": "branch-uuid",
  "roles": ["teller", "supervisor"],
  "mfa_required": true,
  "require_password_change": true
}
```

- Generate temporary password
- Kirim welcome email dengan link setup password
- Return user object (tanpa password)

### 2.3 Update User

```
PUT /admin/v1/users/{id}
{
  "full_name": "John D. Doe",
  "branch_id": "new-branch-uuid",
  "roles": ["supervisor"],
  "status": "active"
}
```

### 2.4 Delete User (Soft Delete)

```
DELETE /admin/v1/users/{id}
```

- Set `deleted_at` + status `deleted`
- Data tetap untuk audit trail
- Username/email tidak bisa dipakai ulang

### 2.5 Suspend User

```
POST /admin/v1/users/{id}/suspend
{
  "reason": "Security violation"
}
```

- Set status `suspended`
- Revoke semua session aktif
- User tidak bisa login

### 2.6 Bulk Import (CSV)

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

## 3. Role Management

### 3.1 List Roles

```
GET /admin/v1/roles

Response:
{
  "roles": [
    {
      "id": "uuid",
      "name": "teller",
      "description": "Bank teller role",
      "is_system": false,
      "permissions": ["account:read", "transaction:create"]
    }
  ]
}
```

### 3.2 Create Role

```
POST /admin/v1/roles
{
  "name": "loan_officer",
  "description": "Loan officer role",
  "permissions": ["loan:read", "loan:create", "loan:approve"]
}
```

### 3.3 Update Role Permissions

```
POST /admin/v1/roles/{id}/permissions
{
  "permission_ids": ["perm-uuid-1", "perm-uuid-2"]
}
```

### 3.4 Delete Role

```
DELETE /admin/v1/roles/{id}
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

### 5.1 List Clients

```
GET /admin/v1/clients

Response:
{
  "clients": [
    {
      "id": "bos7-portal-prod",
      "name": "BOS7 Portal Production",
      "client_type": "confidential",
      "redirect_uris": ["https://portal.bank.co.id/callback"],
      "allowed_scopes": ["openid", "profile", "email", "roles"],
      "status": "active"
    }
  ]
}
```

### 5.2 Create Client

```
POST /admin/v1/clients
{
  "id": "my-app-prod",
  "name": "My App Production",
  "client_type": "confidential",
  "redirect_uris": ["https://myapp.bank.co.id/callback"],
  "allowed_scopes": ["openid", "profile", "email"],
  "allowed_grant_types": ["authorization_code", "refresh_token"],
  "require_pkce": true,
  "token_format": "jwt"
}
```

### 5.3 Regenerate Client Secret

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

## 6. Audit Log

### 6.1 Query Audit Logs

```
GET /admin/v1/audit-logs?user_id=uuid&action=login&from=2026-04-01&to=2026-04-22&page=1&limit=50

Response:
{
  "logs": [
    {
      "id": "uuid",
      "user_id": "uuid",
      "action": "login",
      "ip_address": "192.168.1.1",
      "user_agent": "Mozilla/5.0...",
      "details": {"method": "password+mfa"},
      "created_at": "2026-04-22T08:00:00Z"
    }
  ],
  "total": 500,
  "page": 1,
  "limit": 50
}
```

### 6.2 Audit Log Retention

- **Retention**: 5 tahun (sesuai regulasi perbankan Indonesia)
- **Immutable**: Tidak bisa dihapus atau diubah
- **Partitioning**: Monthly partition untuk performance

---

## 7. Admin Action Audit

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

## 8. Dual Approval (v2.0)

- v1.0: Audit trail + reason wajib, tapi tidak perlu 4-eyes approval
- v2.0: Integrasi dengan workflow7 untuk approval flow (sensitive actions)

---

## 9. Open Questions

1. **Apakah perlu bulk actions di user list?**
   → v1.0: Tidak (satu per satu)
   → v1.1: Ya (bulk suspend, bulk role assignment)

2. **Apakah perlu audit trail UI?**
   → v1.0: Tidak (audit log hanya via API)
   → v1.1: Ya (audit log viewer di auth7-ui)

---

*Prev: [06-mfa.md](./06-mfa.md) | Next: [08-data-model.md](./08-data-model.md)*
