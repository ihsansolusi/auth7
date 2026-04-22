# Auth7 — Spec 10: Security

> **Versi**: 1.0-draft | **Tanggal**: 2026-04-22 | **Fase**: Brainstorming

---

## 1. Cryptography

### 1.1 Password Hashing

| Property | Value |
|---|---|
| Algorithm | Argon2id |
| Memory | 64 MB |
| Iterations | 3 |
| Parallelism | 4 |
| Key length | 32 bytes |
| Salt length | 16 bytes |

### 1.2 JWT Signing

| Property | Value |
|---|---|
| Algorithm | RS256 (RSA 2048-bit) |
| Key rotation | 90 hari |
| Private key | Stored encrypted at-rest |
| Public key | JWKS endpoint |

### 1.3 Encryption at Rest

| Data | Method |
|---|---|
| TOTP secret | AES-256-GCM |
| Refresh token hash | SHA-256 |
| Client secret | Argon2id (same as password) |
| Backup codes | SHA-256 (hashed) |

### 1.4 Key Encryption Key (KEK)

- v1.0: Software encryption (KEK dari env var)
- v2.0: HSM/Vault jika ada requirement regulator OJK

```env
AUTH7_ENCRYPTION_KEY=${AUTH7_ENCRYPTION_KEY}  # 256-bit key
```

---

## 2. OWASP Compliance

### 2.1 OWASP Top 10

| Risk | Mitigation |
|---|---|
| **A01: Broken Access Control** | RBAC + ABAC, Casbin enforcement |
| **A02: Cryptographic Failures** | Argon2id, RS256, AES-256-GCM |
| **A03: Injection** | Parameterized queries (pgx), input validation |
| **A04: Insecure Design** | Threat modeling, secure by default |
| **A05: Security Misconfiguration** | Hardened defaults, no secrets in config |
| **A06: Vulnerable Components** | Dependabot, regular updates |
| **A07: Auth Failures** | MFA, rate limiting, brute force protection |
| **A08: Data Integrity** | JWT signatures, HTTPS, HSTS |
| **A09: Logging Failures** | Immutable audit trail, 5 year retention |
| **A10: SSRF** | Redirect URI validation, allowlist |

### 2.2 Rate Limiting

| Endpoint | Limit | Window |
|---|---|---|
| `/auth/login` | 100 req/s | Per IP |
| `/auth/login/mfa` | 50 req/s | Per IP |
| `/auth/recover` | 3 req/hour | Per email |
| `/admin/v1/*` | 10 req/s | Per user |
| `/oauth2/token` | 100 req/s | Per client |

### 2.3 Brute Force Protection

- Max 5 gagal login berturut-turut
- Lockout: 15 menit (configurable per org)
- Counter reset setelah sukses login atau lockout expire

---

## 3. Banking Compliance

### 3.1 OJK/BI Requirements

| Requirement | Implementation |
|---|---|
| Multi-factor authentication | TOTP + Email OTP |
| Audit trail | Immutable, 5 year retention |
| Session timeout | 8 jam (jam kerja) |
| Password policy | Min 8 chars, complexity, history |
| Account lockout | After 5 failed attempts |
| Data encryption | At-rest + in-transit |
| Access control | RBAC + ABAC |

### 3.2 Penetration Testing

- **Mandatory** sebelum go-live
- Annual pentest schedule
- Third-party certified pentester

---

## 4. WAF (Web Application Firewall)

- **Level**: Infrastructure (bukan aplikasi)
- **Stack**: Nginx + ModSecurity di depan auth7-svc
- **Rules**: OWASP ModSecurity Core Rule Set (CRS)

---

## 5. Security Headers

| Header | Value |
|---|---|
| `Strict-Transport-Security` | `max-age=31536000; includeSubDomains` |
| `X-Content-Type-Options` | `nosniff` |
| `X-Frame-Options` | `DENY` |
| `X-XSS-Protection` | `1; mode=block` |
| `Referrer-Policy` | `strict-origin-when-cross-origin` |
| `Permissions-Policy` | `camera=(), microphone=(), geolocation=()` |

---

## 6. Secret Management

### 6.1 No Secrets in Config Files

```yaml
# ✅ GOOD
database:
  password: "${AUTH7_DB_PASSWORD}"

# ❌ BAD
database:
  password: "supersecret123"
```

### 6.2 Environment Variables

| Variable | Deskripsi |
|---|---|
| `AUTH7_DB_PASSWORD` | Database password |
| `AUTH7_REDIS_PASSWORD` | Redis password |
| `AUTH7_ENCRYPTION_KEY` | KEK for encryption at-rest |
| `AUTH7_JWT_PRIVATE_KEY_PATH` | Path to JWT private key |

---

## 7. Audit Trail

### 7.1 Immutable Logs

- Append-only table (no UPDATE/DELETE)
- Partitioned monthly untuk performance
- 5 year retention (sesuai regulasi perbankan)

### 7.2 Events Logged

| Event | Details |
|---|---|
| Login success/fail | IP, user agent, method |
| Logout | IP, session ID |
| MFA enroll/verify | Method, success/fail |
| Password change | User ID, admin ID (if admin-initiated) |
| Role assignment | Admin ID, user ID, role |
| Client create/update | Admin ID, client ID |
| Token revoke | Reason, user ID |
| Admin actions | All CRUD operations |

---

## 8. Open Questions

1. **Apakah perlu device fingerprinting?**
   → v1.0: Tidak
   → v1.1: Ya (untuk detect new device login)

2. **Apakah perlu CAPTCHA di login form?**
   → v1.0: Tidak (rate limiting sudah cukup)
   → v1.1: Ya (Google reCAPTCHA atau hCaptcha)

---

*Prev: [09-integration.md](./09-integration.md)*
