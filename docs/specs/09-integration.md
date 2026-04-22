# Auth7 — Spec 09: Integration

> **Versi**: 1.0-draft | **Tanggal**: 2026-04-22 | **Fase**: Brainstorming

---

## 1. Integration Patterns

### 1.1 Protected Services

Services dalam ekosistem Core7 memverifikasi token via:

| Method | Deskripsi | Latency |
|---|---|---|
| **JWT Verification** | Verify via JWKS public key (stateless) | Zero |
| **Introspection Endpoint** | POST /oauth2/introspect (real-time) | ~10ms |
| **gRPC AuthCheck** | Inter-service communication | ~5ms |

### 1.2 Recommended Pattern

```
Client App → Verify JWT locally (JWKS) → If expired → Introspect/gRPC
```

---

## 2. lib7-auth-go

### 2.1 Overview

- **Repo**: `ihsansolusi/lib7-auth-go` (terpisah dari auth7)
- **Konsisten** dengan pola `lib7-service-go`
- Menyediakan Go client untuk auth7 integration

### 2.2 Features

```go
// JWT Verification
verifier := auth.NewJWTVerifier(jwksURL)
claims, err := verifier.Verify(ctx, token)

// Introspection
client := auth.NewIntrospectionClient(auth7URL, clientID, clientSecret)
tokenInfo, err := client.Introspect(ctx, token)

// gRPC AuthCheck
conn, _ := grpc.Dial(auth7GRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
authClient := pb.NewAuthServiceClient(conn)
resp, err := authClient.Authenticate(ctx, &pb.AuthenticateRequest{Token: token})

// Permission Check
authzClient := pb.NewAuthzServiceClient(conn)
resp, err := authzClient.CheckPermission(ctx, &pb.CheckPermissionRequest{
    Subject: userID,
    Resource: "account",
    Action: "read",
})
```

### 2.3 Middleware

```go
// Gin middleware
router.Use(auth7.GinMiddleware(verifier, authzClient))

// Route-level permission check
router.GET("/accounts", auth7.RequirePermission("account:read"), handler)
```

---

## 3. gRPC Service Definition

### 3.1 AuthService

```protobuf
syntax = "proto3";
package auth7.v1;

service AuthService {
  rpc Authenticate(AuthenticateRequest) returns (AuthenticateResponse);
  rpc IntrospectToken(IntrospectTokenRequest) returns (IntrospectTokenResponse);
}

message AuthenticateRequest {
  string token = 1;
}

message AuthenticateResponse {
  bool valid = 1;
  string user_id = 2;
  string org_id = 3;
  string branch_id = 4;
  repeated string roles = 5;
  repeated string permissions = 6;
  bool mfa_verified = 7;
}

message IntrospectTokenRequest {
  string token = 1;
  string token_type_hint = 2;
}

message IntrospectTokenResponse {
  bool active = 1;
  string client_id = 2;
  string user_id = 3;
  string scope = 4;
  int64 exp = 5;
  int64 iat = 6;
}
```

### 3.2 AuthzService

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

message ListPermissionsRequest {
  string subject = 1;
}

message ListPermissionsResponse {
  repeated string permissions = 1;
}
```

### 3.3 TenantService

```protobuf
service TenantService {
  rpc GetOrg(GetOrgRequest) returns (GetOrgResponse);
  rpc GetBranch(GetBranchRequest) returns (GetBranchResponse);
}

message GetOrgRequest {
  string org_id = 1;
}

message GetOrgResponse {
  string id = 1;
  string code = 2;
  string name = 3;
}

message GetBranchRequest {
  string branch_id = 1;
}

message GetBranchResponse {
  string id = 1;
  string code = 2;
  string name = 3;
  string branch_type = 4;
}
```

---

## 4. Per-Service Integration

### 4.1 workflow7-svc

```go
// cmd/server/main.go
verifier := auth.NewJWTVerifier(cfg.Auth7JWKSURL)
authzClient := pb.NewAuthzServiceClient(auth7Conn)

router.Use(auth7.GinMiddleware(verifier, authzClient))

router.GET("/flows", auth7.RequirePermission("flow:read"), flowHandler.List)
router.POST("/flows", auth7.RequirePermission("flow:create"), flowHandler.Create)
```

### 4.2 notif7-svc

```go
// M2M communication
client := auth.NewClientCredentialsClient(auth7TokenURL, clientID, clientSecret)
token, err := client.GetToken(ctx, "service:read service:write")

// Use token for API calls
req, _ := http.NewRequest("GET", notif7URL+"/inbox", nil)
req.Header.Set("Authorization", "Bearer "+token.AccessToken)
```

### 4.3 bos7-portal (Next.js)

```typescript
// middleware.ts
import { NextResponse } from 'next/server'
import { jwtVerify } from 'jose'

export async function middleware(request: NextRequest) {
  const token = request.cookies.get('access_token')?.value

  if (!token) {
    return NextResponse.redirect(buildLoginURL(request))
  }

  try {
    const payload = await jwtVerify(token, JWKS_PUBLIC_KEY)
    return NextResponse.next()
  } catch {
    return NextResponse.redirect(buildLoginURL(request))
  }
}
```

---

## 5. Webhook ke Notif7 (v1.1)

- v1.0: Tidak ada webhook
- v1.1: Webhook atau event streaming ke notif7 untuk critical security events
  - Login dari device baru
  - MFA reset
  - Account locked
  - Suspicious activity

---

## 6. Open Questions

1. **Apakah perlu SDK untuk bahasa lain (TypeScript, Python)?**
   → v1.0: Go only
   → v1.1: TypeScript SDK untuk frontend apps

2. **Apakah perlu GraphQL API untuk auth7?**
   → v1.0: REST + gRPC only
   → v2.0: Mungkin (jika ada demand)

---

*Prev: [08-data-model.md](./08-data-model.md) | Next: [10-security.md](./10-security.md)*
