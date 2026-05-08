# Stability Remediation Roadmap

## Scope
This roadmap covers the full stabilization and architecture alignment across:
- Anthropic gateway compatibility and tool-call reliability
- Quota leak prevention and settlement correctness
- Root account security baseline
- Model visibility + pagination correctness
- Authentication and org/team governance alignment
- Migration and observability hardening

## Principles
- Keep production behavior stable by default
- Deliver in rollback-safe, isolated commits
- Add tests before broad refactors
- Prefer compatibility mode first, feature mode by whitelist

## P0 (This Week)

### P0-1: Anthropic quota leak closure
**Goal**: `/anthropic/v1/messages` and `/v1/messages` must never return valid business response when key quota is insufficient.

**Requirements**
- Align with `RelayTextHelper` lifecycle:
  - pre-consume before upstream request
  - rollback on all failure paths
  - post-consume on successful responses
- Non-stream:
  - settle by exact `usage` tokens
- Stream:
  - if usage missing, fallback settlement with prompt-only estimate and warning logs

**Done Criteria**
- insufficient quota returns 403 before effective upstream completion
- no pre-consume leak on network/5xx/parse failures
- settlement completed on 200 responses

### P0-2: Root bootstrap security
- Replace default `root/123456`
- Require `INITIAL_ROOT_PASSWORD` or generate one-time random bootstrap password
- Never silently create weak credentials

### P0-3: Market pagination correctness
- Push team-visibility filtering into query layer
- Return true filtered `total`
- Ensure `limit/offset` works on filtered dataset

### P0-4: Anthropic thinking policy unification
- Replace conflicting double-write behavior with strategy mode:
  - `strict_compat` (default)
  - `pass_thinking` (whitelist model/channel)
- Preserve current stable behavior while enabling controlled rollout

## P1 (1-2 Weeks)

### P1-1: Visibility model normalization
- Introduce `model_visible_tenants` relation table
- Keep old string field as read compatibility during migration

### P1-2: Auth + Org + Team governance alignment
- Introduce canonical hierarchy:
  - User -> Company -> Department -> Team -> Role
- Add unified resolver for Feishu/email/phone onboarding
- Define default org pool for external users
- Expand team admin capability scope (not root-only)

### P1-3: Lark sync observability and retries
- Error aggregation by category (permission/id_type/timeout)
- dry-run mode
- retry queue for recoverable failures

### P1-4: Migration discipline
- Move one-off SQL operations to versioned migrations
- Remove temporary startup migration hacks

## P2 (After Stability Window)

### P2-1: Frontend data-layer consistency
- Unify query/pagination/error/empty-state handling for:
  - Usage
  - Teams
  - PlaygroundConfig

### P2-2: i18n completeness and CI checks
- Fill missing zh/en keys
- Add CI fail-fast on missing translations

### P2-3: Regression matrix expansion
- Cover full matrix for:
  - `/v1/chat/completions`
  - `/anthropic/v1/messages`
  - stream/non-stream
  - tool loop
  - thinking policy combinations

## Execution Order
1. P0-1 + P0-2 + P0-3 in isolated commits
2. P0-4 with targeted regression tests
3. P1 auth/org/team and visibility normalization in phased rollout
4. P2 engineering polish and CI hardening

## Rollout and Safety
- Feature flags for risky behavior changes
- Canary deployment before full rollout
- Keep each change rollbackable at commit level
