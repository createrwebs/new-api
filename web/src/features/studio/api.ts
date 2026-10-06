import axios from 'axios'

import { api } from '@/lib/api'
import { requireServerSuccess } from '@/lib/server-error-message'

import type {
  CreateStudioJobPayload,
  InsufficientCreditData,
  StudioJob,
  StudioTelemetrySummary,
  StudioTemplate,
  StudioTool,
} from './types'

export class InsufficientCreditException extends Error {
  data: InsufficientCreditData

  constructor(message: string, data: InsufficientCreditData) {
    super(message)
    this.name = 'InsufficientCreditException'
    this.data = data
  }
}

/**
 * Fetch all public Studio tools from the server.
 */
export async function getStudioTools(): Promise<StudioTool[]> {
  const res = await api.get('/api/studio/tools')
  const payload = requireServerSuccess(res.data)
  return Array.isArray(payload?.data) ? payload.data : []
}

/**
 * Fetch a single Studio tool by slug or ID.
 */
export async function getStudioToolBySlug(slug: string): Promise<StudioTool> {
  const res = await api.get(`/api/studio/tools/${slug}`)
  const payload = requireServerSuccess(res.data)
  return payload?.data
}

/**
 * Fetch curated Studio templates.
 */
export async function getStudioTemplates(
  toolId?: string,
  category?: string
): Promise<StudioTemplate[]> {
  const params: Record<string, string> = {}
  if (toolId) params.tool_id = toolId
  if (category && category !== 'all') params.category = category

  const res = await api.get('/api/studio/templates', { params })
  const payload = requireServerSuccess(res.data)
  return Array.isArray(payload?.data) ? payload.data : []
}

/**
 * Submit a Studio generation job.
 * Throws InsufficientCreditException on HTTP 402 with exact balance details.
 */
export async function createStudioJob(
  payload: CreateStudioJobPayload
): Promise<StudioJob> {
  try {
    const res = await api.post('/api/studio/jobs', payload, {
      skipErrorHandler: true,
    })
    const data = requireServerSuccess(res.data)
    return data?.data
  } catch (error: unknown) {
    if (axios.isAxiosError(error) && error.response) {
      if (error.response.status === 402) {
        const body = error.response.data as {
          message?: string
          data?: InsufficientCreditData
        }
        throw new InsufficientCreditException(
          body.message || 'เครดิต Tora Credits ไม่เพียงพอ',
          body.data || {
            required_credits: 0,
            current_credits: 0,
            missing_credits: 0,
          }
        )
      }
      const message = error.response.data?.message || error.message
      throw new Error(message)
    }
    throw error
  }
}

/**
 * Poll job status and result.
 */
export async function getStudioJob(jobId: string): Promise<StudioJob> {
  const res = await api.get(`/api/studio/jobs/${jobId}`, {
    skipErrorHandler: true,
  })
  const payload = requireServerSuccess(res.data)
  return payload?.data
}

/**
 * List the current user's generation history.
 */
export async function getUserStudioJobs(
  page = 1,
  pageSize = 20
): Promise<{ jobs: StudioJob[]; total: number }> {
  const res = await api.get('/api/studio/jobs', {
    params: { page, page_size: pageSize },
  })
  const payload = requireServerSuccess(res.data)
  return {
    jobs: Array.isArray(payload?.data?.jobs) ? payload.data.jobs : [],
    total: payload?.data?.total || 0,
  }
}

/**
 * Cancel an active generation job.
 */
export async function cancelStudioJob(jobId: string): Promise<StudioJob> {
  const res = await api.post(`/api/studio/jobs/${jobId}/cancel`)
  const payload = requireServerSuccess(res.data)
  return payload?.data
}

/**
 * Fetch admin telemetry for studio.
 */
export async function getStudioTelemetry(): Promise<StudioTelemetrySummary> {
  const res = await api.get('/api/admin/studio/telemetry')
  const payload = requireServerSuccess(res.data)
  return payload?.data
}

/**
 * Request dynamic quote for a studio job.
 */
export async function quoteStudioJob(payload: {
  tool_id: string
  template_id?: string
  input_params: Record<string, unknown>
}): Promise<{
  quote_id: string
  calculated_credits: number
  expires_at: number
  estimated_cost_usd: number
}> {
  const res = await api.post('/api/studio/quote', payload)
  const payloadData = requireServerSuccess(res.data)
  return payloadData?.data
}

/**
 * Record studio conversion milestone.
 */
export async function recordStudioAttribution(
  eventType: 'insufficient_credit' | 'buy_credit_click' | 'purchase_return' | 'generation_after_purchase',
  toolId: string,
  credits = 0,
  sessionId?: string
): Promise<void> {
  try {
    await api.post('/api/studio/attribution', {
      event_type: eventType,
      tool_id: toolId,
      credits,
      session_id: sessionId,
    })
  } catch {
    // Non-blocking telemetry
  }
}

