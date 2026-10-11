# TORA AI — GA4 GROWTH ATTRIBUTION RUNBOOK
# SEARCH → CONTENT → PRODUCT CONVERSION CLOSED LOOP

## 1. Executive Summary

This runbook guides operators through provisioning and connecting Google Analytics 4 (GA4) to Tora AI (`www.toraapi.com`) to close the external growth attribution loop:
$$\text{Search / Social} \longrightarrow \text{News SSR} \longrightarrow \text{CTA} \longrightarrow \text{Tora Product SPA} \longrightarrow \text{Signup / Activation / Subscription}$$

The system implementation has been completely deployed and verified. Because production does not yet have an operator-provisioned GA4 Property and credentials, GA4 is safely marked:
```json
"ga4_status": "OPERATOR_BLOCKED",
"ga4_data_available": false
```
Once the operator completes the steps in this runbook and sets the environment variables, GA4 transitions automatically to `CONNECTED` and `DATA_AVAILABLE`.

---

## 2. Architecture & Double-Counting Prevention

Tora AI operates a hybrid frontend architecture:
1. **Server-Side Rendered (SSR) News**: `/news` and `/news/:slug` served directly by Go Gin for immediate search indexing and fast TTFB.
2. **Single-Page Application (SPA)**: Dashboard, Playground, Settings, Subscriptions served by React & TanStack Router.

### Prevention of Double Pageviews
- GA4 configuration script initializes with:
  ```javascript
  gtag('config', 'G-XXXXXXXXXX', { send_page_view: false });
  ```
- **SSR News**: Dispatches an explicit `page_view` event within the initial HTML `<head>` with privacy-safe parameters (`page_type: "news_article"`, `news_post_id: 123`, `content_type: "guide"`).
- **SPA Client**: TanStack Router listener explicitly calls `trackPageView(location.pathname + location.search)` only on route transitions. A deduplication guard (`lastTrackedPath`) prevents duplicate events.
- **Environment Isolation**: `isTrackingAllowed()` ensures telemetry only fires on `www.toraapi.com` and `toraapi.com`. Traffic from `localhost`, `127.0.0.1`, and staging domains is dropped.

---

## 3. Privacy & Safety Invariants

Telemetry is governed by strict sanitization (`sanitizeEventParams`):
1. **Prohibited Fields Dropped Automatically**:
   - AI prompt text & AI completion text
   - Chat message content & conversation histories
   - Email addresses, usernames, real names
   - Passwords, access tokens, API keys, BYOK keys
   - Raw user IDs & database primary keys
   - Payment card numbers, secrets, PromptPay tokens
2. **Regex Filter Engine**:
   - Drops keys matching `/token|secret|key|auth|pass|bearer|email|prompt|completion|message/i`.
   - Drops values matching email patterns (`/^[^@\s]+@[^@\s]+\.[^@\s]+$/`) or token hashes (`/^[a-zA-Z0-9_-]{32,}$/`).

---

## 4. Telemetry Event Registry

| Event Name | Source | Description & Attribution Purpose |
| :--- | :--- | :--- |
| `page_view` | SSR News / SPA Router | Unique pageview with deduplication guard. |
| `news_cta_click` | SSR News Client (`controller/news.go`) | Click on primary conversion CTAs (e.g. Try Tora, Register). |
| `news_internal_link_click` | SSR News Client | Internal links between articles with UTM preservation. |
| `sign_up` | SPA `sign-up-form.tsx` | User successfully registered an account. |
| `login` | SPA `user-auth-form.tsx` | User authenticated. |
| `pricing_view` | SPA `pricing/index.tsx` | User inspected tier pricing. |
| `chat_started` | SPA `use-chat-handler.ts` | User initiated an AI chat request. |
| `first_successful_chat` | SPA `use-chat-handler.ts` | First AI completion delivered (activation milestone). |
| `begin_checkout` | SPA `use-payment.ts` / Dialog | User clicked top-up / checkout with a payment method. |
| `purchase` | SPA `subscription-purchase-dialog.tsx` | Subscription transaction successfully redeemed. |

---

## 5. Operator Setup Step-by-Step

### Step 1: Create GA4 Property
1. Navigate to [Google Analytics Admin](https://analytics.google.com/analytics/web/#/admin).
2. Select your Google Analytics Account.
3. Click **Create Property**:
   - **Property name**: `Tora AI Production`
   - **Reporting time zone**: `Thailand (GMT+07:00)`
   - **Currency**: `Thai Baht (THB)`
4. Note the numeric **Property ID** (e.g. `123456789`).

### Step 2: Create Web Data Stream
1. In the newly created property, click **Data Streams** → **Add stream** → **Web**.
2. **Website URL**: `https://www.toraapi.com`
3. **Stream name**: `Tora AI Web Stream`
4. Click **Create stream**.
5. Copy the **Measurement ID** (format: `G-XXXXXXXXXX`).

### Step 3: Enable Google Analytics Data API in Google Cloud
1. Go to [Google Cloud Console](https://console.cloud.google.com/apis/library/analyticsdata.googleapis.com).
2. Select your project (the same project holding the Search Console service account or a dedicated project).
3. Search for **Google Analytics Data API** (v1beta).
4. Click **Enable**.

### Step 4: Grant Access to the Service Account
1. You may reuse the existing Search Console service account email (e.g. `tora-growth-bot@...iam.gserviceaccount.com`) or create a new dedicated service account.
2. In [Google Analytics Admin](https://analytics.google.com/analytics/web/#/admin), under **Property**, click **Property Access Management**.
3. Click the blue **+** button → **Add users**.
4. Enter the service account email.
5. Assign the role: **Viewer** (or **Analyst**).
6. Click **Add**.

### Step 5: Configure Production Environment Variables
On the production server (`/home/ubuntu/new-api/.env`):
```bash
# GA4 Web Tracking (Injected into SSR News and React SPA)
GA4_MEASUREMENT_ID=G-XXXXXXXXXX

# GA4 Data API (Collector & Admin Reporting)
GA4_PROPERTY_ID=properties/123456789

# Optional: dedicated GA4 service account file/json (defaults to GSC service account if omitted)
# GA4_CREDENTIALS_FILE=/path/to/credentials.json
# GA4_CREDENTIALS_JSON='{...}'
```

### Step 6: Restart Container & Verify
```bash
cd /home/ubuntu/new-api
docker compose down && docker compose up -d
```

Verify in the Growth Overview API:
```bash
curl -s -H "Authorization: Bearer <ADMIN_TOKEN>" https://www.toraapi.com/api/admin/news/growth/overview | jq .ga4_status
```
Expected output:
```json
"CONNECTED"
```
Or `"DATA_AVAILABLE"` once real traffic rows have accumulated in the property.

---

## 6. Verification Checklist

- [x] SSR News contains `<script async src="https://www.googletagmanager.com/gtag/js?id=..."></script>` with `send_page_view: false`.
- [x] Single-page transitions in React SPA track single `page_view` events without double counting.
- [x] CTA clicks pass through `trackNewsCTAClick` with UTM preserving `sessionStorage`.
- [x] `sign_up`, `chat_started`, `first_successful_chat`, `begin_checkout`, `purchase` events instrumented.
- [x] Localhost and staging environments drop telemetry cleanly.
- [x] GSC + GA4 + Local conversions joined in `GET /api/admin/news/posts/:id/growth`.
- [x] Opportunity quality classifies `TRAFFIC_OPPORTUNITY` vs `BUSINESS_VALUE_OPPORTUNITY`.
- [x] Status truthfully reports `OPERATOR_BLOCKED` until property credentials configured.
