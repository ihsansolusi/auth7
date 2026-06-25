# auth7 Makefile
# Run from supported-apps/auth7/

-include .env
export

# ─── Variabel ──────────────────────────────────────────────────────────────────
DATABASE_URL  ?= postgres://auth7:auth7secret@localhost:5432/auth7?sslmode=disable
SEED_PROFILE  ?= demo

# ─── Help ──────────────────────────────────────────────────────────────────────
.PHONY: help
help:
	@echo ""
	@echo "auth7 — perintah migration & seed"
	@echo ""
	@echo "  GENERATE MIGRATION FILES (DEF → SQL):  cd ../../appdefs/auth7 && make migrate-gen-reset"
	@echo ""
	@echo "  JALANKAN MIGRATION:"
	@echo "  make migrate-up                 Apply semua pending migrations"
	@echo "  make migrate-down               Rollback 1 migration terakhir"
	@echo "  make migrate-version            Tampilkan versi migration saat ini"
	@echo "  make migrate-force V=<version>  Force set versi (emergency recovery)"
	@echo ""
	@echo "  SEED DATA (profile-based):"
	@echo "  make seed-up [SEED_PROFILE=demo]   Apply seed: demo (users+branches) / prod (org+oauth2)"
	@echo "  make seed-down [SEED_PROFILE=demo] Rollback seed profile"
	@echo "  make seed-gen-employees            Regenerate 000006 dari ibankdb_medium"
	@echo ""
	@echo "  DEV:"
	@echo "  make db-reset                   DROP SCHEMA + migrate-up (full dev reset)"
	@echo ""

# ─── Jalankan Migration ────────────────────────────────────────────────────────
.PHONY: migrate-up
migrate-up:
	go run ./cmd/server migrate up

.PHONY: migrate-down
migrate-down:
	go run ./cmd/server migrate down

.PHONY: migrate-version
migrate-version:
	go run ./cmd/server migrate version

.PHONY: migrate-force
migrate-force:
	@[ -n "$(V)" ] || (echo "Usage: make migrate-force V=<version>"; exit 1)
	go run ./cmd/server migrate force $(V)

# ─── Seed: Profile-based (migrations-seed/) ───────────────────────────────────
.PHONY: seed-up
seed-up:
	go run ./cmd/server seed up --profile=$(SEED_PROFILE)

.PHONY: seed-down
seed-down:
	go run ./cmd/server seed down --profile=$(SEED_PROFILE)

.PHONY: seed-gen-employees
seed-gen-employees:
	python3 migrations-seed/scripts/gen_ibankdb_employees.py

# ─── Dev: Full Reset ──────────────────────────────────────────────────────────
.PHONY: db-reset
db-reset:
	@echo ""
	@echo "PERINGATAN: DROP SCHEMA public CASCADE + migrate-up."
	@echo "Semua data akan hilang. Gunakan HANYA di development."
	@echo ""
	@printf "Lanjutkan? [y/N] " && read ans && [ "$$ans" = "y" ] || (echo "Dibatalkan."; exit 0)
	psql "$(DATABASE_URL)" \
		-c "DROP SCHEMA public CASCADE; CREATE SCHEMA public; GRANT ALL ON SCHEMA public TO PUBLIC;"
	$(MAKE) migrate-up
