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
- **Branch-Scoped**: permission bisa di-scope ke 1 branch, beberapa branch, atau semua branch
- **Field-Level**: permission bisa membatasi field tertentu (e.g. read rekening tanpa saldo)
- **Role Inheritance**: role dapat inherit dari role lain
- **Casbin Backend**: RBAC model menggunakan Casbin (konsisten dengan service7-template)

- **RBAC** (Role-Based Access Control): 90% use cases — role → permission mapping
- **ABAC** (Attribute-Based Access Control): 10% use cases — time-based, multi-attribute, complex logic

---

## 2. Authorization Layers

Authorization di Auth7 memiliki **4 layer** yang bekerja bersamaan:

```
┌──────────────────────────────────────────────────┐
│  Layer 4: FIELD-LEVEL (attribute masking)       │
│  "User bisa read rekening, TAPI tidak read saldo" │
│  ─────────────────────────────────────────────── │
│  Layer 3: DATA-SCOPE (branch filtering)         │
│  "User bisa read rekening CABANG BANDUNG saja"  │
│  ─────────────────────────────────────────────── │
│  Layer 2: DATA-ACCESS (object permission)        │
│  "User bisa read rekening, write transaksi"      │
│  ─────────────────────────────────────────────── │
│  Layer 1: PAGE/MENU (navigation access)          │
│  "User bisa buka halaman Rekening"               │
└──────────────────────────────────────────────────┘
```

Setiap request melewati **semua layer** dari atas ke bawah. Jika satu layer DENY, akses ditolak.

---

## 3. Layer 1 — Page/Menu Access

### 3.1 Konsep

Menu/page permission mengontrol **navigasi** — halaman apa yang bisa dilihat user di UI.
Ini adalah "gate" pertama: jika user tidak punya akses ke menu, halaman tidak ditampilkan.

### 3.2 Permission Format

```
menu:{menu_id}:access

Contoh:
  menu:accounts:access      → bisa buka halaman Daftar Rekening
  menu:transactions:access  → bisa buka halaman Transaksi
  menu:reports:access       → bisa buka halaman Laporan
  menu:admin:access         → bisa buka halaman Admin
  menu:admin.users:access   → bisa buka submenu Admin > Users
```

### 3.3 Casbin Policy

```ini
# Teller bisa buka menu rekening dan transaksi
p, teller, org-1, menu:accounts, access, allow
p, teller, org-1, menu:transactions, access, allow

# Supervisor bisa buka semua menu semua cabang
p, supervisor, org-1, menu:*:, access, allow
```

### 3.4auth7-ui Integration

```typescript
// Frontend mendapat daftar menu yang diizinkan
const userMenus = await auth7.getMenus(userId, branchId);

// Response:
{
  "menus": [
    {"id": "accounts", "label": "Rekening", "path": "/accounts"},
    {"id": "transactions", "label": "Transaksi", "path": "/transactions"}
  ]
}

// UI menyembunyikan menu yang tidak ada di list
```

---

## 4. Layer 2 — Data Access Permission

### 4.1 Konsep

Setelah user bisa buka halaman, layer ini mengontrol **operasi apa** yang bisa dilakukan pada **tipe data** tertentu.

### 4.2 Permission Format

```
{resource}:{action}

resource  = tipe data (account, transaction, product, user, report, dll)
action    = read, write, create, delete, approve, reject, export, dll

Contoh:
  account:read          → bisa lihat data rekening
  account:write         → bisa edit data rekening
  transaction:create    → bisa buat transaksi
  transaction:approve   → bisa approve transaksi
  product:read          → bisa lihat produk
```

### 4.3 Banking Examples

| Role | Permission | Deskripsi |
|---|---|---|
| Teller | `account:read`, `transaction:create` | Lihat rekening, buat transaksi |
| Supervisor | `account:read`, `account:write`, `transaction:create`, `transaction:approve` | + Edit rekening, approve transaksi |
| Branch Admin | `account:*`, `transaction:*`, `user:read` | Full CRUD per cabang |
| Auditor | `account:read`, `transaction:read`, `audit_log:read` | Read-only semua data |
| GL Accountant | `gl_account:read`, `gl_account:write`, `journal:create` | Akses ke General Ledger |

---

## 5. Layer 3 — Data Scope (Branch Filtering)

### 5.1 Konsep

Permission bisa di-scope ke branch tertentu. User yang punya akses `account:read` mungkin hanya bisa lihat rekening di **cabang tertentu**, bukan semua cabang.

### 5.2 Scope Types

```
scope_type:
  "own_branch"       → hanya branch yang sedang aktif (active_branch_id)
  "assigned_branches" → branch yang di-assign ke user via user_branch_assignments
  "all_branches"     → semua branch dalam org (super admin, auditor)
```

### 5.3 Casbin Policy dengan Branch Scope

```ini
# Teller KC Bandung: hanya bisa read rekening di own branch
p, teller, org-1, account, read, allow, scope=own_branch

# Supervisor: bisa read rekening di semua assigned branches
p, supervisor, org-1, account, read, allow, scope=assigned_branches

# Auditor: bisa read semua branch
p, auditor, org-1, account, read, allow, scope=all_branches
```

### 5.4 Contoh Banking Case

```
John (supervisor) punya assignments:
  - KC Bandung (primary) → role: supervisor
  - KCP Dago            → role: teller
  - KC Jakarta           → role: supervisor

Saat John aktif di KC Bandung:
  account:read  → scope: assigned_branches → bisa lihat rekening KC Bandung, KCP Dago, KC Jakarta
  account:write → scope: own_branch        → bisa edit hanya di KC Bandung

Saat John switch ke KCP Dago:
  account:read  → scope: assigned_branches → sama (assigned branches tidak berubah)
  account:write → scope: own_branch        → bisa edit hanya di KCP Dago (role: teller)
```

### 5.5 Implementation: Query Filtering

```go
// Service layer menerapkan branch scope ke query database
func (s *AccountService) ListAccounts(ctx context.Context, filter AccountFilter) ([]Account, error) {
    const op = "AccountService.ListAccounts"
    
    claims := auth7.ClaimsFromContext(ctx)
    perm := claims.GetPermission("account:read")
    
    switch perm.Scope {
    case "own_branch":
        filter.BranchID = claims.ActiveBranchID
    case "assigned_branches":
        filter.BranchIDs = claims.AssignedBranchIDs  // dari user_branch_assignments
    case "all_branches":
        // tidak filter branch, bisa lihat semua
    default:
        return nil, ErrForbidden
    }
    
    return s.store.ListAccounts(ctx, filter)
}
```

---

## 6. Layer 4 — Field-Level Permission (Attribute Masking)

### 6.1 Konsep

Permission tidak hanya mengontrol **apakah** user bisa read suatu data, tapi juga **field mana** yang bisa dilihat.
Ini penting untuk banking: teller bisa lihat info nasabah, tapi tidak boleh lihat saldo.

### 6.2 Permission Format

```
{resource}:{action}:{field_mask}

Contoh:
  account:read              → bisa lihat semua field rekening (default)
  account:read:no_saldo     → bisa lihat rekening TANPA field saldo
  account:read:no_balance   → bisa lihat rekening TANPA field balance
  gl_account:read           → bisa lihat semua field GL Account

Lebih eksplisit (JSON ABAC):
  {
    "resource": "account",
    "action": "read",
    "fields_allowed": ["account_number", "customer_name", "status", "branch"],
    "fields_denied": ["balance", "limit"]
  }
```

### 6.3 Banking Examples

| User Type | Permission | Field Mask | Hasil |
|---|---|---|---|
| Teller | `account:read` | `{denied: ["balance", "limit"]}` | Lihat rekening tanpa saldo & limit |
| Supervisor | `account:read` | `{}` (full) | Lihat semua field |
| Auditor | `account:read` | `{}` (full) | Lihat semua field |
| CS (Customer Service) | `customer:read` | `{denied: ["credit_score", "internal_notes"]}` | Lihat data nasabah tanpa catatan internal |
| GL Accountant | `gl_account:read` | `{}` (full) | Lihat semua field GL |

### 6.4 Implementation: Response Filtering

```go
// Service layer menerapkan field masking ke response
func (s *AccountService) GetAccount(ctx context.Context, id uuid.UUID) (*Account, error) {
    const op = "AccountService.GetAccount"
    
    account, err := s.store.GetAccount(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("%s: %w", op, err)
    }
    
    // Apply field mask based on permission
    perm := auth7.ClaimsFromContext(ctx).GetPermission("account:read")
    if perm.FieldMask != nil {
        account = perm.FieldMask.Apply(account)  // zero-out denied fields
    }
    
    return account, nil
}

// FieldMask.Apply menghilangkan field yang denied
func (m *FieldMask) Apply(a *Account) *Account {
    for _, denied := range m.FieldsDenied {
        switch denied {
        case "balance":
            a.Balance = 0
        case "limit":
            a.Limit = 0
        }
    }
    return a
}
```

### 6.5 Data Model

```sql
-- Field-level permission masking
CREATE TABLE permission_field_masks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES organizations(id),
    resource        VARCHAR(100) NOT NULL,     -- 'account', 'customer', 'gl_account'
    action          VARCHAR(50) NOT NULL,      -- 'read'
    role_id         UUID REFERENCES roles(id), -- NULL = default mask untuk role
    fields_allowed  TEXT[],                     -- kosong = semua field
    fields_denied   TEXT[],                     -- field yang disensor
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (org_id, resource, action, role_id)
);
```

**Contoh data:**

| resource | action | role | fields_denied |
|---|---|---|---|
| `account` | `read` | teller | `{balance, limit}` |
| `customer` | `read` | cs | `{credit_score, internal_notes}` |
| `account` | `read` | supervisor | `{}` (kosong = semua field) |

---

## 7. RBAC Model Details

### 7.1 Role Entity

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

### 7.2 System Roles

| Role | Deskripsi | Permissions |
|---|---|---|
| `super_admin` | Full access semua org | `*:*` |
| `org_admin` | Admin per organisasi | `*:*` (scoped to org) |
| `branch_admin` | Admin per cabang | `*:*` (scoped to branch) |

### 7.3 Custom Roles

- Dibuat oleh org_admin
- Permissions bisa dipilih dari list available permissions
- Bisa assign ke multiple users
- Permissions bisa berbeda per branch (diatur via user_roles + branch_id)

### 7.4 Role-Branch Assignment

```sql
-- Roles di-assign per user per branch
CREATE TABLE user_roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    role_id     UUID NOT NULL REFERENCES roles(id),
    branch_id   UUID NOT NULL REFERENCES branches(id),
    org_id      UUID NOT NULL REFERENCES organizations(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, role_id, branch_id)
);

CREATE INDEX idx_user_roles_user ON user_roles(user_id);
CREATE INDEX idx_user_roles_branch ON user_roles(branch_id);
CREATE INDEX idx_user_roles_role ON user_roles(role_id);
```

**Contoh:**

| user_id | role_id | branch_id | Artinya |
|---|---|---|---|
| john | supervisor | KC Bandung | John = supervisor di KC Bandung |
| john | teller | KCP Dago | John = teller di KCP Dago |
| john | supervisor | KC Jakarta | John = supervisor di KC Jakarta |

Saat John switch active branch ke KCP Dago, role yang aktif = **teller** (bukan supervisor).

---

## 8. Casbin Integration

### 8.1 Model Conf (RBAC with Domains)

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

### 8.2 Custom PGX Adapter

- **Keputusan**: Custom pgx adapter (bukan gorm-adapter)
- Lebih lean, konsisten dengan stack pgx
- Tanpa dependency gorm

### 8.3 Policy Storage

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

### 8.4 Policy Sync (Redis Pub/Sub)

- Multi-instance auth7 → policy sync via Redis pub/sub
- Channel: `policy:updated`
- Setiap policy change → publish event → semua instance reload

---

## 9. Permission Check Flow (Complete)

Ketika user mengakses resource, system mengecek **semua layer** secara berurutan:

```
User request: "Lihat daftar rekening di KC Bandung"

Layer 1 — Page/Menu:
  ✓ user punya menu:accounts:access? → YES

Layer 2 — Data Access:
  ✓ user punya account:read? → YES

Layer 3 — Branch Scope:
  ✓ user punya akses ke KC Bandung?
    - own_branch? → cek active_branch_id
    - assigned_branches? → cek user_branch_assignments
    - all_branches? → allow
    → YES (assigned, KC Bandung ada di list)

Layer 4 — Field Masking:
  ✓ user bisa lihat field apa?
    - teller → {denied: [balance, limit]} → zero-out
    - supervisor → {} (full access)
    → Apply mask ke response

Result: ALLOW, dengan field masking sesuai role
```

---

## 10. Permission API

### 10.1 Check Permission

```
POST /api/v1/authz/check
{
  "subject": "user-uuid",
  "resource": "account",
  "action": "read",
  "context": {
    "branch_id": "branch-uuid",
    "fields": ["account_number", "customer_name", "balance"]
  }
}

Response:
{
  "allowed": true,
  "reason": "role:teller has permission account:read",
  "scope": "assigned_branches",
  "fields_denied": ["balance", "limit"]
}
```

### 10.2 List Permissions (per branch)

```
GET /api/v1/authz/permissions?subject=user-uuid&branch_id=branch-uuid

Response:
{
  "branch_id": "kc-bandung-uuid",
  "roles": ["supervisor"],
  "permissions": [
    {"resource": "account", "action": "read", "scope": "assigned_branches", "fields_denied": []},
    {"resource": "account", "action": "write", "scope": "own_branch", "fields_denied": []},
    {"resource": "transaction", "action": "create", "scope": "own_branch"},
    {"resource": "transaction", "action": "approve", "scope": "own_branch"}
  ],
  "menus": [
    {"id": "accounts", "label": "Rekening", "path": "/accounts"},
    {"id": "transactions", "label": "Transaksi", "path": "/transactions"}
  ]
}
```

### 10.3 gRPC CheckPermission

```protobuf
service AuthzService {
  rpc CheckPermission(CheckPermissionRequest) returns (CheckPermissionResponse);
  rpc ListPermissions(ListPermissionsRequest) returns (ListPermissionsResponse);
  rpc ListMenus(ListMenusRequest) returns (ListMenusResponse);
}

message CheckPermissionRequest {
  string subject = 1;
  string resource = 2;
  string action = 3;
  map<string, string> context = 4;   // branch_id, dll.
  repeated string fields = 5;         // field yang diminta (untuk field masking)
}

message CheckPermissionResponse {
  bool allowed = 1;
  string reason = 2;
  string scope = 3;                   // own_branch, assigned_branches, all_branches
  repeated string fields_denied = 4;  // field yang di-mask
}
```

---

## 11. Wildcard Permissions

- Casbin support wildcard (`*`) untuk admin super permissions
- Role `super_admin` → `*:*` (all resources, all actions)
- Role `org_admin` → `*:*` scoped to org_id

---

## 12. Banking Authorization Examples

### Contoh 1: Teller lihat rekening di cabangnya sendiri

```
Role: teller
Permission: account:read, scope=own_branch
Field mask: {denied: [balance, limit]}

GET /api/v1/accounts → hanya rekening di active_branch_id, tanpa field saldo & limit
```

### Contoh 2: Supervisor lihat rekening di beberapa cabang

```
Role: supervisor
Permission: account:read, scope=assigned_branches, field mask: {}
Permission: account:write, scope=own_branch, field mask: {}

GET /api/v1/accounts → rekening di semua assigned branches, semua field
PUT /api/v1/accounts/:id → hanya bisa edit di active_branch_id
```

### Contoh 3: Auditor lihat semua cabang

```
Role: auditor
Permission: account:read, scope=all_branches, field mask: {}
Permission: transaction:read, scope=all_branches
No write permissions

GET /api/v1/accounts → semua rekening di semua cabang, semua field
```

### Contoh 4: CS lihat nasabah tanpa data sensitif

```
Role: customer_service
Permission: customer:read, scope=own_branch
Field mask: {denied: [credit_score, internal_notes]}

GET /api/v1/customers → data nasabah tanpa credit_score & internal_notes
```

### Contoh 5: GL Accountant akses rekening GL per branch

```
Role: gl_accountant
Permission: gl_account:read, scope=assigned_branches
Permission: gl_account:write, scope=own_branch
Permission: journal:create, scope=own_branch

GET /api/v1/gl-accounts → GL accounts di assigned branches
POST /api/v1/journals → buat jurnal hanya di active branch
```

---

## 13. Transaction Limits & Approval Authority (Policy7)

> **Keputusan arsitektur**: Transaction limit dan approval limit **bukan domain auth7**.
> Auth7 hanya menyediakan **role, permission, branch scope, dan field mask**.
> Limit transaksi dan otorisasi berjenjang disimpan di **policy7** (service terpisah).
> OPA/Rego di auth7 bisa query policy7 untuk data parameter saat ABAC evaluation.

### 13.1 Auth7 menyediakan claims, bukan limit

Auth7 JWT dan API response memberikan:

```json
{
  "user_id": "john-uuid",
  "org_id": "bank-uuid",
  "branch_id": "kc-bandung-uuid",
  "roles": ["supervisor"],
  "permissions": ["account:read", "account:write", "transaction:create", "transaction:approve"]
}
```

Masing-masing **service consumer** yang mengecek limit. Auth7 TIDAK menyimpan:
- Max transaction amount per role
- Approval threshold (berapa rupiah perlu approval)
- Daily cumulative limit
- Product type filter

**Policy7** yang menyimpan ini semua, termasuk:
- Employee transaction limits
- Customer/nasabah limits
- Interest rates & fees
- Regulatory thresholds (CTR/STR)
- Operational hours

### 13.2 Contoh Case: Transaction Limit di Policy7

```
Policy7 menyimpan table transaction_limits:
  role_id | transaction_type | max_amount  | daily_limit  | requires_approval_above
  --------|------------------|-------------|-------------|------------------------
  teller  | transfer        | 10.000.000  | 50.000.000  | 10.000.000 (supervisor)
  teller  | cash_deposit    | 25.000.000  | 100.000.000 | 25.000.000 (supervisor)
  supervisor| transfer       | 100.000.000| 500.000.000 | 50.000.000 (branch_manager)
  branch_manager| transfer   | 500.000.000| unlimited   | 200.000.000 (head_office)

Policy7 juga menyimpan:
  customer_type | product     | daily_limit
  --------------|-------------|------------
  regular       | transfer    | 50.000.000
  premium       | transfer    | 500.000.000
  vip           | transfer    | unlimited
```

**Alur transaksi Rp 75.000.000 oleh teller:**
```
1. Teller kirim POST /api/v1/transactions {amount: 75M, ...}
2. Core7 cek auth7 permission: transaction:create? → YES
3. Core7 cek policy7 transaction_limits: teller max = 10M → 75M > 10M → DENY
4. Response: 403 "Transaction amount exceeds teller limit (max Rp 10M)"

Harus dibuat oleh supervisor:
1. Supervisor kirim POST /api/v1/transactions {amount: 75M}
2. Core7 cek auth7 permission: transaction:create + scope=own_branch → ALLOW
3. Core7 cek policy7 transaction_limits: supervisor max = 100M → 75M < 100M → ALLOW
4. Core7 cek policy7 approval_threshold: 75M > 50M → perlu branch_manager approve
5. Status: PENDING_APPROVAL → workflow7 mengirim approval task
```

**Auth7 claims yang dipakai core7:**
```
Authorization: Bearer <jwt>
→ core7 extract: user_id, org_id, branch_id, roles, permissions
→ core7 query policy7: transaction_limits where role = 'supervisor'
→ core7 decide: ALLOW or PENDING_APPROVAL or DENY
```

### 13.3 Contoh Case: Daily Cumulative Limit

```
Policy7 menyimpan daily limit, core7 menyimpan runtime counter:
  user_id | branch_id | date       | cumulative_amount
  --------|-----------|------------|-------------------
  teller1 | KC-BDG    | 2026-04-24 | 45.000.000

Teller buat transfer ke-6 (Rp 9M):
1. Cumulative saat ini: Rp 45M
2. Transfer baru: Rp 9M → total: Rp 54M
3. Policy7 daily limit teller: Rp 50M → 54M > 50M → DENY
4. Response: 403 "Daily transaction limit exceeded (Rp 50M)"

Auth7 TIDAK terlibat — ini sepenuhnya domain policy7 + core7.
Auth7 hanya menyediakan role information agar core7 bisa query policy7.

### 13.4 Contoh Case: Inter-Branch Transfer Approval

```
Transfer Rp 200M antar cabang oleh supervisor KC Bandung:

Core7 logic:
1. Auth7 permission check: transaction:create, scope=assigned_branches → ALLOW
2. Transaction limit check: supervisor limit 100M → 200M > 100M → DENY per creation
   → Harus dibuat oleh branch_manager (limit 500M)

Branch_manager create 200M:
1. Core7 cek permission: ALLOW (limit 500M)
2. Core7 cek approval_threshold: inter-branch > 100M → perlu HEAD_OFFICE approve
3. Workflow7 mengirim approval task ke HEAD_OFFICE
4. Status: PENDING_APPROVAL (awaiting HO approval)

Auth7 claims yang dipakai:
→ role: branch_manager → core7 lookup limit table
→ scope: assigned_branches → core7 cek branch access
```

### 13.5 Contoh Case: Time-Based Restriction (Jam Operasional)

```
Policy7 menyimpan table operational_hours:
  role_id     | allowed_hours         | allowed_days
  ------------|-----------------------|-------------------
  teller      | 08:00-16:00 WIB       | mon,tue,wed,thu,fri
  supervisor  | 08:00-18:00 WIB       | mon,tue,wed,thu,fri
  branch_manager| 00:00-23:59 WIB     | *

Teller buat transaksi jam 21:00:
1. Core7 cek permission dari auth7 → ALLOW
2. Core7 cek policy7 operational_hours: teller hanya 08:00-16:00 → DENY
3. Response: 403 "Outside operational hours for teller (08:00-16:00 WIB)"

Auth7 ONLY provides role → core7 queries policy7 for time rules.

### 13.6 Contoh Case: Product Access Control

```
Policy7 menyimpan table role_product_access:
  role_id        | product_types
  ---------------|---------------------------
  teller         | ["tabungan", "deposito"]
  loan_officer   | ["kredit", "tabungan"]
  branch_manager | ["*"]

Teller akses halaman Kredit:
1. Auth7 menu check: teller punya menu:loans:access? → NO
2. UI tidak menampilkan menu Kredit

Jika somehow bypass:
3. Core7 API check: policy7 → teller role + product_type "kredit" → DENY
```

### 13.7 Contoh Case: Suspicious Transaction (AML/KYC)

```
Policy7 menyimpan suspicious thresholds, core7 + workflow7 execute:

Transaksi teller Rp 8M ke rekening baru:
1. Core7 cek permission dari auth7 → ALLOW
2. Core7 cek policy7 transaction_limits: 8M < 10M (teller limit) → ALLOW
3. Core7 business rule ( dari policy7 ): jika transfer > 5M ke rekening baru → RAISE FLAG
4. Core7 kirim event ke notif7: auth.suspicious_transaction
5. Compliance officer (role terpisah) di notif7 → review flag

Auth7 claims yang dipakai:
→ role: teller → core7 decide limit & flagging rules
→ branch_id → core7 decide branch scope
```

### 13.8 Contoh Case: Dormant Account Reactivation

```
Policy7 menyimpan operational matrix:
  role_id         | can_reactivate | requires_approval_by
  ----------------|-----------------|----------------------
  teller          | NO              | -
  supervisor      | YES             | branch_manager
  branch_manager  | YES             | - (auto-approve)

Supervisor request reaktivasi:
1. Auth7 permission check: account:reactivate? → YES
2. Core7 query policy7: supervisor can_reactivate=YES, requires_approval=branch_manager
3. Workflow7 membuat approval task
4. Branch_manager approve
5. Account reactivated

Teller request reaktivasi:
1. Auth7 permission check: account:reactivate? → NO (teller tidak punya permission)
2. Response: 403 directly from auth7, core7 tidak terlibat
```

---

## 14. Summary: Auth7 vs Service Domain

| Domain | Dimiliki | Contoh |
|---|---|---|
| Role & Permission definition | **auth7** | `account:read`, `transaction:approve`, `menu:accounts:access` |
| Branch Scope (own/assigned/all) | **auth7** | `scope=assigned_branches` |
| Field Masking | **auth7** | `{denied: [balance, limit]}` |
| Menu/Page access | **auth7** | `menu:accounts:access` |
| ABAC boolean rules (allow/deny) | **auth7** (OPA/Rego) | "teller hanya jam 08-16" (data dari policy7) |
| Transaction limit amount | **policy7** | max Rp 10M per teller |
| Daily cumulative limit | **policy7** | max Rp 50M per day |
| Approval threshold | **policy7** | > Rp 50M perlu manager approve |
| Operational hours | **policy7** | 08:00-16:00 for teller |
| Product access | **policy7** | teller hanya tabungan & deposito |
| Interest rates & fees | **policy7** | bunga deposito 12m = 4.5% |
| Regulatory thresholds | **policy7** | CTR reporting > Rp 100M |
| Suspicious flag rules | **core7 enterprise + workflow7** | > Rp 5M ke rekening baru |
| Dormant account matrix | **core7 enterprise** | who can reactivate, who must approve |
| Approval flow orchestration | **workflow7** | who approves, how many levels |

> **Prinsip**: Auth7 menjawab "BOLEHKAH user ini akses resource ini di branch ini?" (YES/NO).
> Policy7 menjawab "BOLEHKAH seberapa? BERAPA batasnya? KAPAN?" (numeric/threshold values).
> Workflow7 mengatur "SIAPA yang harus approve? BAGAIMANA flow-nya?"

> **Hubungan Auth7 ↔ Policy7**: OPA/Rego di auth7 bisa query policy7 untuk
> data parameter (jam operasional, threshold) sebagai condition dalam ABAC evaluation.
> Tapi auth7 TIDAK menyimpan data parameter — hanya mengonsumsinya saat evaluasi.

---

## 15. Multi-Domain Execution Flows

Bagian ini menunjukkan **alur eksekusi end-to-end** untuk case multi-domain,
menggambarkan bagaimana auth7, core7 enterprise, dan workflow7 berinteraksi.

### 15.1 Flow: Teller Transfer Melebihi Limit

Teller mencoba transfer Rp 75M (limit teller hanya Rp 10M).

```
Browser               bos7-portal          core7-enterprise      auth7-svc
  │                       │                       │                   │
  │ POST /transactions    │                       │                   │
  │ {amount: 75M, ...}   │                       │                   │
  │──────────────────────►│                       │                   │
  │                       │  1. Extract JWT      │                   │
  │                       │     claims:           │                   │
  │                       │     role: teller      │                   │
  │                       │     branch: KC-BDG   │                   │
  │                       │     permissions:       │                   │
  │                       │       transaction:create│                 │
  │                       │                       │                   │
  │                       │  2. [LAYER 1] Menu check (cached)          │
  │                       │     menu:transactions? → YES               │
  │                       │                       │                   │
  │                       │  3. [LAYER 2] Permission check              │
  │                       │     transaction:create? → YES             │
  │                       │     (dari JWT claims, no call needed)      │
  │                       │                       │                   │
  │                       │  4. [LAYER 3] Branch scope                 │
  │                       │     scope=own_branch → KC-BDG              │
  │                       │     amount branch match? → YES             │
  │                       │                       │                   │
  │                       │  5. [LAYER 4] Field masking                  │
  │                       │     teller → {denied: [balance, limit]}   │
  │                       │     N/A for transaction → skip             │
  │                       │                       │                   │
  │                       │  6. [POLICY7] Transaction limit check     │
  │                       │     GET policy7 /v1/params/limits         │
  │                       │       ?role=teller&type=transfer          │
  │                       │     → max_amount: 10M                     │
  │                       │     → 75M > 10M → DENY                     │
  │                       │                       │                   │
  │◄── 403 Forbidden ─────│                       │                   │
  │   "Amount exceeds      │                       │                   │
  │    teller limit"       │                       │                   │
```

**Catatan**: Auth7 TIDAK menerima request apapun. Semua limit check
dilakukan core7 berdasarkan role dari JWT claims.

### 15.2 Flow: Supervisor Transfer dengan Approval (Berjenjang)

Supervisor transfer Rp 75M. Limit supervisor 100M, tapi > 50M perlu branch_manager approve.

```
Browser         bos7-portal      core7-enterprise  auth7-svc    workflow7-svc  policy7
  │                 │                  │               │              │            │
  │ POST /trans-   │                  │               │              │            │
  │ actions         │                  │               │              │            │
  │ {amount:75M}   │                  │               │              │            │
  │────────────────►│                  │               │              │            │
  │                 │ 1. JWT claims:   │               │              │            │
  │                 │    role: supervisor│               │              │            │
  │                 │    branch: KC-BDG│               │              │            │
  │                 │    permissions:  │               │              │            │
  │                 │      transaction:create            │              │            │
  │                 │                  │               │              │            │
  │                 │ 2. [AUTH7 LAYERS]               │              │            │
  │                 │    menu:transactions → YES       │              │            │
  │                 │    permission:create → YES      │              │            │
  │                 │    scope=own_branch → KC-BDG OK  │              │            │
  │                 │    field_mask: {} (no mask)      │              │            │
  │                 │                  │               │              │            │
  │                 │ 3. [POLICY7: Limit check]       │              │            │
  │                 │    GET policy7 /v1/params/limits?role=supervisor&type=transfer │
  │                 │                  │               │              │            │
  │                 │◄──────────────────────────────────────────────────────────────│
  │                 │    {max_amount: 100M}           │              │            │
  │                 │    75M < 100M → AMOUNT OK       │              │            │
  │                 │                  │               │              │            │
  │                 │ 4. [POLICY7: Approval check]     │              │            │
  │                 │    GET policy7 /v1/params/thresholds?role=supervisor&type=transfer │
  │                 │                  │               │              │            │
  │                 │◄──────────────────────────────────────────────────────────────│
  │                 │    {requires_approval_above: 50M} │              │            │
  │                 │    75M > 50M → REQUIRES APPROVAL │              │            │
  │                 │                  │               │              │            │
  │                 │ 5. INSERT transaction             │              │            │
  │                 │    status: PENDING_APPROVAL      │              │            │
  │                 │                  │               │              │            │
  │                 │ 6. CREATE workflow7 task ──────────────────────►│            │
  │                 │    type: approval                │               │            │
  │                 │    assignee: branch_manager      │               │            │
  │                 │    amount: 75M                   │               │            │
  │                 │                  │               │              │            │
  │◄── 202 Accepted ─│                  │               │              │            │
  │   {status:       │                  │               │              │            │
  │    pending_       │                  │               │              │            │
  │    approval}     │                  │               │              │            │
  │                 │                  │               │              │            │
  │                  ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─  │            │
  │                 │                  │               │              │            │
  │            (branch_manager approves via workflow7 UI)            │            │
  │                 │                  │               │              │            │
  │                 │ 7. workflow7 callback ────────────────────────►│            │
  │                 │    {action: approve,             │               │            │
  │                 │     approved_by: manager-uuid}    │               │            │
  │                 │                  │               │              │            │
  │                 │ 8. UPDATE transaction              │              │            │
  │                 │    status: APPROVED              │              │            │
  │                 │                  │               │              │            │
  │                 │ 9. Execute transaction            │              │            │
  │                 │    status: COMPLETED             │              │            │
  │                 │                  │               │              │            │
  │◄── Notification ─│                 │               │              │            │
  │   "Transaction    │                  │               │              │            │
  │    approved"      │                  │               │              │            │
```

### 15.3 Flow: Supervisor Switch Branch dan Buat Transaksi

John (supervisor @ KC Bandung, teller @ KCP Dago) switch ke KCP Dago lalu buat transaksi.

```
Browser              auth7-ui          auth7-svc          core7-enterprise
  │                     │                  │                    │
  │ 1. POST /auth/     │                  │                    │
  │    switch-branch   │                  │                    │
  │    {target: KCP-   │                  │                    │
  │     Dago,          │                  │                    │
  │     password:xxx}  │                  │                    │
  │────────────────────►│                  │                    │
  │                     │ 2. Verify password (re-auth)          │
  │                     │                  │                    │
  │                     │ 3. Validate branch access           │
  │                     │    user_branch_assignments:          │
  │                     │    KC-BDG (teller), KCP-Dago (teller)│
  │                     │    KC-JKT (supervisor)               │
  │                     │    → KCP-Dago is assigned → OK      │
  │                     │                  │                    │
  │                     │ 4. Update session                   │
  │                     │    active_branch_id = KCP-Dago      │
  │                     │                  │                    │
  │                     │ 5. Get role for KCP-Dago            │
  │                     │    user_roles WHERE                 │
  │                     │    user=john AND branch=KCP-Dago    │
  │                     │    → role: teller                   │
  │                     │                  │                    │
  │                     │ 6. Issue NEW access_token:           │
  │◄────────────────────│    {branch_id: KCP-Dago,            │
  │   {access_token,    │     roles: ["teller"],              │
  │    session_id}     │     permissions: ["account:read",   │
  │                     │       "transaction:create"]}        │
  │                     │                  │                    │
  │ 7. POST /transactions                  │                    │
  │    {amount: 8M}    │                  │                    │
  │───────────────────────────────────────────────────────────►│
  │                     │                  │                    │
  │                     │    [core7 extracts from NEW token]   │
  │                     │    role: teller                      │
  │                     │    branch: KCP-Dago                  │
  │                     │    permissions: transaction:create   │
  │                     │                  │                    │
  │                     │    [core7: teller limit = 10M]       │
  │                     │    8M < 10M → AMOUNT OK              │
  │                     │                  │                    │
  │                     │    [core7: approval threshold]       │
  │                     │    8M < 50M → NO APPROVAL NEEDED     │
  │                     │                  │                    │
  │◄── 201 Created ─────────────────────────────────────────── │
  │   {transaction_id, │                  │                    │
  │    status: COMPLETED}                 │                    │
```

**Perhatikan**: Saat switch branch, role berubah dari `supervisor` → `teller`.
Limit transaksi berubah: 100M → 10M. Ini semua karena claims di JWT berubah.

### 15.4 Flow: Teller Akses Data Rekening (Field Masking)

Teller lihat rekening — boleh lihat data nasabah, TAPI saldo dan limit di-mask.

```
Browser         bos7-portal      core7-enterprise      auth7-svc
  │                 │                  │                    │
  │ GET /accounts   │                  │                    │
  │────────────────►│                  │                    │
  │                 │ 1. JWT claims:   │                    │
  │                 │    role: teller   │                    │
  │                 │    branch: KC-BDG │                    │
  │                 │    permissions:   │                    │
  │                 │      account:read │                    │
  │                 │    field_mask:    │                    │
  │                 │      {denied: [balance, limit]}       │
  │                 │                  │                    │
  │                 │ 2. [LAYER 1] Menu check               │
  │                 │    menu:accounts → YES                 │
  │                 │                  │                    │
  │                 │ 3. [LAYER 2] Permission check          │
  │                 │    account:read → YES                  │
  │                 │                  │                    │
  │                 │ 4. [LAYER 3] Branch scope              │
  │                 │    scope=own_branch → KC-BDG            │
  │                 │    → query WHERE branch_id = KC-BDG     │
  │                 │                  │                    │
  │                 │ 5. [LAYER 4] Field masking              │
  │                 │    For each account in response:        │
  │                 │      account.balance = 0   ← zero-out │
  │                 │      account.limit = 0     ← zero-out │
  │                 │                  │                    │
  │◄── 200 OK ─────│                  │                    │
  │   [{"account_number": "123456",    │                    │
  │     "customer_name": "John Doe", │                    │
  │     "branch": "KC-BDG",          │                    │
  │     "balance": 0,    ← masked    │                    │
  │     "limit": 0},     ← masked    │                    │
  │    {"account_number": "789012",   │                    │
  │     ...}]                        │                    │
```

### 15.5 Flow: Daily Cumulative Limit Exceeded

Teller sudah transfer 5x Rp 9M = Rp 45M (limit harian Rp 50M). Coba transfer lagi.

```
Browser         bos7-portal      core7-enterprise      policy7
  │                 │                  │                    │
  │ POST /trans-   │                  │                    │
  │ actions         │                  │                    │
  │ {amount: 9M}    │                  │                    │
  │────────────────►│                  │                    │
  │                 │ 1. JWT: role=teller, branch=KC-BDG  │
  │                 │                  │                    │
  │                 │ 2. [AUTH7 LAYERS] → all PASS            │
  │                 │    menu: YES, permission: YES, scope: OK │
  │                 │                  │                    │
  │                 │ 3. [POLICY7: Per-transaction limit]  │
  │                 │    GET policy7 /v1/params/limits       │
  │                 │      ?role=teller&type=transfer       │
  │                 │◄─────────────────────────────────────│
  │                 │    {max_amount: 10M}                │
  │                 │    9M < 10M → OK                    │
  │                 │                  │                    │
  │                 │ 4. [POLICY7: Daily cumulative check] │
  │                 │    SELECT SUM(amount) FROM transactions │
  │                 │    WHERE user_id = teller1             │
  │                 │      AND branch_id = KC-BDG            │
  │                 │      AND date = today                   │
  │                 │    → cumulative = 45M                    │
  │                 │    GET policy7 /v1/params/limits         │
  │                 │      ?role=teller&type=transfer&field=daily_limit │
  │                 │◄─────────────────────────────────────│
  │                 │    {daily_limit: 50M}               │
  │                 │    45M + 9M = 54M > 50M → DENY      │
  │                 │                  │                    │
  │◄── 403 Forbidden │                  │                    │
  │   "Daily transaction │              │                    │
  │    limit exceeded │              │
  │    (Rp 50M)"     │              │                    │
```

### 15.6 Flow: Inter-Branch Transfer perlu Head Office Approval

Branch Manager transfer Rp 200M antar cabang (KC Bandung → KC Jakarta).

```
Browser         bos7-portal      core7-enterprise  auth7-svc  workflow7
  │                 │                  │               │          │
  │ POST /transfer │                  │               │          │
  │ {from: KC-BDG, │                  │               │          │
  │  to: KC-JKT,   │                  │               │          │
  │  amount: 200M} │                  │               │          │
  │────────────────►│                  │               │          │
  │                 │ 1. JWT: role=branch_manager     │          │
  │                 │    branch=KC-BDG                 │          │
  │                 │                  │               │          │
  │                 │ 2. [AUTH7 LAYERS] → all PASS    │          │
  │                 │    permission: transaction:create│          │
  │                 │    scope: assigned_branches      │          │
  │                 │                  │               │          │
  │                 │ 3. [CORE7: Limit]                │          │
  │                 │    200M < 500M (BM limit) → OK  │          │
  │                 │                  │               │          │
  │                 │ 4. [CORE7: Inter-branch rule]   │          │
  │                 │    is_cross_branch = YES          │          │
  │                 │    amount > 100M                  │          │
  │                 │    → requires_approval: HEAD_OFFICE│          │
  │                 │                  │               │          │
  │                 │ 5. INSERT transaction              │          │
  │                 │    status: PENDING_HO_APPROVAL    │          │
  │                 │                  │               │          │
  │                 │ 6. CREATE workflow7 task ───────────────────►│
  │                 │    assignee: HO Finance Director │               │
  │                 │    amount: 200M                   │               │
  │                 │    type: inter_branch_transfer    │               │
  │                 │                  │               │          │
  │◄── 202 Accepted │                  │               │          │
  │                 │                  │               │          │
  │              ... (HO Director approves via workflow7-web) ...    │
  │                 │                  │               │          │
  │                 │ 7. workflow7 callback ────────────────────────│
  │                 │    {action: approve}             │               │
  │                 │                  │               │          │
  │                 │ 8. EXECUTE transfer              │               │
  │                 │    Debit: KC-BDG -200M           │               │
  │                 │    Credit: KC-JKT +200M          │               │
  │                 │    status: COMPLETED              │               │
  │                 │                  │               │          │
```

### 15.7 Flow: Auditor Lihat Semua Cabang (No Field Mask)

Auditor — bisa lihat semua data di semua cabang, read-only.

```
Browser         bos7-portal      core7-enterprise      auth7-svc
  │                 │                  │                    │
  │ GET /accounts   │                  │                    │
  │ ?branch=all     │                  │                    │
  │────────────────►│                  │                    │
  │                 │ 1. JWT: role=auditor               │
  │                 │    scope=all_branches               │
  │                 │    field_mask={} (no mask)          │
  │                 │                  │                    │
  │                 │ 2. [LAYER 1] menu:accounts → YES    │
  │                 │ 3. [LAYER 2] account:read → YES     │
  │                 │ 4. [LAYER 3] scope=all_branches     │
  │                 │    → NO branch filter in SQL         │
  │                 │    → SELECT * FROM accounts          │
  │                 │      WHERE org_id = bank-uuid       │
  │                 │ 5. [LAYER 4] field_mask = {}        │
  │                 │    → semua field ditampilkan          │
  │                 │                  │                    │
  │◄── 200 OK ─────│                  │                    │
  │   [all accounts │                  │                    │
  │    from ALL     │                  │                    │
  │    branches,    │                  │                    │
  │    full data]   │                  │                    │
  │                 │                  │                    │
  │                 │    [AUDITOR TIDAK BISA CREATE/UPDATE]│
  │                 │    karena tidak punya               │
  │                 │    account:write permission          │
```

---

> Semua open questions telah dijawab di [OPEN-QUESTIONS.md](../OPEN-QUESTIONS.md).

*Prev: [03-oauth2-oidc.md](./03-oauth2-oidc.md) | Next: [05-session-token.md](./05-session-token.md)*