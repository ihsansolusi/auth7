# Auth7 — Changelog

Catatan perubahan signifikan (di luar histori commit harian). Untuk pekerjaan yang
**belum** dikerjakan lihat [`ROADMAP.md`](./ROADMAP.md).

---

## 2026-06-26 — Repo cleanup + Plan 13 ditutup (sisi auth7)

Coordinator session. Branch `feat/user-wf-callbacks` di-merge ke `main` (fast-forward);
branch usang `feat/admin-session-endpoints` + `feat/user-wf-callbacks` dihapus.

### Repo & dokumentasi
- Hapus artefak proses: `CLAUDE.md`, `ARTIFACT-*.md`, `.agents/`, `.codex/`, `docs/OPEN-QUESTIONS.md`, `docs/auth-gap/`, seluruh `docs/plans/` (di-backup ke `core7-devroot/_backup/auth7-cleanup-20260625/`).
- `README.md` ditulis ulang ke format standar Core7; `docs/specs/` di-refresh (header → implemented, index akurat, link mati diperbaiki).
- Hapus `docker-compose.dev.yml` (digantikan unified infra devroot; build context-nya devroot-root-relative).

### Plan 13 — Enterprise Boundary (ditutup di sisi auth7)
- **Harmonisasi error admin** (`f811906`): shared `respondError` — `ErrNotFound`/`pgx.ErrNoRows`→404, `ErrAlreadyExists`→409, `ErrPermissionDenied`→403; 41 error-site diretrofit. Envelope sukses legacy diresmikan kanonik (bukan facade).
- **Facade → Legacy, facade dihapus** (`0af120d`, `b67c591`): `internal/api/rest/admin/facade.go` dihapus seluruhnya. Temuan: 9 dari 10 endpoint facade nol-konsumer; sisanya shadowed. S3 konfirmasi migrasi DAF IAM tidak dipakai → `compatibility/*` + `contracts/*` ikut dihapus. Legacy `/admin/v1/*` = kontrak Access Management kanonik. Lock W2 di `specs/07-admin-api.md §1.4` diamandemen. Proposal: `core7-devroot/docs/plans/integration/PLAN-13-FACADE-RETIREMENT-PROPOSAL.md`.
- **Parity scoping enforcement** (`5d4f51a`): (F1) `GET /admin/v1/sessions` di-scope ke org pemanggil (sebelumnya bocor lintas-org); (F2/F4) `requireOrgID` menjadikan klaim JWT sumber org otoritatif + 400 konsisten; (F3) middleware tolak token tanpa org-binding kecuali `super_admin`. + unit test.
- **Admin API jadi read-only** (`f9939ee`): endpoint write yang tak dipakai dihapus (user/role/branch/oauth2-client/user-role). Write via workflow7 → `/internal/v1` wf-callbacks. `DELETE /admin/v1/sessions/:id` tetap (aksi keamanan).
- **Subpackage `wfcallback`** (`1acf4d8`): wf-callbacks M2M dipindah ke `internal/api/rest/wfcallback/` (interface-injected, hindari import cycle). Memisahkan write-path M2M dari read-API user-JWT.

**Status blocker W5:** B01 gugur, B02 moot, B03 (scoping) done / (migrasi) moot, B04 done.
**Sisa auth7:** hanya integrasi runtime branch/employee/policy7 (lihat [`ROADMAP.md §2`](./ROADMAP.md)).
