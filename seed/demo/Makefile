# Makefile for auth7 demo seed.
# Run from this directory: `make seed-demo`
#
# DATABASE_URL must point at the auth7 Postgres target (override via env or .env.seed).

.PHONY: help seed-demo seed-demo-extract seed-demo-transform seed-demo-apply seed-clean

help:
	@echo "Targets:"
	@echo "  seed-demo-extract    Pull subset from ibankdb_medium (Oracle) into csv/"
	@echo "  seed-demo-transform  CSV (Indonesian) -> SQL (English, idempotent)"
	@echo "  seed-demo-apply      psql -f seed_*.sql against \$$DATABASE_URL"
	@echo "  seed-demo            extract + transform + apply"
	@echo "  seed-clean           TRUNCATE seeded tables in reverse FK order"

seed-demo-extract:
	bash extract.sh

seed-demo-transform:
	python3 transform.py

seed-demo-apply:
	@for f in seed_001_organization.sql seed_002_branches.sql seed_003_roles.sql seed_004_users.sql; do \
		echo "[seed] applying $$f"; \
		psql "$$DATABASE_URL" -v ON_ERROR_STOP=1 -f $$f ; \
	done

seed-demo: seed-demo-extract seed-demo-transform seed-demo-apply

seed-clean:
	psql "$$DATABASE_URL" -c "TRUNCATE user_branch_assignments, user_roles, user_credentials, users, branch_default_roles, role_permissions, branch_hierarchies, branches, branch_types, roles, permissions, organizations RESTART IDENTITY CASCADE;"
