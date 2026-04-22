# Auth7 — Open Questions Brainstorming

> **Sesi**: 2026-04-22 | **Status**: ✅ **SEMUA SUDAH DIJAWAB**
> **Sumber**: dikumpulkan dari 11 spec files di `docs/specs/`

---

## 🔴 Prioritas Tinggi — ✅ SEMUA SUDAH DIJAWAB

### 1. Repo & Struktur Proyek

**Q1.1** — Apakah auth7 menjadi repo baru `ihsansolusi/auth7` sebagai Git submodule?
> ✅ **KEPUTUSAN: Ya**
> Konsekuensi: perlu buat GitHub repo + tambah submodule ke core7-devroot

**Q1.2** — Apakah `auth7-svc` (Go backend) berada di dalam repo `ihsansolusi/auth7`, atau repo sendiri `ihsansolusi/auth7-svc`?
> ✅ **KEPUTUSAN: Langsung di root repo** `ihsansolusi/auth7`
> Tidak perlu folder `auth7-svc/` — repo ini murni untuk service Go

**Q1.3** — Apakah `auth7-ui` (yang sudah ada di `ihsansolusi/auth7-ui`) dimasukkan ke dalam repo auth7 yang baru, atau tetap repo sendiri?
> ✅ **KEPUTUSAN: Tetap repo sendiri** `ihsansolusi/auth7-ui`
> auth7-playground akan di-reset dari awal

### 2. Infrastructure

**Q2.1** — Redis: wajib di v1.0 atau optional?
> ✅ **KEPUTUSAN: Wajib v1.0**
> Diperlukan untuk session store + rate limiting

**Q2.2** — Apakah auth7 perlu PostgreSQL read replica dari awal?
> ✅ **KEPUTUSAN: Tidak untuk v1.0**
> Bisa ditambahkan v1.1 jika diperlukan

### 3. Token Design

**Q3.1** — Access token TTL: **15 menit** (banking-grade) atau **1 jam** (DX-friendly)?
> ✅ **KEPUTUSAN: 15 menit**
> Banking-grade security, lebih aman

**Q3.2** — Refresh token TTL saat jam kerja: **8 jam** atau **24 jam**?
> ✅ **KEPUTUSAN: 8 jam**
> Session expire di akhir jam kerja (16:00), sesuai banking standard

### 4. Shared Library

**Q4.1** — `lib7-auth-go`: repo sendiri (`ihsansolusi/lib7-auth-go`) atau di dalam repo auth7?
> ✅ **KEPUTUSAN: Repo sendiri**
> Konsisten dengan pola `lib7-service-go`

---

## 🟡 Prioritas Medium — ✅ SEMUA SUDAH DIJAWAB

### 5. Authorization

**Q5.1** — Casbin storage adapter: custom pgx adapter atau `casbin/gorm-adapter`?
> ✅ **KEPUTUSAN: Custom pgx adapter**
> Lebih lean, konsisten dengan stack pgx, tanpa dependency gorm

**Q5.2** — ABAC conditions: JSON-based rules (simple) atau DSL sendiri?
> ✅ **KEPUTUSAN: Hybrid JSON Rules + OPA Rego di v1.0**
> - JSON rules untuk simple policies (90% use cases) — native Go evaluator, zero overhead
> - OPA/Rego untuk complex policies (10% use cases) — time-based, multi-attribute, complex logic
> - Policy schema: `{ "type": "json" | "rego", "rule": ... }`
> - Developer bisa pilih yang sesuai kompleksitas

**Q5.3** — Sinkronisasi Casbin policy ke multiple instances: Redis pub/sub atau polling?
> ✅ **KEPUTUSAN: Redis pub/sub**
> Real-time sync via `policy:updated` channel

**Q5.4** — Apakah perlu wildcard permissions (`*`) untuk admin?
> ✅ **KEPUTUSAN: Ya**
> Casbin sudah support wildcard untuk admin super permissions

### 6. MFA

**Q6.1** — Email OTP: masuk v1.0 atau v1.1?
> ✅ **KEPUTUSAN: Masuk v1.0**
> Delivery via **auth7 internal SMTP mailer** (bukan notif7).
> Email OTP adalah pre-login MFA factor — notif7 membutuhkan user JWT yang sudah aktif,
> tidak bisa dipakai untuk kirim OTP sebelum user berhasil login.
> Lihat `06-mfa.md` Section 3 untuk implementasi internal SMTP mailer.

**Q6.2** — Apakah TOTP "trusted device" (skip MFA untuk device yang sama) diperlukan?
> ✅ **KEPUTUSAN: Tidak**
> MFA setiap login, banking-grade security

### 7. Session

**Q7.1** — Session max concurrent per user: **3** (hardcoded) atau **configurable per org**?
> ✅ **KEPUTUSAN: Configurable per org**
> Diatur di `organizations.settings.session_policy.max_concurrent`

**Q7.2** — IP binding untuk session: hard binding (force logout jika IP beda) atau soft (warn saja)?
> ✅ **KEPUTUSAN: Soft binding**
> Warning saja jika IP berubah, tidak force logout — mobile/VPN friendlier

---

## 🟢 Prioritas Rendah — ✅ SEMUA SUDAH DIJAWAB

### 8. Identity

**Q8.1** — User impersonation feature untuk admin/support?
> ✅ **KEPUTUSAN: v1.1**
> Admin bisa "act as" user dengan RFC 8693 token exchange + full audit trail

**Q8.2** — User deletion: apakah perlu masa grace period sebelum soft-delete?
> ✅ **KEPUTUSAN: Soft delete langsung**
> Banking: tidak perlu grace period

**Q8.3** — Apakah bulk import user dari CSV masuk v1.0?
> ✅ **KEPUTUSAN: Masuk v1.0**
> Diperlukan untuk initial user setup di banking

### 9. OAuth2/OIDC

**Q9.1** — Dynamic Client Registration (RFC 7591): perlu atau tidak?
> ✅ **KEPUTUSAN: Masuk v1.0**
> Support RFC 7591 dari awal

**Q9.2** — Token format: JWT saja atau perlu opaque token mode?
> ✅ **KEPUTUSAN: JWT + Opaque Token dari awal**
> - JWT untuk stateless verification (default, zero latency)
> - Opaque token untuk high-security scenarios (instant revocation)
> - Client bisa request format saat register

**Q9.3** — Consent screen: kapan perlu diimplementasikan?
> ✅ **KEPUTUSAN: v2.0**
> Internal clients auto-approve (skip consent), third-party di v2.0

### 10. Security

**Q10.1** — HSM untuk JWT private key: perlu di v1.0?
> ✅ **KEPUTUSAN: v2.0**
> v1.0: software encryption (KEK dari env var)
> v2.0: HSM/Vault jika ada requirement regulator OJK

**Q10.2** — Penetration testing: mandatory sebelum go-live?
> ✅ **KEPUTUSAN: Ya, wajib**
> Mandatory untuk sistem banking

**Q10.3** — WAF (Web Application Firewall): di level infrastructure atau aplikasi?
> ✅ **KEPUTUSAN: Infrastructure level**
> Nginx + ModSecurity di depan auth7-svc

### 11. Audit

**Q11.1** — Audit log retention: berapa tahun?
> ✅ **KEPUTUSAN: 5 tahun**
> Sesuai regulasi perbankan Indonesia

**Q11.2** — Apakah perlu webhook ke notif7 untuk critical security events?
> ✅ **KEPUTUSAN: v1.0** — auth7 sebagai producer ke notif7 (bukan webhook, tapi HTTP event)
> Event types: `auth.account_locked`, `auth.login_new_device`, `auth.mfa_reset`, `auth.password_changed`
> notif7 Plan 06 (Email Channel) harus selesai dulu sebelum auth7 onboard.
> Lihat `09-integration.md` Section 5 dan `06-mfa.md` Section 11.

### 12. Admin

**Q12.1** — Dual approval untuk sensitive admin actions (MFA reset, suspend user): perlu?
> ✅ **KEPUTUSAN: v2.0 via workflow7**
> v1.0: audit trail + reason wajib, tapi tidak perlu 4-eyes approval
> v2.0: integrasi dengan workflow7 approval flow

**Q12.2** — Apakah admin API perlu rate limiting yang lebih ketat dari public API?
> ✅ **KEPUTUSAN: Ya, lebih ketat**
> 10 req/s admin vs 100 req/s public

---

## 📋 Checklist Sebelum Plan 01

- [x] **Q1.1** dijawab → buat GitHub repo `ihsansolusi/auth7`
- [x] **Q1.2** dijawab → repo langsung service (tanpa folder auth7-svc/)
- [x] **Q1.3** dijawab → auth7-ui tetap repo sendiri (ihsansolusi/auth7-ui), playground reset
- [x] **Q2.1** dijawab → Redis wajib di docker-compose
- [x] **Q2.2** dijawab → tidak perlu read replica di v1.0
- [x] **Q3.1** dijawab → access token TTL = 15 menit
- [x] **Q3.2** dijawab → refresh token TTL = 8 jam
- [x] **Q4.1** dijawab → buat repo `ihsansolusi/lib7-auth-go` terpisah
- [x] **Q5.1** dijawab → custom pgx adapter untuk Casbin
- [x] **Q5.2** dijawab → hybrid JSON Rules + OPA Rego
- [x] **Q5.3** dijawab → Redis pub/sub untuk policy sync
- [x] **Q5.4** dijawab → wildcard permissions untuk admin
- [x] **Q6.1** dijawab → email OTP masuk v1.0 via auth7 internal SMTP mailer (bukan notif7)
- [x] **Q6.2** dijawab → tidak ada trusted device
- [x] **Q7.1** dijawab → max concurrent sessions configurable per org
- [x] **Q7.2** dijawab → soft IP binding (warn saja)
- [x] **Q8.1** dijawab → impersonation di v1.1
- [x] **Q8.2** dijawab → soft delete langsung
- [x] **Q8.3** dijawab → bulk import CSV masuk v1.0
- [x] **Q9.1** dijawab → DCR masuk v1.0
- [x] **Q9.2** dijawab → JWT + opaque token dari awal
- [x] **Q9.3** dijawab → consent screen di v2.0
- [x] **Q10.1** dijawab → HSM di v2.0
- [x] **Q10.2** dijawab → pentest wajib
- [x] **Q10.3** dijawab → WAF di infrastructure level
- [x] **Q11.1** dijawab → 5 tahun retention
- [x] **Q11.2** dijawab → security alert producer events ke notif7 v1.0 (Plan 06)
- [x] **Q12.1** dijawab → dual approval di v2.0 via workflow7
- [x] **Q12.2** dijawab → admin rate limiting lebih ketat (10 req/s)
- [x] Specs recreated (11 files, 00-10)
- [ ] Specs direview dan disetujui oleh user (1-per-1)
- [ ] GitHub Issues dibuat di Project Board Core7 v2026.1

---

## 📊 Ringkasan Keputusan v1.0 Scope

### In Scope v1.0 (Final)
| Komponen | Fitur |
|---|---|
| **Repo** | `ihsansolusi/auth7` (submodule) — service Go langsung di root |
| **UI** | `ihsansolusi/auth7-ui` (repo terpisah), playground reset dari awal |
| **Infrastructure** | PostgreSQL 16 + Redis (wajib) |
| **Identity** | Username/password, bulk import CSV, soft delete langsung |
| **Auth Flows** | Login, logout, register, recovery, email OTP (via internal SMTP) |
| **OAuth2/OIDC** | Auth code + PKCE, client credentials, DCR (RFC 7591), JWT + opaque token |
| **Token** | Access 15 menit, Refresh 8 jam |
| **MFA** | TOTP (setiap login, no trusted device), email OTP (internal SMTP) |
| **Authorization** | RBAC + ABAC (JSON + Rego hybrid), Casbin custom pgx, wildcard admin |
| **Session** | Configurable max concurrent per org, soft IP binding |
| **Admin API** | CRUD user/role/client, rate limiting 10 req/s |
| **Audit** | Immutable log, 5 tahun retention |
| **Security** | Argon2id, RS256, WAF infrastructure, pentest wajib |
| **Multi-tenant** | Org + branch isolation |

### Out of Scope v1.0 (Future)
| Komponen | Fitur | Target |
|---|---|---|
| **Identity** | User impersonation | v1.1 |
| **Security** | HSM untuk JWT key | v2.0 |
| **OAuth2** | Consent screen | v2.0 |
| **notif7 integration** | Security alert producer events (account_locked, mfa_reset, dll) | v1.0 (notif7 Plan 06) |
| **Admin** | Dual approval (4-eyes) | v2.0 via workflow7 |

---

*Diperbarui: 2026-04-22 | Semua open questions telah dijawab | Specs recreated | Lanjut review 1-per-1*
