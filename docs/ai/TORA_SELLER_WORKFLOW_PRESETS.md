# Tora Studio — Seller Workflow Presets & Repeat Last Pack
**File:** `docs/ai/TORA_SELLER_WORKFLOW_PRESETS.md`  
**Version:** `1.0`  
**Status:** IMPLEMENTED & VERIFIED  

---

## 1. Architectural Purpose

E-commerce merchants frequently run repetitive packaging tasks (e.g. taking weekly batches of 5-10 product cutouts and exporting them to the same set of Shopee, Lazada, and TikTok Shop templates).

Tora Studio provides two complementary productivity mechanisms:
1. **Saved Workflow Presets (`SellerWorkflowPreset`)**: Named, reusable pipeline configurations that store template selections, background styling, shadow choice, and branding.
2. **Repeat Last Pack (`SellerLastPackExecution`)**: Instant 1-click retrieval of the user's most recent batch parameters, enabling seamless continuity without manual configuration.

---

## 2. Hard Financial & Security Invariants

> [!IMPORTANT]
> **Configuration-Only Persistence Invariant**:
> Presets persist **ONLY visual and export configuration parameters**. They NEVER store prices, pre-calculated discounts, token escrows, or payment authorizations.
> All quotes and wallet settlements are evaluated strictly at execution time against canonical Tora Credit rules (1 Tora Credit = 1,000 Quota = $0.002 reference value).

---

## 3. Data Schema

### A. Saved Workflow Presets
```go
type SellerWorkflowPreset struct {
    Id                int    `json:"id" gorm:"primaryKey;autoIncrement"`
    UserId            int    `json:"user_id" gorm:"index;not null"`
    ProfileId         int    `json:"profile_id" gorm:"index;default:0"`
    PresetName        string `json:"preset_name" gorm:"size:255;not null"`
    Description       string `json:"description" gorm:"size:512"`
    SelectedTemplates string `json:"selected_templates" gorm:"type:text"` // JSON array string
    BgPreset          string `json:"bg_preset" gorm:"size:64;default:'PURE_WHITE'"`
    ShadowPreset      string `json:"shadow_preset" gorm:"size:64;default:'MARKETPLACE'"`
    BrandHex          string `json:"brand_hex" gorm:"size:32"`
    IncludeZip        bool   `json:"include_zip" gorm:"default:true"`
    CreatedAt         int64  `json:"created_at"`
    UpdatedAt         int64  `json:"updated_at"`
}
```

### B. Repeat Last Pack
```go
type SellerLastPackExecution struct {
    Id                int    `json:"id" gorm:"primaryKey;autoIncrement"`
    UserId            int    `json:"user_id" gorm:"uniqueIndex;not null"`
    SelectedTemplates string `json:"selected_templates" gorm:"type:text"`
    BgPreset          string `json:"bg_preset" gorm:"size:64"`
    ShadowPreset      string `json:"shadow_preset" gorm:"size:64"`
    BrandHex          string `json:"brand_hex" gorm:"size:32"`
    IncludeZip        bool   `json:"include_zip"`
    ItemCount         int    `json:"item_count"`
    LastExecutedAt    int64  `json:"last_executed_at"`
}
```

---

## 4. Endpoints

| Method | Path | Auth | Description |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/studio/seller/presets` | `UserAuth()` | List all saved presets for user |
| `POST` | `/api/studio/seller/presets` | `UserAuth()` | Create a new workflow preset |
| `PUT` | `/api/studio/seller/presets/:id` | `UserAuth()` | Update an existing preset |
| `DELETE` | `/api/studio/seller/presets/:id` | `UserAuth()` | Delete a preset |
| `GET` | `/api/studio/seller/repeat-last` | `UserAuth()` | Fetch most recent product pack execution config |
