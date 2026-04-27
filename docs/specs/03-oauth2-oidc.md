# Auth7 — Spec 03: OAuth2/OIDC

> **Versi**: 1.0-draft | **Tanggal**: 2026-04-22 | **Fase**: Brainstorming
> **Analogi**: Ory Hydra

---

## 1. OAuth2 dalam Konteks Core7

Auth7 berfungsi sebagai **Authorization Server** (AS) dalam ekosistem OAuth2:

```
┌──────────────────────────────────────────────────────────┐
│  OAuth2 Roles                                            │
│                                                          │
│  Resource Owner  = User (manusia yang punya akun)        │
│  Client          = Aplikasi (bos7-portal, mobile app)    │
│  Authorization   = auth7-svc                             │
│  Server (AS)                                             │
│  Resource Server = core7 services (workflow7, dll)       │
└──────────────────────────────────────────────────────────┘
```

### Flows yang Didukung
| Flow | Use Case | Status v1.0 |
|---|---|---|
| Authorization Code + PKCE | Browser apps (bos7-portal, auth7-ui) | ✅ |
| Client Credentials | M2M (service-to-service) | ✅ |
| Refresh Token | Semua flows | ✅ |
| Device Authorization | IoT / CLI tools | 🔲 v1.1 |
| Implicit | Legacy (deprecated) | ❌ disabled |
| Password Grant | Legacy (deprecated) | ❌ disabled |

---

## 2. OAuth2 Flows

### 2.1 Authorization Code Flow + PKCE (Primary)

Untuk browser apps (bos7-portal, workflow7-web, dll):

```
Client App              auth7-ui                auth7-svc
  │                        │                       │
  │  1. Generate PKCE      │                       │
  │     code_verifier      │                       │
  │     code_challenge     │                       │
  │                        │                       │
  │  2. Redirect ke        │                       │
  │     /oauth2/authorize  │                       │
  │     ?client_id=        │                       │
  │     &redirect_uri=     │                       │
  │     &response_type=    │                       │
  │     code               │                       │
  │     &scope=openid      │                       │
  │     profile email      │                       │
  │     roles              │                       │
  │     &state=xyz         │                       │
  │     &code_challenge=   │                       │
  │     &code_challenge_   │                       │
  │     method=S256        │                       │
  │───────────────────────────────────────────────►│
  │                        │                       │
  │  3. Verify client      │                       │
  │     + redirect_uri     │                       │
  │     + create auth code │                       │
  │     (TTL: 5 menit)     │                       │
  │                        │                       │
  │  4. Redirect ke        │                       │
  │     auth7-ui/login     │                       │
  │     (jika belum login) │                       │
  │◄───────────────────────│                       │
  │                        │                       │
  │  5. User login         │                       │
  │     (username+pass+MFA)│                       │
  │───────────────────────────────────────────────►│
  │                        │                       │
  │  6. Redirect ke        │                       │
  │     redirect_uri       │                       │
  │     ?code=abc123       │                       │
  │     &state=xyz         │                       │
  │◄───────────────────────│                       │
  │                        │                       │
  │  7. POST /oauth2/token │                       │
  │     grant_type=        │                       │
  │     authorization_code │                       │
  │     code=abc123        │                       │
  │     code_verifier=v    │                       │
  │     redirect_uri=...   │                       │
  │───────────────────────────────────────────────►│
  │                        │                       │
  │  8. Verify code        │                       │
  │     + PKCE verifier    │                       │
  │     + issue tokens     │                       │
  │                        │                       │
  │  9. Return tokens      │                       │
  │     {access_token,     │                       │
  │      refresh_token,    │                       │
  │      id_token}         │                       │
  │◄───────────────────────│                       │
```

### 2.2 Client Credentials Flow (M2M)

Untuk service-to-service communication:

```
POST /oauth2/token
Content-Type: application/x-www-form-urlencoded
Authorization: Basic base64(client_id:client_secret)

grant_type=client_credentials
&scope=service:read service:write
```

- Tidak ada PKCE (confidential client)
- Tidak ada refresh token
- Access token TTL: 15 menit
- Scope terbatas pada service permissions

### 2.3 Refresh Token Flow

```
POST /oauth2/token
Content-Type: application/x-www-form-urlencoded
Authorization: Basic base64(client_id:client_secret)

grant_type=refresh_token
&refresh_token=xyz
```

- Verify refresh token (valid, belum revoked, belum expired)
- Issue new access token + optional new refresh token (rotation)
- TTL access token: 15 menit
- TTL refresh token: 8 jam (jam kerja)

---

## 3. Token Design

### 2.1 Access Token (JWT)

```json
{
  "iss": "https://auth7.bank.co.id",
  "sub": "user-uuid",
  "aud": "client-id",
  "exp": 1713801600,
  "iat": 1713798000,
  "jti": "token-uuid",
  "org_id": "org-uuid",
  "branch_id": "branch-uuid",       // active branch (bisa berubah via switch-branch)
  "roles": ["teller", "supervisor"],
  "permissions": ["account:read", "transaction:create"],
  "mfa_verified": true,
  "token_type": "access"
}
```

- **Algorithm**: RS256
- **TTL**: 15 menit (banking-grade)
- **Format**: JWT (default) atau Opaque Token (configurable per client)

### 2.2 Refresh Token

```json
{
  "iss": "https://auth7.bank.co.id",
  "sub": "user-uuid",
  "aud": "client-id",
  "exp": 1713826800,
  "iat": 1713798000,
  "jti": "token-uuid",
  "session_id": "session-uuid",
  "token_type": "refresh"
}
```

- **TTL**: 8 jam (expire di akhir jam kerja)
- **Rotation**: Ya (issue new refresh token setiap refresh)
- **Reuse detection**: Jika refresh token dipakai 2x → revoke semua session

### 2.3 ID Token (OIDC)

```json
{
  "iss": "https://auth7.bank.co.id",
  "sub": "user-uuid",
  "aud": "client-id",
  "exp": 1713801600,
  "iat": 1713798000,
  "nonce": "client-nonce",
  "name": "John Doe",
  "email": "john@bank.co.id",
  "email_verified": true,
  "org_id": "org-uuid",
  "branch_id": "branch-uuid",       // active branch saat token di-issue
  "roles": ["teller", "supervisor"]
}
```

---

## 4. Token Format: JWT + Opaque

| Format | Use Case | Revocation |
|---|---|---|
| **JWT** | Stateless verification (default, zero latency) | Via introspection + TTL pendek |
| **Opaque** | High-security scenarios (instant revocation) | Direct DB lookup |

Client bisa request format saat register:

```json
{
  "token_format": "jwt"  // atau "opaque"
}
```

---

## 5. OAuth2 Client

### 4.1 Client Entity

```go
type OAuth2Client struct {
    ID                string
    Name              string
    OrgID             uuid.UUID
    ClientType        ClientType     // confidential, public
    ClientSecret      string         // hashed (confidential only)
    RedirectURIs      []string
    AllowedScopes     []string
    AllowedGrantTypes []GrantType
    RequirePKCE       bool
    TokenFormat       TokenFormat    // jwt, opaque
    AccessTokenTTL    int            // detik (default: 900)
    RefreshTokenTTL   int            // detik (default: 28800)
    Status            ClientStatus   // active, inactive
    CreatedAt         time.Time
    UpdatedAt         time.Time
}
```

### 4.2 Scopes yang Didukung

```
openid          # Required untuk OIDC (mendapatkan ID Token)
profile         # username, full_name
email           # email address
offline_access  # Mengizinkan refresh token
roles           # User roles dalam org
permissions     # User permissions (careful: bisa besar)

# Core7-specific scopes
workflow7:read   # Read access ke workflow7
workflow7:write  # Write access ke workflow7
notif7:read      # Read access ke notif7
notif7:write     # Write access ke notif7
```

### 4.3 Dynamic Client Registration (RFC 7591)

```
POST /oauth2/register
Content-Type: application/json

{
  "client_name": "My App",
  "redirect_uris": ["https://myapp.bank.co.id/callback"],
  "grant_types": ["authorization_code", "refresh_token"],
  "response_types": ["code"],
  "scope": "openid profile email roles",
  "token_endpoint_auth_method": "client_secret_basic"
}

Response:
{
  "client_id": "auto-generated-uuid",
  "client_secret": "auto-generated-secret",
  "client_id_issued_at": 1713798000,
  "client_secret_expires_at": 0,
  "redirect_uris": ["https://myapp.bank.co.id/callback"],
  ...
}
```

---

## 6. OIDC Discovery

### 5.1 `.well-known/openid-configuration`

```json
{
  "issuer": "https://auth7.bank.co.id",
  "authorization_endpoint": "https://auth7.bank.co.id/oauth2/authorize",
  "token_endpoint": "https://auth7.bank.co.id/oauth2/token",
  "userinfo_endpoint": "https://auth7.bank.co.id/oauth2/userinfo",
  "jwks_uri": "https://auth7.bank.co.id/.well-known/jwks.json",
  "registration_endpoint": "https://auth7.bank.co.id/oauth2/register",
  "scopes_supported": ["openid", "profile", "email", "roles"],
  "response_types_supported": ["code"],
  "grant_types_supported": ["authorization_code", "refresh_token", "client_credentials"],
  "subject_types_supported": ["public"],
  "id_token_signing_alg_values_supported": ["RS256"],
  "token_endpoint_auth_methods_supported": ["client_secret_basic", "client_secret_post"],
  "code_challenge_methods_supported": ["S256"]
}
```

### 5.2 JWKS Endpoint

```
GET /.well-known/jwks.json

{
  "keys": [
    {
      "kty": "RSA",
      "use": "sig",
      "kid": "key-uuid",
      "alg": "RS256",
      "n": "base64url(modulus)",
      "e": "base64url(exponent)"
    }
  ]
}
```

- Key rotation: generate new key pair setiap 90 hari
- Old key tetap di JWKS selama TTL token terpanjang (8 jam)

---

## 7. Consent Screen

- **v1.0**: Tidak ada (internal clients auto-approve)
- **v2.0**: Consent screen untuk third-party clients

---

## 8. Branch Context in Tokens

Karena user bisa punya akses multi-branch, token JWT selalu mengandung `branch_id` yang merepresentasikan **active branch** saat ini.

- `access_token.branch_id` = branch yang sedang aktif (bisa berubah via `/auth/switch-branch`)
- `id_token.branch_id` = branch saat token di-issue (info saja, tidak dipakai untuk authorization)
- Saat switch branch: issue **new access_token** dengan `branch_id` baru, refresh_token tetap valid
- Permission/role di-derive dari kombinasi `user_id + org_id + branch_id` (bukan `user_id + org_id` saja)

**Contoh: User John di KC Bandung (supervisor) vs KCP Dago (teller):**

```
# Token saat aktif di KC Bandung
{
  "sub": "john-uuid",
  "org_id": "bank-uuid",
  "branch_id": "kc-bandung-uuid",
  "roles": ["supervisor"],
  "permissions": ["account:read", "account:write", "transaction:approve"]
}

# Token saat switch ke KCP Dago
{
  "sub": "john-uuid",
  "org_id": "bank-uuid",
  "branch_id": "kcp-dago-uuid",
  "roles": ["teller"],
  "permissions": ["account:read", "transaction:create"]
}
```

---

> Semua open questions telah dijawab di [OPEN-QUESTIONS.md](../OPEN-QUESTIONS.md).

*Prev: [02-identity.md](./02-identity.md) | Next: [04-authorization.md](./04-authorization.md)*
