# Migration Guide — auth7

Panduan lengkap untuk 4 proses migration di auth7.

---

## Prasyarat

- `psql` di PATH (untuk `db-reset` dan seed)
- `python3` di PATH (untuk generate migration dari DEF)
- `go` di PATH (untuk menjalankan migration)
- File `.env` di `supported-apps/auth7/` (copy dari `.env.example` jika belum ada)

---

## Proses 1 — Generate Migration: Reset dari Awal

**Kapan digunakan:** Database belum pernah ke production. Schema berubah besar dan lebih mudah mulai dari nol daripada menulis ALTER TABLE.

**Yang terjadi:**
1. Semua file `migrations/*.sql` dihapus
2. Generator membaca `appdefs/auth7/data_model.def`
3. Dibuat 20 pasang file `up.sql` / `down.sql` baru di `supported-apps/auth7/migrations/`

```bash
# Generate file migration dari DEF (target ada di appdefs/auth7)
cd appdefs/auth7
make migrate-gen-reset

# Reset DB + apply migration (target ada di supported-apps/auth7)
cd ../../supported-apps/auth7
make db-reset      # opsional: hapus semua tabel + data
make migrate-up
```

> **Catatan:** `make migrate-gen-reset` meminta konfirmasi sebelum menghapus file.
> Jangan jalankan di server production yang sudah running.

---

## Proses 2 — Generate Migration: Perubahan Post-Production

**Kapan digunakan:** Database sudah berjalan di production. Ada perubahan skema (tambah kolom, indeks baru, tabel baru).

**Aturan penting:**
- File migration lama **tidak boleh diubah** — mereka sudah dijalankan dan dicatat di tabel `schema_migrations`
- Setiap perubahan ditulis sebagai file baru dengan nomor urut lebih besar

### 2a. Modifikasi tabel yang sudah ada

Tulis ALTER TABLE secara manual:

```bash
cd appdefs/auth7

# Buat file migration baru kosong dengan nama yang deskriptif
# (file dibuat di supported-apps/auth7/migrations/)
make migrate-gen-add NAME=add_phone_to_users
# Output:
#   File dibuat:
#     migrations/20260613000001_add_phone_to_users.up.sql
#     migrations/20260613000001_add_phone_to_users.down.sql
```

Edit `up.sql`:
```sql
-- Migration: add_phone_to_users
ALTER TABLE users ADD COLUMN phone VARCHAR(20);
CREATE INDEX idx_users_phone ON users(phone) WHERE phone IS NOT NULL;
```

Edit `down.sql`:
```sql
-- Rollback: add_phone_to_users
DROP INDEX IF EXISTS idx_users_phone;
ALTER TABLE users DROP COLUMN IF EXISTS phone;
```

Kemudian apply:
```bash
make migrate-up
```

### 2b. Tabel baru setelah production

Untuk tabel baru, gunakan generator tapi preview dulu ke `/tmp`:

```bash
# Preview tabel baru ke /tmp (dari appdefs/auth7)
cd appdefs/auth7
python3 ../scripts/gen_migrations.py \
  --def data_model.def \
  --out /tmp/preview_new_table \
  --module auth7 \
  --date $(date +%Y%m%d) \
  --start 21  # nomor setelah migration terakhir

# Review hasilnya, lalu copy file yang relevan saja ke app repo
cp /tmp/preview_new_table/20260613000021_create_new_table.*.sql \
  ../../supported-apps/auth7/migrations/

# Apply
make migrate-up
```

---

## Proses 3 — Jalankan Migration

Perintah ini **sama** untuk kedua kasus di atas.

```bash
cd supported-apps/auth7

# Apply semua pending migration
make migrate-up

# Cek versi saat ini
make migrate-version
# Output: version: 20260612000020, dirty: false

# Rollback 1 step (jika ada masalah)
make migrate-down

# Force set versi (emergency — jika state 'dirty')
make migrate-force V=20260612000019
```

> **Jika dirty=true:** Artinya migration terakhir gagal di tengah jalan.
> Perbaiki SQL-nya dulu, lalu `make migrate-force V=<versi_sebelumnya>` untuk reset state,
> kemudian `make migrate-up` lagi.

---

## Proses 4 — Seed Data: Profile-based (Demo vs Production)

Auth7 mendukung dua **profil seed** yang dijalankan terpisah dari schema migration:

| Profile | Isi | Kapan |
|---------|-----|-------|
| `demo` | Org + 70 cabang + 6 user demo + 17 oauth2 clients | Dev local, Railway staging |
| `prod` | Org (setting MFA ketat) + 17 oauth2 clients | Railway production/implementasi |

### Menjalankan seed profile

```bash
cd supported-apps/auth7

# Apply demo seed (default)
make seed-up
# atau eksplisit:
make seed-up SEED_PROFILE=demo

# Apply production seed
make seed-up SEED_PROFILE=prod

# Rollback seed (urutan terbalik)
make seed-down SEED_PROFILE=demo
```

### Di Railway

Set environment variable `SEED_PROFILE=prod` di Railway service. Entrypoint akan otomatis membaca env ini.

Contoh Railway Start Command:
```bash
./auth7 migrate up && ./auth7 seed up
```

`seed up` tanpa `--profile` otomatis membaca `SEED_PROFILE` env var, default `demo`.

### Menambah data ke profile prod

Edit atau tambah file SQL di `migrations-seed/prod/`:
- Urut berdasarkan prefix `000001_`, `000002_`, dst
- Selalu gunakan `ON CONFLICT ... DO NOTHING` atau `DO UPDATE` agar idempotent
- Tambah down file yang bersesuaian

---

## Proses 5 — Seed Pipeline ibankdb (Opsional)

Seed lebih lengkap yang diambil dari `ibankdb_medium` (Oracle lokal) — ratusan karyawan, data real BJBS.

```bash
cd supported-apps/auth7

# Full pipeline: extract dari ibankdb_medium → transform → apply ke auth7 DB
make seed-demo

# Hanya apply SQL yang sudah ada (skip extract dari Oracle)
make seed-demo-apply

# Hapus data seed (TRUNCATE semua tabel)
make seed-clean
```

> **Prasyarat:** `ibankdb_medium` harus bisa diakses dari localhost.
> Konfigurasi Oracle ada di `seed/demo/.env.seed`.

---

## Urutan Standar (Dev Reset)

```bash
make db-reset        # DROP SCHEMA + migrate-up (schema only)
make seed-up         # Seed demo: org, cabang, 6 user, oauth2 clients
```

Untuk Railway production:
```bash
./auth7 migrate up
./auth7 seed up --profile=prod
```

---

## Ringkasan Perintah

| Perintah | Kapan |
|----------|-------|
| `make migrate-gen-reset` | Schema berubah besar, DB belum production |
| `make migrate-gen-add NAME=xxx` | Tambah/ubah tabel di DB yang sudah production |
| `make migrate-up` | Apply pending schema migration |
| `make migrate-down` | Rollback 1 schema migration step |
| `make migrate-version` | Cek status migration |
| `make migrate-force V=xxx` | Emergency: reset dirty state |
| `make seed-up [SEED_PROFILE=demo]` | Apply seed profile (demo/prod) |
| `make seed-down [SEED_PROFILE=demo]` | Rollback seed profile |
| `make seed-demo` | Seed pipeline dari ibankdb_medium |
| `make seed-demo-apply` | Apply CSV yang sudah ada |
| `make seed-clean` | Kosongkan data seed |
| `make db-reset` | Full reset: DROP SCHEMA + migrate-up |

---

## File Referensi

| File | Keterangan |
|------|-----------|
| `appdefs/auth7/src/defappconfig/data_model.def` | Source of truth schema |
| `appdefs/scripts/gen_migrations.py` | Generator DEF → SQL migration (dipanggil dari `appdefs/<modul>/Makefile`) |
| `migrations/` | Schema DDL only (001-020, jangan edit yang lama) |
| `migrations-seed/demo/` | Seed profile: org, cabang, user demo, oauth2 clients |
| `migrations-seed/prod/` | Seed profile: org (setting prod) + oauth2 clients |
| `seed/demo/` | Script seed pipeline ibankdb_medium |
| `configs/config.yaml` | Konfigurasi service (termasuk DSN default) |
| `.env` | Override env vars lokal (tidak di-commit) |
