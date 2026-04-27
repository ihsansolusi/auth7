# Prompt: Auth7 — Update Spec & Plan for NATS Integration

**Context:** Auth7 sedang dalam fase planning. Hybrid messaging model (Redis + NATS) telah diadopsi sebagai standard untuk Core7 ecosystem. Auth7 perlu mengintegrasikan NATS untuk token revocation events.

**Reference:**
- `docs/infra/HYBRID-MESSAGING-MODEL.md` — decision document
- `docs/infra/hybrid-messaging-plans/P-AUTH7-01-NATS-Token-Events.md` — implementation plan
- `supported-apps/auth7/docs/specs/05-session-token.md` — current spec

---

## Tasks for Auth7 Session

### 1. Review Current Spec 05 (Session Token)

Baca `docs/specs/05-session-token.md` — identifikasi:
- Di mana token revocation ditangani saat ini (Redis-only)
- Di mana session events dipublish
- Gap antara current design dan hybrid model requirement

### 2. Review Current Plans (Plan 10/11)

Baca `docs/plans/PLAN-10.md` dan `docs/plans/PLAN-11.md`:
- Plan 10 (Integration) → target NATS for token revocation
- Plan 11 (Service Migration) → target multi-instance with shared NATS

Identifikasi apakah NATS tasks sudah termasuk, jika belum perlu ditambahkan.

### 3. Update Spec 05 (if needed)

Tambahkan section untuk NATS integration:

```markdown
## X. NATS Integration (v1.0)

### X.1 Events Published

| Event | Subject | Subscribers |
|-------|---------|-------------|
| `token.revoked` | `auth7.tokens.revoked` | workflow7, notif7 |
| `session.created` | `auth7.sessions.created` | audit |
| `session.terminated` | `auth7.sessions.terminated` | audit, notif7 |

### X.2 Configuration

```yaml
messaging:
  nats:
    url: "${NATS_URL}"
    name: "auth7"
```

### X.3 Fail-Safe Design

- NATS publish failures should be non-fatal
- Database is source of truth for token validity
- Log warnings on publish failures
```

### 4. Update Plan 10/11 (if needed)

Tambahkan tasks untuk NATS:

```markdown
## NATS Integration
- [ ] Add lib7-service-go/nats dependency
- [ ] Create messaging package
- [ ] Wire NATS client in cmd/start.go
- [ ] Publish TokenRevokedEvent on org-wide revocation
- [ ] Publish SessionCreatedEvent on login
- [ ] Subscribe to policy7.params.updated (for OPA cache invalidation)
- [ ] Integration tests with NATS
```

---

## Key Decisions to Confirm

1. **Subject naming convention:** `auth7.tokens.revoked` vs `auth7.token.revoked` (singular vs plural)
2. **Event payload:** Include `RevokedBy` admin ID? Include `Reason`?
3. **Fallback:** Keep Redis pub/sub sebagai fallback jika NATS unavailable?
4. **Scheduling:** Apakah NATS integration masuk Plan 10 atau perlu plan baru (Plan 12)?

---

## Output

1. Updated Spec 05 dengan NATS section (atau konfirmasi bahwa existing design sudah compatible)
2. Updated Plan 10/11 dengan NATS tasks (atau confirmation bahwa sudah included)
3. List decisions yang perlu dibuat + recommended choices
4. Any spec changes → commit to auth7 repo
