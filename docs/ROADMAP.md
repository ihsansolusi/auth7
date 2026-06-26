# Auth7 — Roadmap (Belum Terimplementasi)

> Status v1.0: ✅ **COMPLETE** — Plan 01–12 + auth-gap terimplementasi (lihat [README](../README.md) & [specs/](./specs/README.md)).
> Dokumen ini hanya memuat hal yang **belum** dikerjakan: scope future v1.1/v2.0 dan residual lintas-modul Plan 13.

---

## 1. Future Scope (di luar v1.0)

Diturunkan dari "Out of Scope v1.0" di [`specs/00-overview.md`](./specs/00-overview.md).

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

## 2. Plan 13 — Enterprise Boundary: ✅ Ditutup di sisi auth7 (2026-06-26)

> Boundary dikunci: auth7 = owner IAM (identity/credential/session/role/permission/authz);
> `core7-service-enterprise` = owner branch/employee master; `policy7` = owner policy/parameter; `bos7-enterprise` = admin UI utama.

Konformansi internal auth7 **PASS** (audit W5). Seluruh residual sisi auth7 sudah **selesai** pada coordinator session 2026-06-26:
harmonisasi error (§2.1), parity scoping (§2.1), keputusan facade→legacy + facade dihapus (§2.5).
Semua blocker W5 closed/moot (§2.4). Tidak ada lagi pekerjaan Plan 13 yang menggantung di auth7 —
yang tersisa hanya **integrasi runtime branch/employee/policy7** (§3), sebuah wave baru yang butuh kontrak dari modul lain.

### 2.1 Harmonization (auth7-side)

- ✅ **Not-found mapping** — DONE (2026-06-26, commit `f811906`). Shared `respondError` di `internal/api/rest/admin/helpers.go` memetakan `ErrNotFound`/`pgx.ErrNoRows`→404, `ErrAlreadyExists`→409, `ErrPermissionDenied`→403; 41 error-site di handler legacy diretrofit.
- ✅ **Payload envelope** — DITUTUP dengan keputusan: envelope legacy (`{users}`/`{roles}`/`{branches}`/…) **diresmikan sebagai kontrak kanonik**, BUKAN dimigrasi ke bentuk facade `{success,data,meta}`. Lihat [§2.5](#25-keputusan-facade-vs-legacy-2026-06-26).
- ✅ **Parity enforcement org/branch/role scoping** — DONE (2026-06-26). Audit + fix: (F1) `GET /admin/v1/sessions` kini di-scope ke org pemanggil (sebelumnya bocor lintas-org); (F2/F4) shared `requireOrgID` menjadikan klaim JWT sebagai sumber org otoritatif + 400 konsisten untuk org invalid, diretrofit ke seluruh handler admin; (F3) middleware menolak token tanpa org-binding kecuali `super_admin`. Regular admin tidak bisa lintas-org (403 di middleware); super_admin tetap `*:*`. Tertutup unit test (`requireOrgID`, session scoping, middleware empty-org/mismatch).

### 2.2 Compatibility artifacts — ✅ RETIRED (2026-06-26)

S3 mengonfirmasi migrasi data legacy DAF IAM (`peran`/`listperanuser`/`rolemenulist`/`usermenulist`/function-map)
**tidak dipakai lagi**. Karena itu seluruh tooling jembatan migrasi (`facade/compatibility/*` + `facade/contracts/*`)
**dihapus** bersama facade. Steady-state IAM auth7 (`roles`/`user_roles`/`menu:{key}:access`/`{resource}:{action}`)
sudah jadi satu-satunya runtime authority; guard `isLegacySemanticPath` di bos7-enterprise tetap memblokir path legacy.

### 2.3 Cutover — ✅ Selesai (read/write split menggantikan facade)

Kondisi cutover lama berbasis facade sudah tidak berlaku. Steady-state final:
1. Read Access Management di `bos7-enterprise` → langsung ke legacy `/admin/v1/*` (read-only). ✅
2. Write → `workflow7` → M2M `/internal/v1/*` wf-callbacks; tidak ada write langsung ke `/admin/v1`. ✅
3. Tidak ada write path ke artifact legacy sebagai authority runtime. ✅
4. Audit event admin (wf-callback) membawa `correlation_id` = wf instance id. ✅

### 2.4 Blockers W5 — ✅ Semua closed/moot

| Blocker | Outcome |
|---|---|
| `W5-S1-B01` (wiring facade S5) | ✅ **gugur** — tidak ada migrasi ke facade |
| `W5-S1-B02` (freeze mapping S3) | ✅ **moot** — migrasi legacy role/menu/function tidak dijalankan |
| `W5-S1-B03` (parity matrix) | parity *migrasi* **moot**; parity *scoping enforcement* **DONE** (§2.1) |
| `W5-S1-B04` (sunset facade) | ✅ **done** — seluruh `/admin/v1/facade/*` dihapus |

### 2.5 Keputusan: Facade → Legacy (2026-06-26) — ✅ RATIFIED + EXECUTED

> Mengamandemen lock W2 di `specs/07-admin-api.md §1.4`.
> Proposal + log eksekusi: [`core7-devroot/docs/plans/integration/PLAN-13-FACADE-RETIREMENT-PROPOSAL.md`](../../../docs/plans/integration/PLAN-13-FACADE-RETIREMENT-PROPOSAL.md).

**Temuan:** dari 10 endpoint `/admin/v1/facade/*`, hanya 1 yang pernah dikonsumsi (shadowed pula); konsumer admin nyata = handler legacy `/admin/v1/*` (read) + `/internal/v1/*` wf-callbacks (write).

**Yang dieksekusi:**
- **A2 (migrasi CRUD ke facade) dibatalkan** — ROI anti-corruption layer rendah (satu tim/devroot); pain kontrak sudah ditutup A1 (§2.1).
- **Legacy `/admin/v1/*` = kontrak Access Management kanonik**, dan dibuat **read-only** (write endpoint yang tak dipakai dihapus; writes via workflow7 → `/internal/v1`).
- **Seluruh `facade/*` dihapus** (`access/*` redundan + `contracts/*`/`compatibility/*` migrasi-mati); `facade/access/permissions` bos7-enterprise di-repoint ke legacy.
- **wf-callbacks dipindah ke subpackage `internal/api/rest/wfcallback/`** untuk memisahkan write-path M2M dari read-API user-JWT.

---

**Referensi historis** (umbrella & bukti): `core7-devroot#200` (umbrella Plan 13), `auth7#114` (stream epic), `auth7#128` (W5 audit). Detail plan asli diarsipkan di `_backup/auth7-cleanup-20260625/docs/plans/`.

---

## 3. Integrasi runtime branch/employee/policy7 (turunan Plan 13, belum diimplementasi)

Berada di "Out of Scope" Plan 13 — **satu-satunya pekerjaan auth7 yang tersisa**, perlu wave tersendiri
dan kontrak dari modul lain (bukan murni in-repo):

- Runtime adapter/scheduler/sync worker untuk **branch projection** dari `core7-service-enterprise` → auth7 (butuh kontrak field final dari S3). Saat ini sinkronisasi masih via `scripts/sync-branches-from-enterprise.sh` (manual/script), belum runtime.
- Runtime sync untuk **employee reference** (employee_id, department, position, branch_code) sebagai attribute, bukan master (butuh kontrak employee dari S3).
- Cache/event consumer untuk **policy7** parameter context (ABAC input) tanpa memindahkan policy truth ke auth7 (butuh kontrak parameter dari policy7).

> Migrasi data legacy DAF role/menu/function **tidak lagi relevan** (dikonfirmasi tidak dipakai, 2026-06-26).
