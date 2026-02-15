# small-merchant-ops-hub

## Problem
Small merchants need a low-cost private-domain operations hub to manage members, orders, campaigns, and repurchase metrics without expensive SaaS lock-in.

## Architecture
- server: Go + Gin + GORM, sqlite(local) / pgsql(production), local cache(local) / redis(production)
- admin: copied from root art-design-pro
- client: Nuxt 4

## Milestones
1. API and persistence
2. Admin workflow integration
3. Client workflow integration
4. Automated release and templates

## Current Scope Delivered
- Server: member/order/campaign APIs with sqlite/pgsql and local/redis factories
- Server analytics: summary KPI + repurchase follow-up + campaign attribution report + CSV export
- Server auth/system endpoints: `/api/auth/login`, `/api/auth/logout`, `/api/auth/refresh`, `/api/user/info`, `/api/user/list`, `/api/role/list`
- Server backend-mode menu endpoint: `/api/v3/system/menus`
- Client: Nuxt 4 flow for create member, create order, create campaign, and monitor repurchase KPI
- Admin: dedicated merchant operations page wired to backend APIs (member/order/campaign/follow-up/report)
- Admin auth lifecycle: 401 responses now trigger one-time token refresh via `/api/auth/refresh`, then retry the original request
- Permissions: operations page uses route meta + button auth marks (`member:create`, `order:create`, `campaign:create`, `followup:view`, `report:export`), and supports `R_USER` read-only access
- Automation: release workflow, issue templates, PR template
