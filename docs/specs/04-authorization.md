# Auth7 — Spec 04: Authorization

> **Versi**: 1.0-draft | **Tanggal**: 2026-04-22 | **Fase**: Brainstorming
> **Analogi**: Ory Keto (simplified) + Casbin

---

## 1. Authorization Model

Auth7 menggunakan **hybrid RBAC + ABAC** untuk authorization:

```
┌────────────────────────────────────────────────────────┐
│  Authorization Decision                                │
│                                                        │
│  RBAC:  User → Roles → Permissions → Resource:Action  │
│                    +                                   │
│  ABAC:  Context Conditions (branch, time, IP, etc.)    │
│                    =                                   │
│  Decision: ALLOW or DENY                               │
└────────────────────────────────────────────────────────┘
```

### Filosofi
- **Default Deny**: jika tidak ada rule yang explicitly allow, maka DENY
- **Tenant-Scoped**: semua permission di-scope ke org_id
- **Role Inheritance**: role dapat inherit dari role lain
- **Casbin Backend**: RBAC model menggunakan Casbin (konsisten dengan service7-template)

- **RBAC** (Role-Based Access Control): 90% use cases — role → permission mapping
- **ABAC** (Attribute-Based Access Control): 10% use cases — time-based, multi-attribute, complex logic

### 1.1 RBAC Model

```
Organization (Bank)
  ├── Role (dapat per-org)
  │     ├── Permission Set
  │     └── inherits_from (role lain)
  └── User
        └── UserRole (user → role, dalam konteks org)
```

```
User → Role → Permission → Resource
```

| Entity | Deskripsi |
|---|---|
| **User** | User yang punya role |
| **Role** | Kumpulan permissions (teller, supervisor, org_admin) |
| **Permission** | Action pada resource (account:read, transaction:create) |
| **Resource** | Objek yang dilindungi (account, transaction, user, dll) |

### 1.2 Permission Model

```
permission = resource:action

resource  = "workflow", "user", "report", "config", etc.
action    = "read", "write", "delete", "approve", "reject", dll.

Contoh:
  "workflow:read"
  "workflow:approve"
  "user:create"
  "user:delete"
  "report:export"
  "config:manage"
```

### 1.2 ABAC Model

ABAC menggunakan **hybrid JSON Rules + OPA Rego**:

```json
{
  "type": "json",
  "rule": {
    "resource": "account",
    "action": "read",
    "condition": {
      "attribute": "branch_id",
      "operator": "eq",
      "value": "${user.branch_id}"
    }
  }
}
```

```rego
# Rego policy untuk complex rules
package authz

default allow = false

allow {
  input.user.roles[_] == "org_admin"
  input.resource.type == "account"
  input.action == "read"
  input.resource.org_id == input.user.org_id
}

# Time-based rule
allow {
  input.user.roles[_] == "teller"
  input.resource.type == "transaction"
  input.action == "create"
  time.now_ns() >= input.policy.start_time_ns
  time.now_ns() <= input.policy.end_time_ns
}
```

### 1.3 Policy Schema

```json
{
  "type": "json" | "rego",
  "rule": { ... }  // JSON rule atau Rego policy
}
```

Developer bisa pilih yang sesuai kompleksitas.

---

## 2. Casbin Integration

### 2.1 Model Conf (RBAC with Domains)

Auth7 menggunakan **RBAC dengan domain/tenant** support:

```ini
# casbin model
[request_definition]
r = sub, dom, obj, act

[policy_definition]
p = sub, dom, obj, act, eft

[role_definition]
g = _, _, _   # user, role, domain

[policy_effect]
e = some(where (p.eft == allow)) && !some(where (p.eft == deny))

[matchers]
m = g(r.sub, p.sub, r.dom) && r.dom == p.dom && r.obj == p.obj && r.act == p.act
```

### 2.2 Custom PGX Adapter

- **Keputusan**: Custom pgx adapter (bukan gorm-adapter)
- Lebih lean, konsisten dengan stack pgx
- Tanpa dependency gorm

### 2.3 Policy Storage

```sql
CREATE TABLE casbin_rule (
    id SERIAL PRIMARY KEY,
    org_id UUID NOT NULL,
    ptype VARCHAR(100) NOT NULL,
    v0 VARCHAR(100) NOT NULL,
    v1 VARCHAR(100) NOT NULL,
    v2 VARCHAR(100) NOT NULL,
    v3 VARCHAR(100) DEFAULT '',
    v4 VARCHAR(100) DEFAULT '',
    v5 VARCHAR(100) DEFAULT ''
);

CREATE INDEX idx_casbin_rule_org ON casbin_rule(org_id);
CREATE INDEX idx_casbin_rule_ptype ON casbin_rule(ptype, v0);
```

### 2.4 Policy Sync (Redis Pub/Sub)

- Multi-instance auth7 → policy sync via Redis pub/sub
- Channel: `policy:updated`
- Setiap policy change → publish event → semua instance reload

---

## 3. Permission API

### 3.1 Check Permission

```
POST /api/v1/authz/check
{
  "subject": "user-uuid",
  "resource": "account",
  "action": "read",
  "context": {
    "branch_id": "branch-uuid",
    "time": "2026-04-22T10:00:00Z"
  }
}

Response:
{
  "allowed": true,
  "reason": "role:teller has permission account:read"
}
```

### 3.2 List Permissions

```
GET /api/v1/authz/permissions?subject=user-uuid

Response:
{
  "permissions": [
    "account:read",
    "account:write",
    "transaction:create"
  ]
}
```

### 3.3 gRPC CheckPermission

```protobuf
service AuthzService {
  rpc CheckPermission(CheckPermissionRequest) returns (CheckPermissionResponse);
  rpc ListPermissions(ListPermissionsRequest) returns (ListPermissionsResponse);
}

message CheckPermissionRequest {
  string subject = 1;
  string resource = 2;
  string action = 3;
  map<string, string> context = 4;
}

message CheckPermissionResponse {
  bool allowed = 1;
  string reason = 2;
}
```

---

## 4. Wildcard Permissions

- Casbin support wildcard (`*`) untuk admin super permissions
- Role `super_admin` → `*:*` (all resources, all actions)
- Role `org_admin` → `*:*` scoped to org_id

---

## 5. Role Management

### 5.1 Role Entity

```go
type Role struct {
    ID          uuid.UUID
    OrgID       uuid.UUID
    Name        string
    Description string
    IsSystem    bool          // system roles tidak bisa dihapus
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### 5.2 System Roles

| Role | Deskripsi | Permissions |
|---|---|---|
| `super_admin` | Full access semua org | `*:*` |
| `org_admin` | Admin per organisasi | `*:*` (scoped to org) |
| `branch_admin` | Admin per cabang | `*:*` (scoped to branch) |

### 5.3 Custom Roles

- Dibuat oleh org_admin
- Permissions bisa dipilih dari list available permissions
- Bisa assign ke multiple users

---

## 6. Open Questions

1. **Apakah perlu Zanzibar-style relation-based authz di v2.0?**
   → Ya, untuk fine-grained resource-level permissions

2. **Apakah perlu policy versioning?**
   → v1.0: Tidak
   → v1.1: Ya (audit trail policy changes)

---

*Prev: [03-oauth2-oidc.md](./03-oauth2-oidc.md) | Next: [05-session-token.md](./05-session-token.md)*
