# Auth7 — Spec 02: Identity

> **Versi**: 1.0-draft | **Tanggal**: 2026-04-22 | **Fase**: Brainstorming

---

## 1. Konsep Dasar

### 1.1 Identity vs User vs Account

```
Identity    = siapa seseorang (attributes, credentials)
User        = representasi identity dalam sistem
Account     = akses user ke sebuah tenant/org
```

Satu `Identity` bisa memiliki beberapa `Account` di tenant berbeda (multi-org support).
Namun di v1.0, kita sederhanakan: **1 user = 1 account = 1 org/branch**.

### 1.2 Credential Types
- **Password** — username + argon2id hash (utama)
- **TOTP** — time-based one-time password (MFA)
- **Backup Codes** — recovery codes saat MFA device hilang
- **API Key** — untuk M2M / service account (v1.1)

---

## 2. User Entity

```go
type User struct {
    ID              uuid.UUID
    OrgID           uuid.UUID
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
    UserStatusCreated              UserStatus = "created"                // admin-created, belum setup password
    UserStatusPendingVerification UserStatus = "pending_verification"  // menunggu email verification
    UserStatusActive               UserStatus = "active"               // bisa login
    UserStatusInactive             UserStatus = "inactive"             // dinonaktifkan
    UserStatusLocked               UserStatus = "locked"              // brute force lockout
    UserStatusSuspended            UserStatus = "suspended"            // admin suspend
    UserStatusDeleted              UserStatus = "deleted"              // soft delete
)

// Multi-branch: user bisa akses beberapa branch dengan role/permission berbeda
type UserBranchAssignment struct {
    UserID    uuid.UUID
    BranchID  uuid.UUID
    IsPrimary bool      // true = default branch saat login (hanya 1 per user)
}

// Contoh: John punya akses ke 3 branch
// KC Bandung  (primary, role: supervisor)
// KCP Dago    (role: teller)
// KC Jakarta  (role: supervisor)
```

### 2.1 User Lifecycle States

```
[Created] ──► [Pending Verification] ──► [Active] ──► [Inactive/Suspended] ──► [Soft Deleted]
                      │                                    ▲
                      └── [Email Verified] ────────────────┘
                                                          │
                      [Locked (brute force)] ──► [Active] (admin unlock)
```

| State | Deskripsi |
|---|---|
| `Created` | User dibuat oleh admin, belum setup password |
| `Pending Verification` | Email verification token dikirim |
| `Active` | User bisa login, email verified, password setup |
| `Inactive` | User dinonaktifkan, tidak bisa login |
| `Locked` | User dikunci karena brute force, unlock oleh admin |
| `Suspended` | User disuspend oleh admin |
| `Deleted` | Soft delete, data tetap untuk audit |

### 2.2 User Attributes (extensible)
```go
type UserAttribute struct {
    UserID uuid.UUID
    Key    string   // "employee_id", "department", "position", etc.
    Value  string
}
```

---

## 3. Identity Flows

### 3.1 Registration Flow

```
POST /api/v1/auth/register
{
  "username": "john.doe",
  "email": "john.doe@bank.co.id",
  "password": "...",
  "full_name": "John Doe",
  "org_id": "uuid",
  "branch_id": "uuid"  // optional
}
```

**Steps:**
1. Validate input (format, length, policy)
2. Check uniqueness (username + email per org)
3. Validate password policy
4. Hash password dengan Argon2id
5. Create user record (status: `pending_verification`)
6. Send verification email (via auth7 internal SMTP mailer)
7. Insert audit event: `user.registered`
8. Return user info (tanpa credentials)

**Password Policy (configurable per org):**
- Min 8 karakter
- Harus ada huruf besar, kecil, angka
- Boleh ada simbol
- Tidak boleh sama dengan 5 password terakhir
- Expired setelah N hari (banking standard: 90 hari)

### 3.2 Email Verification Flow

```
POST /api/v1/auth/verify-email
{
  "token": "6-digit-code or UUID-token"
}
```

**Steps:**
1. Lookup verification token (valid? expired? used?)
2. Mark token as used
3. Update user status: `active`
4. Insert audit event: `user.email_verified`
5. Return success

**Token:** UUID + expiry 24 jam, single-use.

### 3.3 Login Flow

```
POST /api/v1/auth/login
{
  "username": "john.doe",          // atau email
  "password": "...",
  "org_id": "uuid",
  "mfa_code": "123456"             // optional, jika MFA enabled
}
```

**Steps:**
1. Find user by (username OR email) + org_id
2. Check user status (active only)
3. Check brute force (failed attempts dalam 15 menit)
4. Verify argon2id hash
5. **[If MFA enabled]** Verify TOTP code
6. **[If MFA not submitted]** Return `mfa_required` response
7. Reset failed attempts counter
8. Create session (Redis) + issue JWT
9. Insert audit event: `user.login`
10. Return access_token, refresh_token, session_id

**Brute Force Protection:**
```
- After 3 failed: 1 minute cooldown
- After 5 failed: 5 minute cooldown
- After 10 failed: account locked (manual unlock by admin)
- Counter stored in Redis dengan TTL
```

**Response (success):**
```json
{
  "access_token": "eyJ...",
  "refresh_token": "eyJ...",
  "token_type": "Bearer",
  "expires_in": 900,
  "session_id": "uuid",
  "user": {
    "id": "uuid",
    "username": "john.doe",
    "email": "john.doe@bank.co.id",
    "full_name": "John Doe",
    "mfa_enabled": true
  }
}
```

**Response (MFA required):**
```json
{
  "mfa_required": true,
  "mfa_type": "totp",
  "login_token": "temp-token-uuid"  // dipakai untuk submit MFA
}
```

### 3.4 MFA Submit Flow (saat mfa_required)

```
POST /api/v1/auth/login/mfa
{
  "login_token": "temp-token-uuid",
  "mfa_code": "123456"
}
```

**Steps:**
1. Validate `login_token` (single-use, expiry 5 menit)
2. Verify TOTP
3. Lanjutkan ke step 7 dari Login Flow
4. Insert audit: `user.mfa_verified`

### 3.5 Logout Flow

```
POST /api/v1/auth/logout
Authorization: Bearer <access_token>
```

**Steps:**
1. Extract session_id dari token
2. Revoke session di Redis
3. Revoke access_token (blacklist atau stateful expiry)
4. Insert audit: `user.logout`

### 3.6 Password Recovery Flow

**Step 1 — Request recovery:**
```
POST /api/v1/auth/recover
{
  "email": "john.doe@bank.co.id",
  "org_id": "uuid"
}
```
- Generate recovery token (UUID, expiry 1 jam)
- Kirim email via auth7 internal SMTP mailer
- Always return 200 (jangan bocorkan apakah email exist)

**Step 2 — Submit new password:**
```
PUT /api/v1/auth/recover/:token
{
  "password": "new-password",
  "password_confirm": "new-password"
}
```
- Validate token (valid, not used, not expired)
- Validate new password policy
- Check tidak sama dengan N password terakhir
- Hash dengan argon2id
- Update credential
- Invalidate semua active sessions user tersebut
- Insert audit: `user.password_reset`

### 3.7 Change Password (self-service)

```
PUT /api/v1/me/password
Authorization: Bearer <access_token>
{
  "current_password": "...",
  "new_password": "...",
  "new_password_confirm": "..."
}
```

**Steps:**
1. Verify current_password
2. Validate new password policy
3. Hash new password
4. Update credential
5. Optionally: invalidate other sessions (configurable)
6. Insert audit: `user.password_changed`

---

## 4. Self-Service Profile

### 4.1 Get Profile
```
GET /api/v1/me
Authorization: Bearer <access_token>

Response:
{
  "id": "uuid",
  "username": "john.doe",
  "email": "john.doe@bank.co.id",
  "full_name": "John Doe",
  "status": "active",
  "mfa_enabled": true,
  "org_id": "uuid",
  "branch_id": "uuid",
  "last_login_at": "2026-04-22T10:00:00Z",
  "attributes": {
    "employee_id": "EMP-001",
    "department": "IT"
  }
}
```

### 4.2 Update Profile
```
PUT /api/v1/me
{
  "full_name": "John Doe Updated"
  // username dan email tidak bisa self-service update (perlu admin)
}
```

---

## 5. Session Management

### 5.1 Session Metadata
```go
type Session struct {
    ID              uuid.UUID
    UserID          uuid.UUID
    OrgID           uuid.UUID
    ActiveBranchID  uuid.UUID   // branch yang sedang aktif saat ini
    IPAddress       string
    UserAgent       string
    DeviceInfo      string    // parsed dari User-Agent
    CreatedAt       time.Time
    ExpiresAt       time.Time
    LastUsedAt      time.Time
}
```

### 5.2 Session API
```
GET    /api/v1/me/sessions          # list sessions
DELETE /api/v1/me/sessions/:id      # revoke specific session
DELETE /api/v1/me/sessions          # revoke all sessions (except current)
```

### 5.3 Multi-Branch Access & Switching

User bisa punya akses ke beberapa branch. Satu branch sebagai `is_primary` (default saat login).
Saat user switch branch, diperlukan **re-authentication** (password atau MFA) untuk keamanan banking.

**Login flow (default branch):**
```
1. User login → JWT claim berisi: user_id, org_id, branch_id (primary)
2. Session disimpan di Redis: {user_id, active_branch_id: primary_branch_id}
3. Frontend menampilkan dropdown branch yang bisa diakses
```

**Switch branch flow:**
```
POST /api/v1/auth/switch-branch
{
  "target_branch_id": "uuid",
  "password": "..."       // re-auth required
}

Steps:
1. Validate: target_branch_id ada di user_branch_assignments
2. Verify password (re-auth)
3. Update session: active_branch_id = target_branch_id
4. Issue new JWT dengan branch_id baru (optional, bisa juga lewat session only)
5. Invalidate cached permissions (role/permission bisa beda per branch)
6. Insert audit: user.switch_branch
7. Return new access_token + updated session
```

**Rules:**
- User wajib punya minimal 1 branch assignment (primary)
- `is_primary` hanya boleh 1 per user (application-level enforcement)
- Saat user dibuat admin, wajib assign minimal 1 branch
- Role/permission bisa berbeda per branch (diatur via `user_roles` yang include `branch_id`)
- Switch branch tanpa re-auth → ditolak (HTTP 401)

**Admin API untuk branch assignments:**
```
GET    /admin/v1/users/:id/branches          # list branch assignments
POST   /admin/v1/users/:id/branches          # assign branch {branch_id, is_primary}
PUT    /admin/v1/users/:id/branches/:bid      # update assignment (set primary, dll)
DELETE /admin/v1/users/:id/branches/:bid      # revoke branch access
```

---

## 6. Admin User Management

### 6.1 Create User (Admin)
Sama dengan registration tapi:
- Tanpa email verification (admin bisa set status `active` langsung)
- Admin yang set initial password atau generate temp password

### 6.2 Lock / Unlock User
```
POST /admin/v1/users/:id/lock
POST /admin/v1/users/:id/unlock

Body: { "reason": "..." }  // wajib untuk audit
```

### 6.3 Reset User Password (Admin)
```
POST /admin/v1/users/:id/reset-password
{
  "send_email": true   // kirim reset link via email, atau
  "temp_password": "..." // set temp password langsung
}
```

---

## 7. Credential Management

### 7.1 Password Hashing

- **Algorithm**: Argon2id (bukan bcrypt)
- **Parameters**:
  - Memory: 64 MB
  - Iterations: 3
  - Parallelism: 4
  - Key length: 32 bytes
  - Salt length: 16 bytes

### 7.2 Password Policy

- Minimal 8 karakter, maksimal 128
- Harus mengandung: huruf besar, huruf kecil, angka
- Tidak boleh mengandung username atau email
- History 5 password terakhir (tidak boleh reuse)
- Expire: 90 hari (configurable per org)

---

## 8. Bulk Import (CSV)

### 8.1 CSV Format

```csv
username,email,full_name,branch_code,roles
john.doe,john@bank.co.id,John Doe,BDG,teller;supervisor
jane.smith,jane@bank.co.id,Jane Smith,JKT,org_admin
```

### 8.2 Flow

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

### 8.3 Duplicate Handling

- Duplicate username/email → skip row dengan error message
- Tidak ada auto-update existing user

---

## 9. User Impersonation (v1.1)

- Admin bisa "act as" user via RFC 8693 token exchange
- Full audit trail: siapa yang impersonate, kapan, berapa lama
- Token impersonate punya claim berbeda (`impersonated_by`)
- Tidak masuk v1.0

---

## 10. Soft Delete

- User dihapus → set `deleted_at` + status `deleted`
- Data tetap ada untuk audit trail
- Tidak ada grace period (banking standard)
- Username/email tidak bisa dipakai ulang

---

> Semua open questions telah dijawab di [OPEN-QUESTIONS.md](../OPEN-QUESTIONS.md).

*Prev: [01-architecture.md](./01-architecture.md) | Next: [03-oauth2-oidc.md](./03-oauth2-oidc.md)*
