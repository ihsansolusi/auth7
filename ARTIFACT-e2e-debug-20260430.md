# auth7 E2E Test Debug Session — 2026-04-30

## Goal
Fix auth7 login returning "invalid credentials" despite correct password.

## Root Cause Found (FINAL)

**Bug location**: `internal/service/password/hasher.go` lines 76 and 82

**Problem**: Off-by-one index error when parsing argon2 hash parts.

```go
// WRONG (before fix)
if _, err := fmt.Sscanf(parts[1], "v=%d", &version); err != nil {  // parts[1] = "argon2id"
    return false
}
if _, err := fmt.Sscanf(parts[2], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {  // parts[2] = "v=19"

// CORRECT (after fix)
if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {  // parts[2] = "v=19"
    return false
}
if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {  // parts[3] = "m=65536,t=3,p=4"
```

**Hash format** (6 parts split by `$`):
```
parts[0] = ""           // empty
parts[1] = "argon2id"  // algorithm
parts[2] = "v=19"       // version
parts[3] = "m=65536,t=3,p=4"  // params
parts[4] = "H7rI6yKAwFHrqn/pY1YjNw"  // salt (base64)
parts[5] = "j7+H1jqPziicUKcfSWfCbjBvsM5AUKLa2nqla8zDUzc"  // hash (base64)
```

## What Was Done

1. **Register works** - POST /v1/auth/register returns 201 with user + verify_token
2. **User activated manually** - set status='active' in DB for testuser
3. **Hasher verified correct** - debug test confirmed argon2 recomputation matches stored hash
4. **Bug identified** - wrong array index in Verify() parsing
5. **Bug fixed** - changed `parts[1]`→`parts[2]` and `parts[2]`→`parts[3]`
6. **Test passes** - TestVerifyStoredHash now returns true

## Current State

- **auth7 binary rebuilt** with fix applied
- **Build**: OK
- **auth7 process killed** - needs restart
- **Login NOT yet tested** after fix

## What Needs To Be Done Next

1. Restart auth7 service with new binary
2. Test login: `curl -X POST http://localhost:8090/v1/auth/login -H "Content-Type: application/json" -d '{"org_id":"00000000-0000-0000-0000-000000000001","username":"testuser","password":"Test1234!"}'`
3. Verify login returns JWT token (not "invalid credentials")
4. Clean up debug test files created:
   - `internal/service/password/test_verify_test.go`
   - `internal/service/password/debug_test.go`

## Test User Credentials

| Field | Value |
|-------|-------|
| Username | testuser |
| Password | Test1234! |
| Org ID | 00000000-0000-0000-0000-000000000001 |
| Status | active (manually set) |
| User ID | 019ddbe3-c5c9-7d56-a399-82755be93d80 |

## Files Changed

| File | Change |
|------|--------|
| `internal/service/password/hasher.go` | Fixed index parsing bug |

## Files To Delete (debug artifacts)

```
internal/service/password/test_verify_test.go
internal/service/password/debug_test.go
```
