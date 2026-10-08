# Tora Studio — Seller Brand Profile Architecture
**File:** `docs/ai/TORA_SELLER_PROFILE_ARCHITECTURE.md`  
**Version:** `1.0`  
**Status:** IMPLEMENTED & VERIFIED  

---

## 1. Architectural Purpose

In e-commerce operations, sellers manage multiple product lines across Shopee, Lazada, TikTok Shop, and Instagram. Requiring merchants to repeatedly select brand HEX codes, shadow styles, and channel lists creates friction.

**Seller Brand Profiles** (`SellerBrandProfile`) provide a persistent store identity for merchants in Tora Studio:
- Store Name & Primary Sales Channels
- Brand Primary and Secondary Hex Colors
- Preferred E-Commerce Background (`PURE_WHITE`, `WARM_WHITE`, `TRANSPARENT`)
- Preferred Drop Shadow (`MARKETPLACE`, `SOFT_STUDIO`, `CONTACT_ONLY`)
- Default Safe Margin Padding Percentage
- Default Profile Flag (automatically selected in UI)

---

## 2. Data Schema & Isolation

Each profile is strictly scoped to the authenticated `user_id`:
```go
type SellerBrandProfile struct {
    Id               int     `json:"id" gorm:"primaryKey;autoIncrement"`
    UserId           int     `json:"user_id" gorm:"index;not null"`
    StoreName        string  `json:"store_name" gorm:"size:255;not null"`
    PrimaryChannels  string  `json:"primary_channels" gorm:"size:255"` // "shopee,lazada,tiktok_shop"
    BrandHex         string  `json:"brand_hex" gorm:"size:32"`
    SecondaryHex     string  `json:"secondary_hex" gorm:"size:32"`
    PreferredBg      string  `json:"preferred_bg" gorm:"size:64;default:'PURE_WHITE'"`
    PreferredShadow  string  `json:"preferred_shadow" gorm:"size:64;default:'MARKETPLACE'"`
    DefaultMarginPct float64 `json:"default_margin_pct" gorm:"default:0.10"`
    IsDefault        bool    `json:"is_default" gorm:"default:false"`
    CreatedAt        int64   `json:"created_at"`
    UpdatedAt        int64   `json:"updated_at"`
}
```

### Invariants:
1. **User Isolation**: A user cannot read, update, or delete profiles belonging to other users.
2. **Single Default Guarantee**: When a profile is marked `is_default: true`, any previous default profile belonging to that user is automatically updated to `is_default: false`.
3. **Zero Credentials**: Profiles store store visual branding only. No marketplace API keys or credentials are stored.

---

## 3. Endpoints

| Method | Path | Auth | Description |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/studio/seller/profiles` | `UserAuth()` | List all profiles belonging to user (default first) |
| `POST` | `/api/studio/seller/profiles` | `UserAuth()` | Create a new brand profile |
| `PUT` | `/api/studio/seller/profiles/:id` | `UserAuth()` | Update an existing brand profile |
| `DELETE` | `/api/studio/seller/profiles/:id` | `UserAuth()` | Delete a brand profile |
