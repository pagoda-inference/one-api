# Org Migration Runbook (P1)

## Goal
Enable org/department/team governance without impacting existing model routing behavior.

## Safety Guarantees
- Existing model routing path is unchanged.
- Existing tenant/model visibility logic remains compatible.
- New permissions are behind `ORG_MEMBERSHIP_V2_ENABLED` (default: `false`).
- Auto org bootstrap is disabled by default (`ORG_AUTO_BOOTSTRAP_ENABLED=false`).

## New Data Objects
- `users.company_id`
- `users.department_id`
- `users.org_source`
- `user_org_memberships`

## New Admin APIs
- `GET /api/admin/org/config`
- `POST /api/admin/org/migrate-users`
  - body: `{"dry_run": true, "limit": 100000}`

## Migration Steps
1. Deploy code with defaults (all org-v2 features still OFF).
2. Run dry-run:
   - `POST /api/admin/org/migrate-users` with `dry_run=true`
3. Inspect report:
   - total/updated/skipped/failed/examples
4. Execute migration:
   - `POST /api/admin/org/migrate-users` with `dry_run=false`
5. Verify spot checks:
   - sample users have company/department values
   - `user_org_memberships` has active rows
6. Enable v2 permission model:
   - set `ORG_MEMBERSHIP_V2_ENABLED=true`

## Permission Model (Org-V2)
- `department_admin`: can manage teams/members within own department scope.
- `team_admin`: can manage team/members within own department scope.
- target-user write operations enforce same-department check.

## Login-Time Org Assignment
- Password login/register -> source `password` -> default external pool if user has no explicit org.
- Lark login/bind -> source `lark` -> default formal pool if user has no explicit org.
- If user already has explicit org values, resolver preserves them and only backfills membership.

## Rollback
- Set `ORG_MEMBERSHIP_V2_ENABLED=false`.
- Keep migrated data (backward compatible).
- Existing legacy owner/admin permission path continues working.

