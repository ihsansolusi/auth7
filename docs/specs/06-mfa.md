# Auth7 — Spec 06: Multi-Factor Authentication

> **Versi**: 1.0-draft | **Tanggal**: 2026-04-22 | **Fase**: Brainstorming

---

## 1. MFA Methods

### 1.1 TOTP (Time-based One-Time Password)

- **Algorithm**: RFC 6238 (TOTP)
- **Library**: `pquerna/otp`
- **Digits**: 6 digit
- **Period**: 30 detik
- **Compatibility**: Google Authenticator, Authy, dll

### 1.2 Email OTP

- **Delivery**: Via notif7 (email service)
- **TTL**: 5 menit
- **Digits**: 6 digit
- **Rate limit**: 3 request per jam
- **Fallback**: Jika authenticator app tidak tersedia

---

## 2. TOTP Enrollment

### 2.1 Flow

```
User                    auth7-ui                auth7-svc
  │                        │                       │
  │  1. POST /api/v1/me/   │                       │
  │     mfa/totp/setup     │                       │
  │───────────────────────────────────────────────►│
  │                        │                       │
  │  2. Generate TOTP      │                       │
  │     secret + QR code   │                       │
  │                        │                       │
  │  3. Return QR code     │                       │
  │     + secret           │                       │
  │◄───────────────────────────────────────────────│
  │                        │                       │
  │  4. Scan QR code di    │                       │
  │     authenticator app  │                       │
  │                        │                       │
  │  5. Input kode dari    │                       │
  │     authenticator app  │                       │
  │───────────────────────►│                       │
  │                        │                       │
  │  6. POST /api/v1/me/   │                       │
  │     mfa/totp/activate  │                       │
  │     {code}             │                       │
  │───────────────────────────────────────────────►│
  │                        │                       │
  │  7. Verify code        │                       │
  │     + activate TOTP    │                       │
  │     + generate backup  │                       │
  │     codes              │                       │
  │                        │                       │
  │  8. Return backup      │                       │
  │      codes             │                       │
  │◄───────────────────────────────────────────────│
```

### 2.2 Backup Codes

- 10 backup codes di-generate saat aktivasi
- Setiap backup code hanya bisa dipakai 1x
- Format: `XXXXX-XXXXX` (uppercase alphanumeric)
- User wajib konfirmasi sudah simpan backup codes
- Tidak ada "trusted device" — MFA setiap login (banking-grade)

---

## 3. MFA Verification (Login)

### 3.1 Flow

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

### 3.2 Login Token

- Temporary token setelah sukses username+password
- TTL: 5 menit
- Digunakan untuk MFA verification step
- Tidak bisa dipakai untuk akses API lain

### 3.3 Brute Force Protection

- Max 5 gagal MFA berturut-turut
- Lockout: 15 menit
- Reset counter setelah sukses MFA

---

## 4. MFA Policy

### 4.1 Per-Org Configuration

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

### 4.2 MFA Reset oleh Admin

- Admin bisa reset MFA untuk user yang kehilangan device
- Full audit trail: siapa yang reset, kapan, alasan
- User harus setup MFA ulang saat login berikutnya

---

## 5. MFA Entity

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

## 6. Open Questions

1. **Apakah perlu SMS OTP?**
   → v1.0: Tidak
   → v1.1: Mungkin (via SMS gateway)

2. **Bagaimana jika user kehilangan authenticator device DAN backup codes?**
   → Contact admin untuk manual reset
   → Admin dapat reset MFA via admin panel

---

*Prev: [05-session-token.md](./05-session-token.md) | Next: [07-admin-api.md](./07-admin-api.md)*
