# Auth7 — Spec 02: Identity

> **Versi**: 1.0-draft | **Tanggal**: 2026-04-22 | **Fase**: Brainstorming

---

## 1. User Lifecycle

### 1.1 States

```
[Created] → [Pending Verification] → [Active] → [Inactive/Suspended] → [Soft Deleted]
                 ↓
           [Email Verified]
```

| State | Deskripsi |
|---|---|
| `Created` | User dibuat oleh admin, belum setup password |
| `Pending Verification` | Email verification token dikirim |
| `Active` | User bisa login, email verified, password setup |
| `Inactive` | User dinonaktifkan (suspend), tidak bisa login |
| `Soft Deleted` | User dihapus (soft), data tetap untuk audit |

### 1.2 Flow: Admin-Created User

```
Admin                   auth7-svc               Email
  │                        │                       │
  │  1. POST /admin/v1/    │                       │
  │     users              │                       │
  │     {user_data}        │                       │
  │───────────────────────►│                       │
  │                        │                       │
  │  2. Create user        │                       │
  │     + generate temp    │                       │
  │     password           │                       │
  │     + send welcome     │                       │
  │     email              │                       │
  │                        │──────────────────────►│
  │                        │                       │
  │  3. Return user object │                       │
  │◄───────────────────────│                       │
```

### 1.3 Flow: First-Time Login

```
User                    auth7-ui                auth7-svc
  │                        │                       │
  │  1. Klik link dari     │                       │
  │     email              │                       │
  │───────────────────────────────────────────────►│
  │   GET /verify/         │                       │
  │   setup-password/      │                       │
  │   {token}              │                       │
  │                        │                       │
  │  2. Verify token       │                       │
  │     + redirect ke      │                       │
  │     setup password     │                       │
  │◄───────────────────────────────────────────────│
  │                        │                       │
  │  3. Input new password │                       │
  │     + confirm          │                       │
  │───────────────────────►│                       │
  │                        │                       │
  │  4. POST /api/v1/      │                       │
  │     auth/setup-password│                       │
  │     {token, password}  │                       │
  │───────────────────────────────────────────────►│
  │                        │                       │
  │  5. Update password    │                       │
  │     + mark verified    │                       │
  │     + redirect ke      │                       │
  │     MFA setup          │                       │
  │◄───────────────────────────────────────────────│
```

---

## 2. User Entity

```go
type User struct {
    ID              uuid.UUID
    OrgID           uuid.UUID
    BranchID        *uuid.UUID  // nullable (admin pusat)
    Username        string
    Email           string
    FullName        string
    PasswordHash    string      // Argon2id
    Status          UserStatus  // active, inactive, suspended, deleted
    EmailVerified   bool
    MFAEnabled      bool
    MFAMethod       MFAMethod   // totp, email_otp
    FailedAttempts  int
    LockedUntil     *time.Time
    LastLoginAt     *time.Time
    CreatedAt       time.Time
    UpdatedAt       time.Time
    DeletedAt       *time.Time  // soft delete
}

type UserStatus string
const (
    UserStatusActive      UserStatus = "active"
    UserStatusInactive    UserStatus = "inactive"
    UserStatusSuspended   UserStatus = "suspended"
    UserStatusDeleted     UserStatus = "deleted"
)
```

---

## 3. Credential Management

### 3.1 Password Hashing

- **Algorithm**: Argon2id (bukan bcrypt)
- **Parameters**:
  - Memory: 64 MB
  - Iterations: 3
  - Parallelism: 4
  - Key length: 32 bytes
  - Salt length: 16 bytes

### 3.2 Password Policy

- Minimal 8 karakter, maksimal 128
- Harus mengandung: huruf besar, huruf kecil, angka
- Tidak boleh mengandung username atau email
- History 5 password terakhir (tidak boleh reuse)
- Expire: 90 hari (configurable per org)

### 3.3 Brute Force Protection

- Max 5 gagal login berturut-turut
- Lockout: 15 menit (configurable per org)
- Reset counter setelah sukses login atau setelah lockout expire

---

## 4. Bulk Import (CSV)

### 4.1 CSV Format

```csv
username,email,full_name,branch_code,roles
john.doe,john@bank.co.id,John Doe,BDG,teller;supervisor
jane.smith,jane@bank.co.id,Jane Smith,JKT,org_admin
```

### 4.2 Flow

```
Admin                   auth7-svc
  │                        │
  │  1. POST /admin/v1/    │
  │     users/import       │
  │     (multipart/form)   │
  │───────────────────────►│
  │                        │
  │  2. Parse CSV          │
  │     + validate rows    │
  │     + check duplicates │
  │                        │
  │  3. Create users       │
  │     (transactional)    │
  │     + send emails      │
  │                        │
  │  4. Return results     │
  │     {created, failed}  │
  │◄───────────────────────│
```

### 4.3 Duplicate Handling

- Duplicate username/email → skip row dengan error message
- Tidak ada auto-update existing user

---

## 5. Self-Service

### 5.1 Change Password

```
POST /api/v1/me/password
{
  "current_password": "old_pass",
  "new_password": "new_pass"
}
```

- Verifikasi current password
- Validasi password policy
- Cek password history (tidak boleh reuse 5 terakhir)
- Update password hash
- Revoke semua session lain (security)

### 5.2 Forgot Password

```
POST /api/v1/auth/recover
{
  "email": "user@bank.co.id"
}
```

- Generate recovery token (TTL: 15 menit)
- Kirim email dengan link recovery
- Selalu return success (jangan bocorkan apakah email terdaftar)
- Rate limit: 3 request per jam per email

### 5.3 Reset Password

```
PUT /api/v1/auth/recover/{token}
{
  "new_password": "new_pass"
}
```

- Verify token (valid, belum expired)
- Validasi password policy
- Update password hash
- Invalidate token
- Revoke semua session (security)

---

## 6. User Impersonation (v1.1)

- Admin bisa "act as" user via RFC 8693 token exchange
- Full audit trail: siapa yang impersonate, kapan, berapa lama
- Token impersonate punya claim berbeda (`impersonated_by`)
- Tidak masuk v1.0

---

## 7. Soft Delete

- User dihapus → set `deleted_at` + status `deleted`
- Data tetap ada untuk audit trail
- Tidak ada grace period (banking standard)
- Username/email tidak bisa dipakai ulang

---

## 8. Open Questions

1. **Apakah perlu user profile fields tambahan (phone, address, photo)?**
   → v1.0: minimal (full_name, email, username)
   → v1.1: extended profile

2. **Apakah perlu approval workflow untuk user creation?**
   → v1.0: Tidak (langsung create)
   → v2.0: Integrasi workflow7

---

*Prev: [01-architecture.md](./01-architecture.md) | Next: [03-oauth2-oidc.md](./03-oauth2-oidc.md)*
