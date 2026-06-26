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

> Turunan Plan 13 (out-of-scope). Sebagian besar sudah dikerjakan di Wave S3 (umbrella `core7-devroot#601`).
> Detail + status per-issue: [`core7-devroot/docs/plans/integration/PLAN-S3-AUTH7-ABAC-CONTROL.md`](../../../docs/plans/integration/PLAN-S3-AUTH7-ABAC-CONTROL.md).

**Status S3 (per 2026-06-27):**
- ✅ **S3.1** branch projection sync — poller enable + delete/tombstone (#157).
- ✅ **S3.2** time-based ABAC — konsumsi policy7 `operational_hours` (cache + NATS fetch-through), wired ke checker (#158).
- ✅ **S3.3** auth7 jadi **PDP**: REST `/internal/v1/authz/*` (#163) + gRPC `auth.v1.AuthCheckService` (lib7 auth7grpc contract, #167) berbagi satu decision core.
- 🟡 **S3.3d / #609** PEP — workflow7 `Auth7RBAC` di-wire ke auth7 gRPC (`AUTH7_GRPC_ADDR`). **Wiring done; live e2e PENDING** (butuh stack jalan) → prompt: [`PLAN-S3-E2E-609.md`](../../../docs/plans/integration/PLAN-S3-E2E-609.md).
- ⏸️ **S3.3b/c** Casbin enforcer (#164) + ABAC policy store (#165) — **deferred** (belum ada konsumer hidup).

**Sisa yang belum dikerjakan (butuh kontrak modul lain):**

| Item | Butuh kontrak dari | Kondisi sekarang |
|---|---|---|
| Runtime **branch projection** sync | `core7-service-enterprise` (S3) — endpoint `/v1/source-contracts/branches` final & stabil | **SUDAH ADA (S3.1, #157)**: `internal/service/branchsync/poller.go` HTTP poller (M2M, 5-min, upsert 5 kolom) ter-wire di `cmd/server/start.go`. Enable via env (`ENTERPRISE_SOURCE_URL`+`ENTERPRISE_CLIENT_ID`, lihat `.env.example` + spec 09 §4.5.4). **Delete/tombstone handling**: pass sukses-penuh men-deactivate branch yang absen dari source, dengan guard partial-fetch + empty-set. Belum: NATS push, hierarchy/type (out of scope projection) |
| Runtime sync **employee reference** (employee_id, department, position, branch_code) sebagai attribute (bukan master) | `core7-service-enterprise` (S3) — kontrak employee reference | Belum ada (tidak dimodelkan di auth7) |
| Cache/event consumer **policy7** parameter context untuk ABAC input | `policy7` — kontrak parameter (tanpa memindahkan policy truth ke auth7) | **operational_hours SUDAH (S3.2)** via `opacache` fetch-through + NATS invalidate. Parameter lain (product_access, dll) menyusul saat fitur butuh |

**Boundary (tetap):** auth7 = owner IAM; branch di auth7 = **projeksi** (bukan master); employee = **reference/attribute** (bukan master); policy = milik policy7, dikonsumsi sebagai input ABAC.

**Drift handling & sinkronisasi** adalah bagian dari desain wave ini (belum diputuskan).

> Migrasi data legacy DAF role/menu/function **tidak relevan** — dikonfirmasi tidak dipakai (2026-06-26).
