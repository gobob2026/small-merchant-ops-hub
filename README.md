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
- Server analytics: summary KPI + repurchase follow-up list endpoint
- Client: Nuxt 4 flow for create member, create order, create campaign, and monitor repurchase KPI
- Admin: dedicated merchant operations page wired to backend APIs (member/order/campaign/follow-up)
- Automation: release workflow, issue templates, PR template
