# auth7 — Auth Gap Implementation Brief

**Session**: E2E Auth Gap — Backend Endpoints
**Date**: 2026-05-03
**Source**: `core7-devroot/docs/test/e2e-auth/v2/GAP-ANALYSIS-REMAINING-SCENARIOS.md`
**Priority**: CRITICAL — Blocking remaining E2E test scenarios

---

## Task 1: Register MFA Routes (quick win — handler sudah ada)

### Problem
`HandleMFASetup` handler sudah diimplementasi tapi tidak diregister di `RegisterRoutes()`. `HandleMFAVerify` dan `HandleMFADisable` belum dibuat.

### Changes
**File**: `internal/api/rest/auth.go`

1. Di `RegisterRoutes()`, tambahkan:
```go
mfa := auth.Group("/mfa")
{
    mfa.POST("/setup", h.HandleMFASetup)
    mfa.POST("/verify", h.HandleMFAVerify)
    mfa.POST("/disable", h.HandleMFADisable)
}
```

2. Tambah `HandleMFAVerify` handler:
- Input: `{ user_id, code }`
- Validasi TOTP/email code terhadap secret user
- Return: `{ verified: true, backup_codes: [...] }`

3. Tambah `HandleMFADisable` handler:
- Input: `{ user_id }`
- Reset `mfa_enabled`, `mfa_method`, `totp_secret` di user
- Return: `{ success: true }`

---

## Task 2: Change Password Endpoint

### Problem
auth7-ui punya UI change-password, client method `POST /api/v1/user/change-password`, tapi auth7 tidak punya endpoint ini.

### Changes
**File**: `internal/api/rest/auth.go`

1. Tambah di `RegisterRoutes()`:
```go
auth.POST("/change-password", h.HandleChangePassword)
```

2. Buat `HandleChangePassword`:
```
POST /v1/auth/change-password
Authorization: Bearer <access_token>
Body: { current_password, new_password }
```
- Verifikasi access token → dapat user_id
- Validasi current_password terhadap credential hash
- Hash new_password dengan argon2 → update credential
- Invalidasi semua session lain (opsional, bisa di-skip dulu)
- Return: `{ success: true }`

---

## Task 3: Forgot Password + Reset

### Problem
Recovery/reset flow butuh 2 endpoint yang belum ada.

### Changes
**File**: `internal/api/rest/auth.go`

1. `POST /v1/auth/forgot-password`:
```
Body: { email, org_id }
```
- Cari user by email
- Generate reset token (UUID) + simpan ke `password_reset_tokens` table (atau Redis dengan TTL 15m)
- **Dev mode**: return token langsung di response (skip email)
- **Prod mode**: kirim email dengan link reset
- Return: `{ message: "If the email exists, a reset link has been sent" }`

2. `POST /v1/auth/reset-password`:
```
Body: { token, new_password }
```
- Validasi reset token (cek Redis/DB)
- Update credential dengan new_password hash
- Hapus reset token
- Return: `{ success: true }`

**DB migration** (jika pakai DB untuk reset tokens):
```sql
CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    token VARCHAR(128) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

---

## Task 4: Branch ID in JWT Claims

### Problem
JWT claims tidak mengandung `branch_id` — user tidak tahu cabang mana yang aktif.

### Changes
**File**: `internal/api/rest/auth.go`

Di `HandleLogin`, setelah dapat user, cari default branch:
```go
claims := jwt.Claims{
    Username: user.Username,
    Email:    user.Email,
    BranchID: user.DefaultBranchID.String(),  // ← tambah
    Roles:    []string{},
}
```

**File**: `internal/domain/user.go` (jika belum ada field)
Tambahkan `DefaultBranchID uuid.UUID` ke User struct.

---

## Task 5: Switch Branch — Data Real

### Problem
`handleListUserBranches` return data dummy. `handleSwitchBranch` belum di-trace.

### Changes
**File**: `internal/api/rest/branch.go`

1. `handleListUserBranches`: Query user_branches dari DB untuk user yang sedang login
2. `handleSwitchBranch`: Return access token baru dengan `branch_id` yang baru di claims

---

## Task 6: User Roles in JWT Claims

### Problem
Roles selalu kosong `[]string{}`. Userrole tidak dipopulate.

### Changes
**File**: `internal/api/rest/auth.go`

Di `HandleLogin`, query roles user dari DB:
```go
roles, _ := h.store.UserRepository.GetRoles(c.Request.Context(), user.ID, orgID)
claims := jwt.Claims{
    ...
    Roles: roles,
}
```

---

## Verification

Setelah semua task selesai, jalankan:
```bash
cd /home/galih/Works/projects/banks/core7-devroot/supported-apps/auth7
go build -o auth7-bin ./cmd/server/
./auth7-bin start
```

Test endpoints:
```bash
# Change password
curl -X POST http://localhost:8090/v1/auth/change-password \
  -H "Authorization: Bearer <token>" \
  -d '{"current_password":"...","new_password":"..."}'

# Forgot password
curl -X POST http://localhost:8090/v1/auth/forgot-password \
  -d '{"email":"e2euser@test.com","org_id":"00000000-0000-0000-0000-000000000001"}'

# MFA setup
curl -X POST http://localhost:8090/v1/auth/mfa/setup \
  -H "Authorization: Bearer <token>" \
  -d '{"user_id":"...","method":"totp","totp_code":"123456"}'

# List branches
curl http://localhost:8090/auth/branches \
  -H "Authorization: Bearer <token>"
```
