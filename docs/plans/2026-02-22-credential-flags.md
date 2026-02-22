# Add --user/--password flags and ES_USER/ES_PASSWORD environment variables

## Overview
- **Problem**: Passwords containing `#` (and other URL-special characters like `?`, `%`) break when passed via URI because Go's `url.Parse()` treats `#` as fragment delimiter. Example: `https://root:op0107##@host:9200` gets truncated to `https://root:op0107`.
- **Solution**: Add `--user` and `--password` CLI flags plus `ES_USER`/`ES_PASSWORD` environment variable support as alternative credential sources that bypass URL parsing entirely.
- **Priority order**: `--user`/`--password` flags > `ES_USER`/`ES_PASSWORD` env vars > URI-embedded credentials
- **Backward compatible**: existing URI-embedded credentials continue to work

## Context
- **Primary file**: `cmd/epm/main.go` — flag parsing, `parseESURI()`, credential extraction, `ClientConfig` assembly
- **Test file**: `cmd/epm/main_test.go` — table-driven `TestParseESURI`
- Current flow: `flag.Args()[0]` → `parseESURI()` → extracts username/password from `url.Parse()` → `ClientConfig`
- `parseESURI` already strips credentials from returned baseURL and handles percent-encoded passwords correctly
- `ClientConfig` in `internal/client/client.go` accepts plain `Username`/`Password` strings — no changes needed there

## Development Approach
- **Testing approach**: Regular (code first, then tests)
- Small, focused changes to `cmd/epm/main.go` only (+ tests)
- No changes to `internal/client/` — credential injection point (`ClientConfig`) stays the same
- Maintain backward compatibility with URI-embedded credentials

## Implementation Steps

### Task 1: Add --user and --password flags
- [x] Add `user` and `password` flag declarations alongside existing flags in `main()` (lines 66-70)
- [x] After `parseESURI()` call (line 116), add credential resolution logic with priority: flags > env > URI
  ```
  // Resolve credentials: flags > env vars > URI-embedded
  finalUser, finalPass := username, password  // from parseESURI (URI-embedded)
  if envUser := os.Getenv("ES_USER"); envUser != "" { finalUser = envUser }
  if envPass := os.Getenv("ES_PASSWORD"); envPass != "" { finalPass = envPass }
  if *userFlag != "" { finalUser = *userFlag }
  if *passFlag != "" { finalPass = *passFlag }
  ```
- [x] Use `finalUser`/`finalPass` in the `ClientConfig` and HTTP warning check instead of `username`/`password`
- [x] Update `flag.Usage` help text to document new flags and env vars
- [x] Write tests for flag override of URI credentials (new test function `TestCredentialResolution`)
- [x] Write tests for env var override of URI credentials (use `t.Setenv`)
- [x] Write tests for priority chain: flag > env > URI
- [x] Run tests — must pass before next task

### Task 2: Verify acceptance criteria
- [ ] Verify: `epm --user root --password "op0107##" https://host:9200` works (no URL parse error)
- [ ] Verify: `ES_PASSWORD="op0107##" epm https://host:9200` works
- [ ] Verify: URI-embedded credentials still work: `epm http://elastic:changeme@host:9200`
- [ ] Verify: flag overrides URI: `epm --password new http://elastic:old@host:9200` uses "new"
- [ ] Run full test suite (`make test`)
- [ ] Run linter (`make lint`)

### Task 3: Update documentation
- [ ] Update CLAUDE.md usage section with new flags and env var examples
- [ ] Add credential examples to the examples section showing all three methods

## Technical Details

**New flags** (in `main()`):
```go
userFlag = flag.String("user", "", "Elasticsearch username (overrides URI and ES_USER)")
passFlag = flag.String("password", "", "Elasticsearch password (overrides URI and ES_PASSWORD)")
```

**Credential resolution order** (each layer overrides the previous):
1. URI-embedded (`https://user:pass@host`) — parsed by `parseESURI()`
2. Environment variables `ES_USER` / `ES_PASSWORD`
3. CLI flags `--user` / `--password`

**Updated usage output**:
```
epm v0.x — Elasticsearch Performance Monitor

usage:
  epm [flags] <elasticsearch-uri>

examples:
  epm http://localhost:9200
  epm --insecure https://elastic:changeme@prod.example.com:9200
  epm --user root --password "s3cr#t!" https://host:9200
  epm --interval 30s http://localhost:9200

environment variables:
  ES_USER       Elasticsearch username (overridden by --user flag)
  ES_PASSWORD   Elasticsearch password (overridden by --password flag)

flags:
  ...
```

## Post-Completion

**Manual verification**:
- Test with actual ES cluster using password containing `#`: `epm --insecure --user root --password "op0107##" https://logs-es-app-aws-1.h.ecentria.com:9200/`
- Test env var approach: `ES_USER=root ES_PASSWORD="op0107##" epm --insecure https://logs-es-app-aws-1.h.ecentria.com:9200/`
