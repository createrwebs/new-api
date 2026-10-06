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

export interface SavedPendingJob {
  tool_id: string
  template_id?: string
  input_params: Record<string, unknown>
  timestamp: number
}
