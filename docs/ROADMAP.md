# Auth7 — Roadmap (Belum Terimplementasi)

> Hanya memuat hal yang **belum** dikerjakan. Status v1.0: ✅ COMPLETE (Plan 01–12 + auth-gap).
> Pekerjaan yang sudah selesai tercatat di [`CHANGELOG.md`](./CHANGELOG.md) — Plan 13 sudah **ditutup di sisi auth7** (2026-06-26).

---

## 1. Future Scope (v1.1 / v2.0)

Diturunkan dari "Out of Scope v1.0" di [`specs/00-overview.md`](./specs/00-overview.md). Butuh keputusan produk kapan diangkat.

| Item | Target | Catatan |
|---|---|---|
| SMS-based OTP | v1.1 | Channel MFA tambahan selain TOTP + Email OTP |
| User impersonation | v1.1 | Admin login-as-user untuk support, dengan jejak audit |
| Passkeys / FIDO2 / WebAuthn | v2.0 | Passwordless / phishing-resistant auth |
| SAML 2.0 | v2.0 | Federasi enterprise IdP berbasis SAML |
| Social login (Google, GitHub, dst.) | v2.0 | OIDC upstream provider brokering |
| Identity federation (external IdP brokering) | v2.0 | Login via IdP eksternal |
| Zanzibar-style fine-grained authz | v2.0 | Relasi-based authz (ReBAC) di atas RBAC + ABAC saat ini |
| Consent screen | v2.0 | OAuth2 consent UI untuk third-party client |
| HSM untuk JWT signing key | v2.0 | Kunci RS256 di Hardware Security Module |
| Dual approval / 4-eyes | v2.0 | Via integrasi `workflow7` untuk operasi admin sensitif |

---

## 2. Integrasi runtime branch / employee / policy7 (wave "S3")

> **Satu-satunya pekerjaan auth7 yang tersisa.** Turunan Plan 13 yang sengaja ditaruh di luar scope.
> **Bukan murni in-repo** — setiap item butuh kontrak final dari modul lain, jadi perlu brainstorm/koordinasi lintas-stream dulu.

| Item | Butuh kontrak dari | Kondisi sekarang |
|---|---|---|
| Runtime **branch projection** sync | `core7-service-enterprise` (S3) — endpoint `/v1/source-contracts/branches` final & stabil | **Sebagian SUDAH ADA**: `internal/service/branchsync/poller.go` HTTP poller (M2M, 5-min, upsert 5 kolom) ter-wire di `cmd/server/start.go`, tapi **dorman** (aktif hanya bila `ENTERPRISE_SOURCE_URL`+`ENTERPRISE_CLIENT_ID` di-set). Script manual = fallback. Belum: NATS push, drift/delete handling, hierarchy/type |
| Runtime sync **employee reference** (employee_id, department, position, branch_code) sebagai attribute (bukan master) | `core7-service-enterprise` (S3) — kontrak employee reference | Belum ada (tidak dimodelkan di auth7) |
| Cache/event consumer **policy7** parameter context untuk ABAC input | `policy7` — kontrak parameter (tanpa memindahkan policy truth ke auth7) | Belum ada (OPA `opacache` ada, tapi belum ada consumer parameter policy7) |

**Boundary (tetap):** auth7 = owner IAM; branch di auth7 = **projeksi** (bukan master); employee = **reference/attribute** (bukan master); policy = milik policy7, dikonsumsi sebagai input ABAC.

**Drift handling & sinkronisasi** adalah bagian dari desain wave ini (belum diputuskan).

> Migrasi data legacy DAF role/menu/function **tidak relevan** — dikonfirmasi tidak dipakai (2026-06-26).
