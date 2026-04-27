# Auth7 — Plan 12: NATS Event Streaming Integration

> **Status**: 📋 Planned  
> **Total Issues**: 7  
> **Prerequisite**: Plan 10 (Integration) — bisa parallel  
> **Timeline**: v1.0 (Parallel dengan Plan 10/11)

---

## Goal

Integrasi **NATS** untuk event streaming dan service communication di ekosistem Core7, sebagai bagian dari **Hybrid Messaging Model** (Redis + NATS).

**Redis**: Caching, sessions, hot data  
**NATS**: Event streaming, service communication, cache invalidation

---

## Background

Hybrid Messaging Model telah diadopsi sebagai standard untuk Core7 ecosystem:
- `docs/infra/HYBRID-MESSAGING-MODEL.md` — decision document
- Policy7 akan menggunakan NATS untuk parameter change events
- Auth7 perlu publish token/session events untuk konsumsi service lain
- Auth7 perlu subscribe ke policy7 untuk OPA cache invalidation

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Auth7                                │
│                                                              │
│  ┌─────────────┐        ┌─────────────────────────────┐     │
│  │   Redis     │        │            NATS             │     │
│  │  (Cache)    │        │      (Event Streaming)      │     │
│  │             │        │                             │     │
│  │ • Sessions  │        │  PUBLISH:                   │     │
│  │ • Tokens    │        │  • auth7.tokens.revoked     │     │
│  │ • Hot data  │        │  • auth7.sessions.created   │     │
│  └─────────────┘        │  • auth7.sessions.terminated│     │
│                         │  • auth7.security.alert     │     │
│                         │                             │     │
│                         │  SUBSCRIBE:                 │     │
│                         │  • policy7.params.updated   │     │
│                         │                             │     │
│                         └──────────────┬──────────────┘     │
│                                        │                    │
└────────────────────────────────────────┼────────────────────┘
                                         │
           ┌─────────────────────────────┼─────────────────────┐
           │                             │                     │
           ▼                             ▼                     ▼
      ┌─────────┐                   ┌─────────┐          ┌─────────┐
      │workflow7│                   │ core7   │          │ notif7  │
      │         │                   │enter-   │          │         │
      │         │                   │prise    │          │         │
      └─────────┘                   └─────────┘          └─────────┘
```

---

## Issues

| # | GitHub Issue | Title | Est. Points | Priority | Dependencies |
|---|--------------|-------|-------------|----------|--------------|
| 12.1 | [#106](https://github.com/ihsansolusi/auth7/issues/106) | NATS client setup & configuration | 2 | 🔴 HIGH | **lib7 L07** |
| 12.2 | [#107](https://github.com/ihsansolusi/auth7/issues/107) | Publish token revocation events | 3 | 🔴 HIGH | — |
| 12.3 | [#108](https://github.com/ihsansolusi/auth7/issues/108) | Publish session lifecycle events | 2 | 🟡 MEDIUM | — |
| 12.4 | [#109](https://github.com/ihsansolusi/auth7/issues/109) | Publish security alert events | 2 | 🟡 MEDIUM | — |
| 12.5 | [#110](https://github.com/ihsansolusi/auth7/issues/110) | Subscribe to policy7 parameter updates | 3 | 🔴 HIGH | — |
| 12.6 | [#111](https://github.com/ihsansolusi/auth7/issues/111) | OPA cache invalidation on policy updates | 3 | 🔴 HIGH | — |
| 12.7 | [#112](https://github.com/ihsansolusi/auth7/issues/112) | Integration tests dengan NATS | 3 | 🟡 MEDIUM | **lib7 L07** |

**Total**: 18 points, 7 issues

### GitHub Group Issue
- **#105** — [Plan 12 — NATS Event Streaming Integration](https://github.com/ihsansolusi/auth7/issues/105)

---

## Key Deliverables

### Publishers (Auth7 → NATS)

| Event | Subject | Payload | Trigger |
|-------|---------|---------|---------|
| Token Revoked | `auth7.tokens.revoked` | `{token_id, org_id, revoked_by, reason}` | Admin revoke, logout, password change |
| Token Refreshed | `auth7.tokens.refreshed` | `{token_id, org_id, user_id}` | Token refresh |
| Session Created | `auth7.sessions.created` | `{session_id, org_id, user_id, ip}` | Login sukses |
| Session Terminated | `auth7.sessions.terminated` | `{session_id, org_id, reason}` | Logout, timeout, admin revoke |
| Session Revoked All | `auth7.sessions.revoked_all` | `{org_id, revoked_by}` | Emergency logout all |
| Security Alert | `auth7.security.alert` | `{type, org_id, user_id, details}` | Brute force, new device, etc. |

### Subscribers (NATS → Auth7)

| Subject | Publisher | Handler |
|---------|-----------|---------|
| `policy7.params.updated` | policy7 | Invalidate OPA cache untuk parameter yang berubah |
| `policy7.params.deleted` | policy7 | Invalidate OPA cache |

---

## Configuration

```yaml
# configs/nats.yaml
messaging:
  nats:
    url: "${NATS_URL}"           # nats://localhost:4222
    name: "auth7"                 # Client name
    reconnect_wait: 2s            # Reconnect backoff
    max_reconnects: 10            # Max reconnect attempts
    
    # JetStream (for persistence - v1.1)
    jetstream:
      enabled: false              # v1.0: false, v1.1: true
      
    # Publishing
    publish:
      timeout: 5s
      retry: 3
      
    # Subscriptions
    subscribe:
      queue_group: "auth7-opa"    # Load balancing untuk OPA updates
      durable: "auth7-opa-durable" # Durable consumer name
```

---

## Fail-Safe Design

### Publishing Failures
- **Non-fatal**: NATS publish failures tidak block business logic
- **Logging**: Log warning dengan correlation ID
- **Retry**: Retry dengan exponential backoff (3 attempts)
- **Fallback**: Database tetap source of truth

### Subscription Failures
- **Reconnection**: Automatic reconnect dengan backoff
- **Buffering**: Buffer messages saat offline (JetStream v1.1)
- **Graceful degradation**: Continue tanpa cache invalidation (stale data acceptable briefly)

### Example Code

```go
// internal/messaging/nats/publisher.go

func (p *Publisher) PublishTokenRevoked(ctx context.Context, event TokenRevokedEvent) {
    const op = "messaging.PublishTokenRevoked"
    
    data, _ := json.Marshal(event)
    
    err := p.nc.Publish("auth7.tokens.revoked", data)
    if err != nil {
        // Log warning, don't fail
        p.logger.Warn().
            Str("op", op).
            Err(err).
            Str("token_id", event.TokenID).
            Msg("failed to publish token revoked event")
        return
    }
    
    p.logger.Debug().
        Str("op", op).
        Str("token_id", event.TokenID).
        Msg("token revoked event published")
}

// internal/messaging/nats/subscriber.go

func (s *Subscriber) SubscribePolicyUpdates() {
    sub, _ := s.nc.Subscribe("policy7.params.updated", func(msg *nats.Msg) {
        var event PolicyUpdateEvent
        json.Unmarshal(msg.Data, &event)
        
        // Invalidate OPA cache
        s.opa.InvalidateCache(event.OrgID, event.ParameterName)
    })
    
    s.subscriptions = append(s.subscriptions, sub)
}
```

---

## Integration dengan Layanan Lain

### workflow7
- Subscribe: `auth7.tokens.revoked` → Invalidate local token cache
- Subscribe: `auth7.sessions.revoked_all` → Force logout users

### core7-enterprise
- Subscribe: `auth7.tokens.revoked` → Invalidate token cache
- Subscribe: `auth7.security.alert` → Log security events

### notif7
- Subscribe: `auth7.security.alert` → Send notification (email/SMS)
- Subscribe: `auth7.sessions.terminated` → Audit log

### policy7
- Publish: `policy7.params.updated` → Auth7 invalidates OPA cache
- Publish: `policy7.params.deleted` → Auth7 invalidates OPA cache

---

## Dependencies

| Dependency | Status | Notes |
|------------|--------|-------|
| **Plan 10 (Integration)** | 📋 Planned | Bisa parallel, tidak blocking |
| **NATS Infrastructure** | 🔲 TODO | Add ke unified docker-compose |
| **lib7-nats-client (L07)** | ⏳ **TODO** | **BLOCKING** — Task di lib7-service-go |

### ⚠️ Critical Dependency: lib7-nats-client (L07)

**Status**: ⏳ TODO di `libs/service-go/docs/plans/L07-nats-client.md`

**Task untuk lib7 team**:
- [ ] Create package `nats` di lib7-service-go
- [ ] Implementasi Client, Publisher, Subscriber, RequestReply
- [ ] Config: `config.NATSConfig`
- [ ] Unit tests >80%
- [ ] Documentation

**Timeline**: 
- L07 harus selesai **sebelum** auth7 Plan 12.1 dan policy7 Plan 05.1
- Bisa kerjakan parallel dengan auth7 Plan 01-09

**After L07 selesai, update**:
```go
// Replace inline NATS code dengan:
import "github.com/ihsansolusi/lib7-service-go/nats"
```

---

## Success Criteria

- [ ] Auth7 bisa publish events ke NATS
- [ ] Auth7 bisa subscribe ke policy7 events
- [ ] OPA cache ter-invalidate saat parameter berubah
- [ ] Token revocation events diterima oleh workflow7/core7
- [ ] Security alerts diterima oleh notif7
- [ ] Fail-safe: NATS failure tidak crash auth7
- [ ] Integration tests pass

---

## Notes

### v1.0 vs v1.1

| Feature | v1.0 | v1.1 |
|---------|------|------|
| NATS Core (pub/sub) | ✅ | ✅ |
| JetStream (persistence) | ❌ | ✅ |
| Durable subscriptions | ❌ | ✅ |
| lib7-messaging-go | ❌ (inline) | ✅ (shared lib) |

### Why Not Redis Pub/Sub?

| Aspect | Redis Pub/Sub | NATS |
|--------|---------------|------|
| Request-Reply | ❌ No | ✅ Yes |
| Queue Groups | ❌ No | ✅ Yes (load balancing) |
| Durable Subs | ❌ No | ✅ Yes |
| Reconnection | Basic | Advanced |
| Service Discovery | ❌ No | ✅ Yes |

**Decision**: NATS untuk service communication, Redis untuk caching.

---

*Created: 2026-04-27 | Part of Hybrid Messaging Model for Core7 Ecosystem*
