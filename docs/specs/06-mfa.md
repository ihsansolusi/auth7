# Auth7 — Spec 06: Multi-Factor Authentication

> **Versi**: 1.0-draft | **Tanggal**: 2026-04-22 | **Fase**: Brainstorming

---

## 1. MFA Overview

Multi-Factor Authentication menambah lapisan keamanan di luar password. Auth7 v1.0 mendukung:

| Method | RFC | Deskripsi | Status v1.0 |
|---|---|---|---|
| TOTP | RFC 6238 | Time-based OTP (Google Authenticator) | ✅ |
| Backup Codes | - | One-time recovery codes | ✅ |
| Email OTP | - | OTP via email (via auth7 internal SMTP) | ✅ |
| SMS OTP | - | OTP via SMS | 🔲 v2.0 |
| FIDO2/WebAuthn | - | Hardware keys, passkeys | 🔲 v2.0 |

---

## 2. TOTP (Time-Based One-Time Password)

### 2.1 Standar
- **RFC 6238** (TOTP) + **RFC 4226** (HOTP sebagai basis)
- Compatible dengan: Google Authenticator, Authy, Microsoft Authenticator, 1Password

### 2.2 Parameter TOTP Auth7

```go
const (
    TOTPIssuer    = "Auth7 — [OrgName]"   // muncul di authenticator app
    TOTPPeriod    = 30                     // seconds per code (standard)
    TOTPDigits    = 6                      // 6-digit codes
    TOTPAlgorithm = "SHA1"                // standard (SHA1 untuk compat)
    TOTPSkew      = 1                     // allow ±1 period (±30 detik tolerance)
)
```

### 2.3 Enrollment Flow

**Step 1: Generate TOTP Secret**
```
POST /api/v1/me/mfa/totp/setup
Authorization: Bearer <access_token>

Response:
{
  "secret": "BASE32-encoded-secret",
  "qr_code": "data:image/png;base64,...",     // QR code untuk scan
  "otpauth_url": "otpauth://totp/Auth7:john.doe?secret=...&issuer=Auth7"
}
```

TOTP secret belum aktif sampai diverifikasi.

**Step 2: Verify dan Activate**
```
POST /api/v1/me/mfa/totp/activate
Authorization: Bearer <access_token>

{
  "code": "123456"   // kode dari authenticator app
}

Response (success):
{
  "activated": true,
  "backup_codes": [
    "XXXX-XXXX-XXXX",
    "YYYY-YYYY-YYYY",
    ...
  ]  // 10 backup codes, show ONCE
}
```

**Step 3: Simpan backup codes**
- 10 backup codes, single-use
- Tampilkan SEKALI, tidak bisa di-retrieve ulang
- User wajib konfirmasi sudah menyimpan sebelum proses selesai

### 2.4 TOTP Verification (saat login)

```go
// Library: github.com/pquerna/otp/totp
valid := totp.Validate(code, user.TOTPSecret)

// Dengan skew tolerance:
valid, err := totp.ValidateCustom(code, user.TOTPSecret, time.Now(), totp.ValidateOpts{
    Period:    30,
    Skew:      1,
    Digits:    6,
    Algorithm: otp.AlgorithmSHA1,
})
```

### 2.5 TOTP Database Storage

```go
type TOTPCredential struct {
    ID          uuid.UUID
    UserID      uuid.UUID
    Secret      string    // AES-256 encrypted saat rest
    Activated   bool
    ActivatedAt *time.Time
    CreatedAt   time.Time
}
```

TOTP secret **wajib di-encrypt at rest** dengan application-level encryption (bukan DB-level saja).

### 2.6 TOTP Unenrollment

```
DELETE /api/v1/me/mfa/totp
Authorization: Bearer <access_token>

{
  "current_password": "...",  // verifikasi password sebelum unenroll
  "code": "123456"            // atau backup code
}
```

Setelah unenroll:
- TOTP secret dihapus
- Backup codes dihapus
- MFA flag = false
- Audit event: `user.mfa_disabled`

---

## 3. Email OTP

Email OTP adalah MFA factor berbasis kode 6-digit yang dikirim ke email address user.
Ini adalah **pre-login flow** — auth7 mengirim OTP langsung via internal SMTP mailer, tidak via notif7.

> **Mengapa bukan via notif7?**
> notif7 membutuhkan user JWT yang valid (authenticated session) untuk menerima event.
> Email OTP dikirim *sebelum* user punya session — hanya email address yang diketahui, bukan user_id session.
> notif7 digunakan hanya untuk post-login security alerts (lihat Section 11).

### 3.1 Email OTP Flow (Login)

```
POST /api/v1/auth/login
{ "username": "...", "password": "..." }

→ Jika user memiliki email_otp sebagai MFA method:
  Response: { "mfa_required": true, "mfa_method": "email_otp", "login_token": "..." }
  auth7 langsung trigger kirim OTP ke email user (via internal SMTP)

POST /api/v1/auth/login/mfa
{ "login_token": "...", "mfa_code": "123456" }

→ Verifikasi OTP dari Redis → create session
```

### 3.2 OTP Generation

```go
const (
    EmailOTPLength  = 6          // 6-digit numeric code
    EmailOTPTTL     = 10 * 60    // 10 menit dalam Redis
    EmailOTPMaxTry  = 5          // 5x gagal → invalidate + kirim ulang required
)

// Generate: crypto/rand → 6 digit angka (000000–999999)
// Store: Redis key "email_otp:{user_id}:{purpose}" → JSON {code, expires_at, attempts}
// purpose: "mfa_login" | "verify_email" | "recovery"
```

### 3.3 SMTP Mailer (auth7 internal)

```go
// internal/mailer/smtp_mailer.go
type Mailer interface {
    Send(ctx context.Context, to, subject, htmlBody string) error
}

// Config (env-based):
// SMTP_HOST, SMTP_PORT, SMTP_USERNAME, SMTP_PASSWORD
// SMTP_FROM = "Auth7 <noreply@bank.id>"
// SMTP_STARTTLS = true
```

Template minimal untuk email OTP:
```
Subject: Kode Verifikasi Login - 123456

Kode verifikasi Anda: 123456
Kode berlaku 10 menit. Jangan bagikan ke siapapun.
```

### 3.4 Email OTP Storage (Redis)

```
Redis key: email_otp:{user_id}:{purpose}    TTL = 10 menit
Value (JSON):
{
  "code": "123456",         // plaintext (aman karena TTL pendek + hanya di Redis)
  "expires_at": "...",
  "attempts": 0,
  "sent_to": "j***@bank.id" // untuk logging/audit (masked)
}
```

Redis keys pattern: lihat `docs/specs/08-data-model.md` Section 7 (Redis Keys).

### 3.5 Resend OTP

```
POST /api/v1/auth/login/mfa/resend
{ "login_token": "..." }
```

- Invalidate OTP lama di Redis
- Generate + kirim OTP baru
- Rate limit: max 3x resend per 10 menit

---

## 4. Backup Codes

### 4.1 Format
```
XXXXX-XXXXX    # 10 karakter alphanumeric, split by dash
Contoh: ABCD1-EF234
```

10 buah backup codes di-generate saat TOTP enrollment.

### 4.2 Storage

```go
type BackupCode struct {
    ID        uuid.UUID
    UserID    uuid.UUID
    CodeHash  string    // SHA-256 dari kode (tidak stored plaintext)
    UsedAt    *time.Time
    CreatedAt time.Time
}
```

### 4.3 Penggunaan

Backup code dapat digunakan sebagai pengganti TOTP code:
```
POST /api/v1/auth/login/mfa
{
  "login_token": "temp-token",
  "backup_code": "ABCD1-EF234"   // atau mfa_code untuk TOTP
}
```

Setelah dipakai: mark `used_at`, tidak bisa dipakai lagi.

### 4.4 Regenerate Backup Codes

Jika backup codes habis atau user ingin regenerate:
```
POST /api/v1/me/mfa/backup-codes/regenerate
Authorization: Bearer <access_token>

{
  "current_password": "...",
  "code": "123456"   // TOTP code
}

Response:
{
  "backup_codes": [...]   // 10 new codes, show ONCE
}
```

Semua backup codes lama di-invalidate.

---

## 5. MFA Policy

> **Keputusan arsitektur**: MFA policy tetap di auth7 (bukan policy7) karena erat kaitannya
> dengan identitas dan keamanan. Policy7 menyimpan operational hours & transaction limits.
> Auth7 menyimpan MFA policy karena ini authentication decision (YES/NO), bukan business parameter.

### 5.1 MFA Enforcement Levels

```go
type MFAPolicy string
const (
    MFAPolicyOptional    = "optional"      // User bisa pilih aktifkan atau tidak
    MFAPolicyRequired    = "required"      // Wajib MFA (banking high-privilege roles)
    MFAPolicyAdminForced = "admin_forced"  // Admin set wajib untuk user tertentu
)
```

### 5.2 Policy per Role (with User Override)

MFA method bisa berbeda per role, dan admin bisa override per user:

```json
// Organization default
{
  "mfa_policy": {
    "required": true,
    "methods": ["totp", "email_otp"],
    "allow_backup_codes": true,
    "enrollment_on_first_login": true
  }
}

// Role-specific defaults (stored in auth7)
Role: teller        → policy: required, default_method: email_otp
Role: supervisor    → policy: required, default_method: totp
Role: manager       → policy: required, default_method: totp
Role: org_admin     → policy: required, default_method: totp
Role: super_admin   → policy: required, default_method: totp

// User-specific override (stored in auth7)
User: john (teller) → default_method: totp (admin override: john lebih prefer TOTP)
User: maria (teller) → policy: required, default_method: email_otp (standard)
```

**Override hierarchy** (paling spesifik menang):
1. User-specific override → prioritas tertinggi
2. Role default → prioritas kedua
3. Organization default → fallback

### 5.3 MFA Enrollment Reminder

Jika MFA optional tapi belum di-set up:
- Setelah login berhasil, response includes `mfa_reminder: true`
- auth7-ui menampilkan banner/prompt untuk setup MFA
- Setelah N hari pengingat tanpa action → escalate ke `required` (configurable)

### 5.4 Per-Org Configuration

```json
{
  "mfa_policy": {
    "required": true,
    "methods": ["totp", "email_otp"],
    "allow_backup_codes": true,
    "enrollment_on_first_login": true
  }
}
```

| Setting | Default | Deskripsi |
|---|---|---|
| `required` | true | MFA wajib untuk semua user |
| `methods` | ["totp", "email_otp"] | Metode yang tersedia |
| `allow_backup_codes` | true | Backup codes diizinkan |
| `enrollment_on_first_login` | true | Wajib setup MFA saat first login |

---

## 6. MFA Verification (Login)

### 6.1 Flow

```
User                    auth7-ui                auth7-svc
  │                        │                       │
  │  1. Login berhasil     │                       │
  │     (username+pass)    │                       │
  │     tapi MFA required  │                       │
  │◄───────────────────────────────────────────────│
  │   { mfa_required: true,│                       │
  │     login_token: "..." }                       │
  │                        │                       │
  │  2. Input TOTP code    │                       │
  │     atau Email OTP     │                       │
  │───────────────────────►│                       │
  │                        │                       │
  │  3. POST /api/v1/      │                       │
  │     auth/login/mfa     │                       │
  │     {login_token,      │                       │
  │      mfa_code}         │                       │
  │───────────────────────────────────────────────►│
  │                        │                       │
  │  4. Verify MFA         │                       │
  │     + create session   │                       │
  │                        │                       │
  │  5. Return session     │                       │
  │     data               │                       │
  │◄───────────────────────────────────────────────│
```

### 6.2 Login Token

- Temporary token setelah sukses username+password
- TTL: 5 menit
- Digunakan untuk MFA verification step
- Tidak bisa dipakai untuk akses API lain

---

## 7. MFA Recovery Flow

Jika user kehilangan authenticator device:

**Step 1: Gunakan backup code**
- Login normal sampai MFA step
- Input backup code

**Step 2: Jika backup code habis/hilang**
- User harus contact admin
- Admin dapat:
  ```
  POST /admin/v1/users/:id/mfa/reset
  {
    "reason": "Lost device - verified via phone call"
  }
  ```
- Admin reset: hapus TOTP + backup codes, set `mfa_reset_required = true`
- User login berikutnya: diminta setup ulang MFA (wajib)
- Audit event: `admin.user_mfa_reset` dengan reason

---

## 8. MFA Status API

```
GET /api/v1/me/mfa
Authorization: Bearer <access_token>

Response:
{
  "enabled": true,
  "methods": [
    {
      "type": "totp",
      "activated_at": "2026-01-15T08:00:00Z"
    }
  ],
  "backup_codes_remaining": 8,
  "policy": "required"
}
```

---

## 9. Anti-Patterns & Security Notes

### 9.1 TOTP Code Reuse Prevention
```
- Track kode yang baru saja dipakai (dalam periode ini + periode sebelumnya)
- Redis: SET "totp:used:{user_id}:{code}" dengan TTL 90 detik
- Jika kode yang sama di-submit lagi dalam window → REJECT
- Mencegah replay attacks
```

### 9.2 Brute Force TOTP
```
- Maksimal 5 failed TOTP attempts → lock sementara (1 menit)
- Setelah 10 failed → force re-login (session terminated)
- TOTP attempts di-track terpisah dari password attempts
```

### 9.3 TOTP Secret Exposure
- Secret tidak pernah di-log
- QR code hanya tersedia satu kali (during enrollment)
- Endpoint setup TOTP memerlukan fresh session (recently authenticated)

---

## 10. MFA Entity

```go
type MFAConfig struct {
    UserID        uuid.UUID
    Method        MFAMethod   // totp, email_otp
    TOTPSecret    string      // encrypted at-rest
    TOTPActivated bool
    BackupCodes   []string    // hashed
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

type MFAMethod string
const (
    MFAMethodTOTP     MFAMethod = "totp"
    MFAMethodEmailOTP MFAMethod = "email_otp"
)
```

---

## 11. Security Alerts via notif7

Setelah user berhasil login dan memiliki session aktif, auth7-svc mengirim **security alert events** ke notif7
sebagai producer. Ini berbeda dari Email OTP — user sudah diketahui (`user_id` tersedia).

notif7 mendeliver alert via dua channel: **in-app bell** (SSE) + **email** (SMTP notif7).

```go
// internal/security/alert_dispatcher.go
type SecurityEvent struct {
    Type    string  // "auth.account_locked", "auth.login_new_device", dll
    UserID  string
    Email   string  // diambil dari user record
    Title   string
    Body    string
    RefURL  string
    Channels []string // ["in_app", "email"] atau ["in_app"]
}

// Fire-and-forget setelah audit event dicatat
func (d *Dispatcher) Send(ctx context.Context, e SecurityEvent) {
    go func() {
        _ = d.notif7Client.Send(context.WithoutCancel(ctx), notif7client.Event{
            Source:           "auth7",
            EventType:        e.Type,
            UserIDs:          []string{e.UserID},
            EmailAddresses:   []string{e.Email},
            DeliveryChannels: e.Channels,
            Title:            e.Title,
            Body:             e.Body,
            RefURL:           e.RefURL,
        })
    }()
}
```

**Alert matrix:**

| EventType | Trigger | DeliveryChannels |
|---|---|---|
| `auth.login_new_device` | Login dari IP / user-agent baru | `["in_app", "email"]` |
| `auth.account_locked` | 5× gagal login → lockout | `["in_app", "email"]` |
| `auth.mfa_reset` | Admin reset MFA user | `["in_app", "email"]` |
| `auth.password_changed` | Self-service password change | `["in_app"]` |

**Setup (auth7-svc):**
1. Dapatkan notif7 API key (producer JWT, issued by devops)
2. Set env: `NOTIF7_BASE_URL`, `NOTIF7_API_KEY`
3. Copy `pkg/notif7client/client.go` dari notif7 ke auth7
4. Wire `SecurityAlertDispatcher` di `cmd/` DI

**notif7 Plan 06** (`docs/plans/plan-06-email-channel.md`) mengimplementasi email channel yang diperlukan untuk ini.

---

> Semua open questions telah dijawab di [OPEN-QUESTIONS.md](../OPEN-QUESTIONS.md).

*Prev: [05-session-token.md](./05-session-token.md) | Next: [07-admin-api.md](./07-admin-api.md)*
