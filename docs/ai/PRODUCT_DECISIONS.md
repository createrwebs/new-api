# Tora AI — Frozen Product Decisions

These decisions are frozen unless the operator explicitly changes them.

## Identity

- Product name: **Tora AI**
- iOS Bundle ID: `com.saascover.tora`
- Android application ID / namespace: `com.saascover.tora`
- production API: `https://api.tora.ai`
- staging API: `https://staging-api.tora.ai`

## Commercial Policy

- Free group: `default`
- Paid plan: **Pro Monthly**
- Paid group: `pro`
- Retail baseline: **USD $9.99/month**
- Quota: **2,000,000 quota units/month**
- Monthly quota reset
- No annual plan for initial launch
- `vip` is internal only and is not a native store product
- BYOK remains available independently of subscription

## Native Store Identity

Apple:

- Subscription group: `Tora AI Subscriptions`
- Product ID: `com.saascover.tora.pro.monthly`

Google:

- Subscription ID: `tora_pro`
- Base plan ID: `monthly`

## Payment Strategy

### Native mobile subscription

- iOS: Apple IAP / StoreKit
- Android distributed through Google Play: Google Play Billing

### Web / external

- Reuse existing New-API Stripe / TopUp stack
- Card: supported through Stripe where configured
- PromptPay: reuse through Stripe if the existing implementation/configuration supports it
- TrueMoney direct: defer unless separately approved or already present and production-used

## UX / Architecture Policy

- API-first for high-value mobile workflows
- hosted payment/provider UI should open in system/in-app browser where practical
- WebView is fallback, not the default
- no BFF by default
- do not duplicate mature New-API functionality without evidence that reuse is inadequate
