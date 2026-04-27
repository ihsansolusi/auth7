# Auth7 — Plan 11: Service Migration & Integration Rollout

> **Status**: 📋 Planned  
> **Prerequisite**: Plan 01-10 selesai (auth7 fully functional)  
> **Total Issues**: 12

---

## Goal

Setelah auth7 v1.0 selesai diimplementasi, lakukan migrasi security untuk services yang sudah ada (service7-template, workflow7, notif7) dan update bos7-template dengan pola security baru.

---

## Background

Berdasarkan [Internal Service Security Analysis](../../docs/security/INTERNAL-SERVICE-SECURITY.md), services yang sudah ada menggunakan ad-hoc security:
- **service7-template**: Tidak ada M2M auth, hanya JWT validation
- **workflow7**: X-Service-Key (static) + Casbin lokal
- **notif7**: Producer JWT (HS256) dengan static secret

Plan ini memastikan semua services bermigrasi ke arsitektur security yang standardized via auth7.

---

## Migration Roadmap

### Phase 1: lib7-auth-go Enhancement (Plan 10 Extension)
Tambahan fitur untuk lib7-auth-go:

| # | Issue | Description | Target |
|---|-------|-------------|--------|
| 11.1 | lib7-auth-go: Token Exchange Client (RFC 8693) | Implementasi on-behalf-of token exchange untuk BFF pattern | lib7-auth-go |
| 11.2 | lib7-auth-go: M2M Token Cache dengan Refresh | Token manager dengan auto-refresh sebelum expiry | lib7-auth-go |

### Phase 2: service7-template Update
Update template dengan pola security baru:

| # | Issue | Description | Target |
|---|-------|-------------|--------|
| 11.3 | service7-template: Add auth7 M2M example | Contoh integrasi client credentials grant | service7-template |
| 11.4 | service7-template: Add BFF token exchange example | Contoh BFF → Backend dengan token exchange | service7-template |
| 11.5 | service7-template: Update specs & guidance | Spec 05-api-design.md, guidance/security.md | service7-template |

### Phase 3: workflow7 Migration
Migrasi workflow7 dari X-Service-Key ke auth7:

| # | Issue | Description | Target |
|---|-------|-------------|--------|
| 11.6 | workflow7: Replace X-Service-Key dengan auth7 M2M | Update notif7 client & external callers | workflow7 |
| 11.7 | workflow7: Integrate centralized RBAC | Ganti Casbin lokal dengan auth7 gRPC AuthCheck | workflow7 |
| 11.8 | workflow7: Update middleware stack | Tambah token exchange support untuk BFF calls | workflow7 |

### Phase 4: notif7 Migration
Migrasi notif7 dari static JWT ke auth7:

| # | Issue | Description | Target |
|---|-------|-------------|--------|
| 11.9 | notif7: Replace producer JWT dengan auth7 M2M | Update producer auth middleware | notif7 |
| 11.10 | notif7: Add token exchange support untuk BFF | Enable bos7-portal dll query notifikasi via delegated token | notif7 |

### Phase 5: bos7-template Security Integration
Update bos7-template dengan pola BFF security:

| # | Issue | Description | Target |
|---|-------|-------------|--------|
| 11.11 | bos7-template: Implement BFF service account | OAuth2 client credentials untuk bos7-portal | bos7-template |
| 11.12 | bos7-template: Add token exchange middleware | Middleware untuk exchange user JWT ke delegated token | bos7-template |

---

## Migration Checklist per Service

### service7-template
- [ ] Update go.mod untuk lib7-auth-go terbaru
- [ ] Add M2M token manager example
- [ ] Add token exchange example
- [ ] Update API design spec dengan security patterns
- [ ] Update guidance/general.md dengan security section

### workflow7
- [ ] Register workflow7-svc sebagai OAuth2 client di auth7
- [ ] Replace `X-Service-Key` dengan `Authorization: Bearer <M2M-token>`
- [ ] Update notif7 dispatcher untuk menggunakan auth7 M2M
- [ ] Migrate Casbin policies ke auth7 (atau tetap lokal dengan sync)
- [ ] Add support untuk delegated token (BFF calls)
- [ ] Update middleware untuk verify via JWKS (stateless)

### notif7
- [ ] Register notif7-svc sebagai OAuth2 client di auth7
- [ ] Replace producer JWT validation dengan auth7 M2M validation
- [ ] Update producer SDK dengan token refresh logic
- [ ] Add support untuk delegated token (BFF query notifikasi)
- [ ] Update documentation untuk producer integration

### bos7-template
- [ ] Register bos7-portal sebagai OAuth2 client di auth7
- [ ] Implement service account (client credentials)
- [ ] Create token exchange middleware
- [ ] Update BFF routes untuk menggunakan delegated tokens
- [ ] Add documentation: "Calling Backend Services from BFF"

---

## Breaking Changes

| Service | Change | Migration Guide |
|---------|--------|-----------------|
| workflow7 | `X-Service-Key` header deprecated | Use `Authorization: Bearer <token>` |
| notif7 | Producer JWT secret rotation | Use auth7 M2M flow |
| All | Casbin policy format | TBD: sync dengan auth7 atau tetap lokal |

---

## Communication Plan

Setelah Plan 11 dibuat, kirim notifikasi ke:

### 1. bos7-template (Issue/Discussion)
```
Subject: [Security Update] Integrasi auth7 untuk BFF Pattern

bos7-template perlu diupdate dengan:
1. BFF Service Account (OAuth2 client credentials)
2. Token Exchange middleware (RFC 8693)
3. Contoh: Call workflow7 dengan delegated token

Deadline: Setelah auth7 Plan 10 selesai
Reference: auth7/docs/security/INTERNAL-SERVICE-SECURITY.md
```

### 2. service7-template (Issue/Discussion)
```
Subject: [Security Update] Security Pattern Update untuk Services

service7-template perlu diupdate dengan:
1. M2M authentication example (client credentials)
2. BFF token exchange example
3. Updated specs & guidance

Impact: Services scaffolded dari template akan otomatis mendapat pola security baru
```

### 3. workflow7 (GitHub Issue)
```
Subject: [Migration] Migrate ke auth7 Centralized Security

workflow7 perlu migrasi:
1. X-Service-Key → auth7 M2M JWT
2. Casbin lokal → auth7 gRPC AuthCheck (opsional)
3. Support delegated token untuk BFF calls

Benefits:
- Standardized security
- No more static API keys
- Centralized audit trail
```

### 4. notif7 (GitHub Issue)
```
Subject: [Migration] Producer Auth Migration ke auth7

notif7 perlu update:
1. Producer JWT (HS256) → auth7 M2M JWT (RS256)
2. Add token exchange support untuk BFF queries

Benefits:
- Standardized dengan ekosistem Core7
- Better token lifecycle management
```

---

## Dependencies

- **auth7**: Plan 01-10 selesai (functional OAuth2, gRPC AuthCheck, lib7-auth-go)
- **lib7-auth-go**: Token exchange client & M2M token manager
- **Project Board**: Issues akan ditambahkan ke Core7 v2026.1 Project #8

---

## Success Criteria

1. ✅ Semua services menggunakan standardized M2M authentication via auth7
2. ✅ BFF pattern (token exchange) tersedia di bos7-template
3. ✅ Tidak ada static API keys di production (X-Service-Key deprecated)
4. ✅ Semua services bisa verify token via JWKS (stateless)
5. ✅ Audit trail terintegrasi di auth7

---

*Created: 2026-04-24 | Plan ini dieksekusi setelah auth7 Plan 01-10 selesai*
