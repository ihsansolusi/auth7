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

## 2. Plan 13 — Enterprise Boundary: Residual Lintas-Modul

Konformansi internal auth7 sudah **PASS** (audit W5, 2026-05-12), tetapi **cutover lintas-modul** belum selesai.
Rekomendasi kesiapan cutover terakhir: **`NOT READY`**.

> Boundary sudah dikunci: auth7 = owner IAM (identity/credential/session/role/permission/authz);
> `core7-service-enterprise` = owner branch/employee master; `policy7` = owner policy/parameter; `bos7-enterprise` = admin UI utama.
> Residual di bawah adalah pekerjaan **implementasi & wiring**, **bukan** perubahan boundary.

### 2.1 Harmonization (auth7-side, implementasi bertahap)

- **Payload response tidak seragam** — sebagian endpoint admin legacy memakai bentuk berbeda (`users` vs `data`) dibanding facade. Perlu diseragamkan.
- **Not-found mapping tidak konsisten** — beberapa handler admin legacy masih fallback `500` alih-alih `404`.
- **Parity enforcement org/branch/role scoping** — aturan sudah terkunci, tetapi enforcement parity antar seluruh endpoint admin perlu audit runtime lintas stream.

### 2.2 Compatibility artifacts (target retire/sunset)

Tidak boleh menjadi runtime authority IAM — semua keputusan allow/deny harus dari role/permission auth7.

| Artifact | Status | Target steady-state |
|---|---|---|
| `legacy_user_id` mapping | facade | user lifecycle penuh via auth7 users + audit lineage |
| `enterprise.peran` | compatibility-only | auth7 `roles` sebagai satu-satunya runtime authority |
| `enterprise.listperanuser` | compatibility-only | auth7 `user_roles` scoped org/branch |
| `enterprise.rolemenulist` | compatibility-only | permission `menu:{menu_key}:access` |
| `enterprise.usermenulist` | retire-target | role-based menu + exception policy eksplisit |
| legacy function/action map | compatibility-only | permission `{resource}:{action}` |

### 2.3 Cutover conditions (harus terpenuhi semua)

1. Semua operasi Access Management di `bos7-enterprise` baca/tulis ke `/admin/v1/facade/*` auth7.
2. Tidak ada write path aktif ke artifact legacy role/menu/function sebagai authority runtime.
3. Parity minimum role-menu-permission translation tercapai (test matrix disepakati lintas stream).
4. Setiap audit event admin dari facade punya `correlation_id` untuk trace lintas modul.

### 2.4 Blockers (owner di luar auth7 — butuh koordinator devroot)

| Blocker | Owner | Deskripsi |
|---|---|---|
| `W5-S1-B01` | `bos7-enterprise` (S5) | facade IAM path belum fully wired/verified ke endpoint auth7 |
| `W5-S1-B02` | `core7-service-enterprise` (S3) | freeze final mapping role/menu/function source belum dinyatakan selesai |
| `W5-S1-B03` | S5 + coordinator | parity test matrix legacy → permission auth7 belum disepakati |
| `W5-S1-B04` | coordinator (`core7-devroot`) | tanggal sunset compatibility endpoints + rollback rule belum ditetapkan |

**Referensi historis** (umbrella & bukti): `core7-devroot#200` (umbrella Plan 13), `auth7#114` (stream epic), `auth7#128` (W5 audit). Detail plan asli diarsipkan di `_backup/auth7-cleanup-20260625/docs/plans/`.

---

## 3. Integrasi runtime branch/employee (turunan Plan 13, belum diimplementasi)

Berada di "Out of Scope" Plan 13 — perlu wave implementasi tersendiri:

- Runtime adapter/scheduler/sync worker untuk **branch projection** dari `core7-service-enterprise` → auth7 (kontrak field final).
- Runtime sync untuk **employee reference** (employee_id, department, position, branch_code) sebagai attribute, bukan master.
- Cache/event consumer untuk **policy7** parameter context (ABAC input) tanpa memindahkan policy truth ke auth7.
- Migrasi data legacy live untuk translasi role/menu/function.
