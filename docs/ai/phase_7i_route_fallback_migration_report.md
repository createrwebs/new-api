# Phase 7I — Route & Fallback Migration Report (Agent A)

## Executive Summary

Phase 7I introduces database schema updates and runtime configurations to support first-class Routes and Ordered Fallbacks without incurring downtime or breaking existing API key traffic.

---

## 1. Schema Modifications

### 1.1 New Table: `routes`
The table is automatically provisioned via GORM's `DB.AutoMigrate(&model.Route{})` during service startup:

```sql
CREATE TABLE IF NOT EXISTS `routes` (
    `id` INTEGER PRIMARY KEY AUTOINCREMENT,
    `name` VARCHAR(64) NOT NULL,
    `slug` VARCHAR(64) NOT NULL UNIQUE,
    `description` TEXT,
    `enabled` BOOLEAN DEFAULT TRUE,
    `kind` VARCHAR(32) DEFAULT 'llm',
    `routing_policy` VARCHAR(32) DEFAULT 'priority',
    `channel_ids` TEXT NOT NULL,
    `channel_tags` TEXT,
    `model_mapping` TEXT,
    `cost_multiplier` DECIMAL(5,4) DEFAULT 1.0000,
    `min_user_group` VARCHAR(32) DEFAULT '',
    `created_at` BIGINT,
    `updated_at` BIGINT,
    `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_routes_name` ON `routes` (`name`);
CREATE UNIQUE INDEX IF NOT EXISTS `idx_routes_slug` ON `routes` (`slug`);
CREATE INDEX IF NOT EXISTS `idx_routes_enabled` ON `routes` (`enabled`);
CREATE INDEX IF NOT EXISTS `idx_routes_kind` ON `routes` (`kind`);
CREATE INDEX IF NOT EXISTS `idx_routes_deleted_at` ON `routes` (`deleted_at`);
```

### 1.2 Updated Table: `tokens`
Two optional columns were appended to the `tokens` table via GORM AutoMigrate:

```sql
ALTER TABLE `tokens` ADD COLUMN `primary_route_id` INTEGER DEFAULT 0;
ALTER TABLE `tokens` ADD COLUMN `fallback_route_ids` TEXT DEFAULT '';

CREATE INDEX IF NOT EXISTS `idx_tokens_primary_route_id` ON `tokens` (`primary_route_id`);
```

---

## 2. Backward Compatibility & Zero-Downtime Guarantee

### 2.1 Default Value Invariant
- Existing API keys automatically have `primary_route_id = 0` and `fallback_route_ids = ""`.
- The routing engine checks `token.IsRouteEnabled()` (`primary_route_id > 0`).
- If `false`, the request bypasses the route fallback engine entirely and continues using legacy channel selection and auto-groups.
- **Zero breakage** for existing enterprise or mobile tokens.

### 2.2 Invalidation & Caching Strategy
- Routes are cached in memory using a fast-lookup sync map.
- When an administrator modifies or deletes a route via the Admin API:
  1. The local in-memory cache is cleaned.
  2. A Redis broadcast message is published on `tora:cache:route:invalidate`.
  3. All distributed New-API instances invalidate their local cache in near real-time.
- The hot relay execution path requires **zero database I/O** for route resolution.

---

## 3. Rollback & Disaster Recovery Plan

If a rollback of the route engine is ever required:
1. Reverting the application binary will cause `primary_route_id` and `fallback_route_ids` to be ignored.
2. The columns and `routes` table may remain in the database schema without interfering with previous application versions.
3. No data loss or table truncation is required.

Migration Sign-Off: **READY FOR DEPLOYMENT (Agent A)**
