# auth7 — Auth Gap Implementation Brief

**Session**: E2E Auth Gap — Backend Endpoints
**Date**: 2026-05-03
**Source**: `core7-devroot/docs/test/e2e-auth/v2/GAP-ANALYSIS-REMAINING-SCENARIOS.md`
**Priority**: CRITICAL — Blocking remaining E2E test scenarios

## Status: ✅ ALL TASKS COMPLETE — Ready for auth7-ui

| # | Task | Status | Evidence |
|---|------|--------|----------|
| 1 | Register MFA Routes | ✅ DONE | Routes at `auth.go:59-64`; handlers at `auth.go:399-549` |
| 2 | Change Password Endpoint | ✅ DONE | Route at `auth.go:54`; handler at `auth.go:556-640` |
| 3 | Forgot Password + Reset | ✅ DONE | Routes at `auth.go:55-56`; handlers at `auth.go:647-768` |
| 4 | Branch ID in JWT Claims | ✅ DONE | BranchID in claims (`auth.go:268`), derived from primary branch assignment |
| 5 | Switch Branch — Data Real | ✅ DONE | Real DB queries in `branch.go:211` (GetByUserID) and `branch.go:285` (GetByUserAndBranch) |
| 6 | User Roles in JWT Claims | ✅ DONE | Roles fetched from DB at `auth.go:257`, included in claims at `auth.go:267` |
| 7 | SMTP Mailer (auth7) | ✅ DONE | `internal/mailer/smtp.go` with templates (verify/reset/OTP) |
| 8 | Email Channel (notif7) | ✅ DONE | `internal/email/` package, sqlc queries, EventService dispatch |
| 9 | Integration (notif7client) | ✅ DONE | `DeliveryChannels: ["in_app","email"]` on security events |
| 10 | Mailpit E2E Test | ✅ DONE | Verification + reset emails received in Mailpit |

**Build**: `go build ./...` ✅ | `go vet ./...` ✅ | `go test ./...` ✅
**Server**: `localhost:8090/health/live` ✅ | `localhost:8090/health/ready` ✅

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
- Generate reset token (UUID) + simpan ke `verification_tokens` table dengan `TokenTypePasswordRecovery` (TTL 15m)
- **Prod mode**: kirim email dengan link reset (SMTP mailer)
- Return: `{ message: "If the email exists, a reset link has been sent" }`

2. `POST /v1/auth/reset-password`:
```
Body: { token, new_password }
```
- Validasi reset token (cek DB)
- Update credential dengan new_password hash
- Hapus reset token
- Return: `{ success: true }`

**Note**: Tidak perlu migration terpisah — menggunakan `verification_tokens` table yang sudah ada dengan `TokenType = "password_recovery"`.

---

## Task 4: Branch ID in JWT Claims

### Problem
JWT claims tidak mengandung `branch_id` — user tidak tahu cabang mana yang aktif.

### Changes
**File**: `internal/api/rest/auth.go`

Di `HandleLogin`, setelah dapat user, cari primary branch dari `UserBranchAssignment`:
```go
var branchID string
if primaryBranch, err := h.store.UserBranchAssignmentRepository.GetPrimaryByUserID(c.Request.Context(), user.ID); err == nil && primaryBranch != nil {
    branchID = primaryBranch.BranchID.String()
}

claims := jwt.Claims{
    Username: user.Username,
    Email:    user.Email,
    Roles:    roles,
    BranchID: branchID,
}
```

**Note**: BranchID diturunkan dari `UserBranchAssignment` (bukan field di User entity) — ini sesuai dengan desain multi-branch auth7.

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
roles, _ := h.store.UserRoleRepository.GetRoleCodesByUser(c.Request.Context(), user.ID)
claims := jwt.Claims{
    ...
    Roles: roles,
}
```

---

## Verification

Semua endpoint sudah di-test dan berfungsi:

```bash
# Health check
curl http://localhost:8090/health/live     # {"status":"ok"}
curl http://localhost:8090/health/ready   # {"status":"ready"}

# Change password
curl -X POST http://localhost:8090/v1/auth/change-password \
  -H "Authorization: Bearer <token>" \
  -d '{"current_password":"...","new_password":"..."}'

# Forgot password (sends email via SMTP)
curl -X POST http://localhost:8090/v1/auth/forgot-password \
  -d '{"email":"e2euser@test.com","org_id":"00000000-0000-0000-0000-000000000001"}'

# MFA setup
curl -X POST http://localhost:8090/v1/auth/mfa/setup \
  -H "Authorization: Bearer <token>" \
  -d '{"user_id":"...","method":"totp","totp_code":"123456"}'

# List branches
curl http://localhost:8090/v1/auth/branches \
  -H "Authorization: Bearer <token>"
```

## Ready for auth7-ui

Backend siap untuk E2E testing dengan auth7-ui. Semua endpoint yang dibutuhkan sudah tersedia:

| Endpoint | Method | Auth | Status |
|----------|--------|------|--------|
| `/v1/auth/register` | POST | Public | ✅ |
| `/v1/auth/login` | POST | Public | ✅ |
| `/v1/auth/verify` | POST | Public | ✅ |
| `/v1/auth/forgot-password` | POST | Public | ✅ |
| `/v1/auth/reset-password` | POST | Public | ✅ |
| `/v1/auth/change-password` | POST | Bearer | ✅ |
| `/v1/auth/mfa/setup` | POST | Bearer | ✅ |
| `/v1/auth/mfa/verify` | POST | Bearer | ✅ |
| `/v1/auth/mfa/disable` | POST | Bearer | ✅ |
| `/v1/auth/branches` | GET | Bearer | ✅ |
| `/v1/auth/branches/switch` | POST | Bearer | ✅ |
