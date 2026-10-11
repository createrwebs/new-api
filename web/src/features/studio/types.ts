export type StudioToolCategory = 'all' | 'image' | 'video' | 'utility' | 'commercial'

export type StudioJobStatus =
  | 'CREATED'
  | 'RESERVED'
  | 'SUBMITTING'
  | 'PROCESSING'
  | 'SUCCEEDED'
  | 'FAILED'
  | 'CANCELLED'
  | 'AMBIGUOUS_SUBMISSION'

export interface StudioTool {
  id: string
  name: string
  name_th: string
  slug: string
  category: string
  description: string
  description_th: string
  provider: string
  primary_provider: string
  endpoint: string
  input_schema: string
  output_schema: string
  credit_cost: number
  quota_cost: number
  estimated_sec: number
  status: 'active' | 'disabled' | 'operator_blocked'
  icon?: string
  badge?: string
  sort_order: number
}

export interface StudioTemplate {
  id: string
  tool_id: string
  title: string
  title_th: string
  description: string
  description_th: string
  category: string
  sample_inputs: string
  preview_image_url: string
  badge?: string
  sort_order: number
}

export interface StudioJob {
  id: string
  user_id: number
  tool_id: string
  template_id?: string
  idempotency_key: string
  provider: string
  provider_job_id?: string
  status: StudioJobStatus
  input_params: string
  output_data?: string
  error_code?: string
  error_message?: string
  quota_reserved: number
  quota_charged: number
  credit_charged: number
  duration_ms?: number
  created_at: string
  updated_at: string
  completed_at?: string
}

export interface CreateStudioJobPayload {
  tool_id: string
  template_id?: string
  idempotency_key?: string
  provider?: string
  input_params: Record<string, unknown>
}

export interface InsufficientCreditData {
  required_credits: number
  current_credits: number
  missing_credits: number
  tool_id?: string
  template_id?: string
}

export interface StudioTelemetrySummary {
  date: string
  total_jobs_today: number
  succeeded_jobs_today: number
  failed_jobs_today: number
  success_rate_percent: number
  gross_revenue_credits_today: number
  gross_cost_usd_today: number
  gross_revenue_usd_est: number
  gross_profit_usd_est: number
  gross_margin_percent: number
  active_tools_count: number
  blocked_tools_count: number
}

export type NativeExecutionClass = 'NATIVE_BROWSER' | 'NATIVE_SERVER' | 'NATIVE_SERVERLESS' | 'EXTERNAL_RELAY' | 'DETERMINISTIC_SERVER'
export type NativeBillingPolicy = 'PREPAID_EXECUTION' | 'SUCCESS_SETTLEMENT' | 'AMBIGUOUS_RECONCILIATION'
export type NativeTicketStatus = 'QUOTED' | 'RESERVED' | 'ACTIVATING' | 'CHARGED' | 'STARTED' | 'COMPLETED' | 'FAILED_CLIENT' | 'EXPIRED' | 'SUPPORT_REFUNDED'

export interface NativeExecutionTicket {
  id: string
  ticket_id: string
  user_id: number
  tool_id: string
  tool_version: string
  route_version: string
  execution_class: NativeExecutionClass
  billing_policy: NativeBillingPolicy
  model_id: string
  model_version: string
  model_version_hash: string
  quote_id: string
  request_id: string
  idempotency_key?: string
  reserved_quota: number
  reserved_credits: number
  charged_quota: number
  charged_credits: number
  status: NativeTicketStatus
  auth_token?: string
  nonce?: string
  issued_at: number
  expires_at: number
  charged_at: number
  retry_until: number
  retry_count: number
  completed_at?: number
  client_execution_ms?: number
  output_asset_hash?: string
  error_reason?: string
}

export interface NativeQuoteResult {
  quote_id: string
  tool_id: string
  execution_class: NativeExecutionClass
  billing_policy: NativeBillingPolicy
  credits: number
  quota: number
  usd_equivalent: number
  model: {
    model_id: string
    model_name: string
    filename: string
    sha256: string
    size_bytes: number
    format: string
    license: string
    execution_env: string
    download_url?: string
  }
  route_version: string
  expires_at: number
  fallback_provider?: string
}

export interface SavedPendingJob {
  tool_id: string
  template_id?: string
  input_params: Record<string, unknown>
  timestamp: number
}

// --- Seller Factory V2 & Object Cleanup Types ---

export type SmartShadowPreset =
  | 'SOFT_STUDIO'
  | 'MARKETPLACE'
  | 'FLOATING'
  | 'GROUND_CONTACT'
  | 'NO_SHADOW'

export type SellerBgPreset =
  | 'PURE_WHITE'
  | 'WARM_WHITE'
  | 'LIGHT_GRAY'
  | 'BRAND_COLOR'
  | 'SOFT_GRADIENT'
  | 'STUDIO_VIGNETTE'
  | 'TRANSPARENT'

export interface SellerTemplateConfig {
  template_id: string
  version: string
  marketplace: string
  filename: string
  width: number
  height: number
  aspect_ratio: string
  padding_pct: number
  default_bg: SellerBgPreset
  default_shadow: SmartShadowPreset
  format: 'jpg' | 'png'
  quality: number
  subject_anchor_y: number
  description: string
}

export interface SellerTemplatesCatalog {
  templates: SellerTemplateConfig[]
  shadow_presets: SmartShadowPreset[]
  bg_presets: SellerBgPreset[]
}

export interface ProductFactoryBatchQuoteRequest {
  input_count: number
  enable_2x_upscale?: boolean
  client_app_version?: string
}

export interface ProductFactoryBatchQuoteResult {
  quote_id: string
  price_scope: 'PER_ITEM' | 'BUNDLE'
  input_count: number
  enable_2x_upscale: boolean
  per_item_credits: number
  bundle_discount_credits: number
  local_steps_total: number
  server_steps_total: number
  total_credits: number
  total_quota: number
  usd_equivalent: number
  pricing_version: string
  minimum_app_version: string
  expires_at: number
}

export interface ProductFactoryV2ItemRequest {
  index: number
  cutout_png_base64?: string
  original_name?: string
}

export interface ProductFactoryV2BatchRequest {
  batch_id?: string
  items: ProductFactoryV2ItemRequest[]
  selected_templates: string[]
  bg_preset?: SellerBgPreset
  brand_hex?: string
  shadow_preset?: SmartShadowPreset
  include_zip?: boolean
}

export interface ProductFactoryV2ItemResult {
  index: number
  status: 'SUCCESS' | 'FAILED'
  error_reason?: string
  duration_ms?: number
  variants: Array<{
    key: string
    marketplace: string
    width: number
    height: number
    file_size: number
    sha256: string
    url?: string
  }>
}

export interface ProductFactoryV2BatchResult {
  batch_id: string
  total_items: number
  success_items: number
  failed_items: number
  items: ProductFactoryV2ItemResult[]
  zip_package?: {
    key: string
    marketplace: string
    file_size: number
    sha256: string
    url?: string
  }
  execution_time: number
}

export interface ObjectCleanupSessionResult {
  session_id: string
  ticket_id: string
  source_hash: string
  credits_deducted: number
  remaining_quota: number
  expires_at: string
  max_exports: number
  exports_used: number
}

