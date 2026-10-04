# Usage and cost

Where charges are recorded and how to read live model prices.

ToraApi records token usage and cost per request. The live prices are on [Models & Pricing](https://ToraApi.com/pricing). This documentation site does not publish a second price list.

## What to check

**A charge you cannot place**

- **The model ID**: The exact ID and the quality or size parameters in the request, not a nearby variant.
- **The routing group**: Group multipliers apply to ordinary chat routing; some image contracts ignore them and bill the requested quality and dimensions instead. The Images guide states that exception explicitly.
- **The usage log**: Check it in the console after the request completes.

## Image and video billing

Image and video catalog pages name the billing unit: token, per-image, per-call, or per-second. Use the unit on the model page together with the live price, not a generic chat-token formula.

## Articles

Cost accounting, caching, and key isolation are covered by the main-site articles:

- [AI API cost tracking](https://ToraApi.com/articles/ai-api-cost-tracking)
- [API key security](https://ToraApi.com/articles/api-key-security-cost-controls)