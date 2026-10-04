/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import axios from 'axios'

import { api } from '@/lib/api'
import { requireServerSuccess } from '@/lib/server-error-message'

import { ENDPOINT_TYPES } from '@/features/pricing/constants'
import type { PricingModel } from '@/features/pricing/types'

import type {
  GenerateImageInput,
  WorkbenchGenerateResult,
  WorkbenchMediaKind,
  WorkbenchModel,
} from './types'

/**
 * Relay client for `/v1/*`.
 *
 * The shared `api` instance attaches the console session token, which the
 * relay router does not accept — it authenticates with an API key instead.
 * This instance therefore stays free of the auth interceptor.
 */
const relayClient = axios.create({ baseURL: '' })

function toWorkbenchKind(
  endpointTypes: string[] | undefined
): WorkbenchMediaKind | null {
  if (endpointTypes?.includes(ENDPOINT_TYPES.IMAGE_GENERATION)) return 'image'
  if (endpointTypes?.includes(ENDPOINT_TYPES.OPENAI_VIDEO)) return 'video'
  return null
}

/**
 * Image and video models from the pricing catalog.
 *
 * `/api/pricing` is the only catalog that reports `supported_endpoint_types`,
 * so the workbench derives its model list from there instead of guessing from
 * model names.
 */
export async function getWorkbenchModels(): Promise<WorkbenchModel[]> {
  const res = await api.get('/api/pricing')
  const data = requireServerSuccess(res.data)
  const models: PricingModel[] = Array.isArray(data?.data) ? data.data : []

  const workbenchModels: WorkbenchModel[] = []
  for (const model of models) {
    const kind = toWorkbenchKind(model.supported_endpoint_types)
    if (kind && model.model_name) {
      workbenchModels.push({ id: model.model_name, kind })
    }
  }
  return workbenchModels
}

/**
 * Run an image generation through `/v1/images/generations`.
 *
 * The response may carry either remote URLs or base64 payloads; base64 payloads
 * are converted to data URIs so the result gallery can render them directly.
 */
export async function generateImage(
  input: GenerateImageInput
): Promise<WorkbenchGenerateResult> {
  const res = await relayClient.post(
    '/v1/images/generations',
    {
      model: input.model,
      prompt: input.prompt,
      n: input.count,
      size: input.size,
    },
    {
      headers: { Authorization: `Bearer ${input.apiKey}` },
      skipErrorHandler: true,
    } as Record<string, unknown>
  )

  const body = res.data as {
    created?: number
    data?: {
      url?: string
      b64_json?: string
      revised_prompt?: string
    }[]
  }

  const items = (body.data ?? [])
    .map((entry, index) => {
      let url = ''
      if (entry.url) {
        url = entry.url
      } else if (entry.b64_json) {
        url = `data:image/png;base64,${entry.b64_json}`
      }
      if (!url) return null
      return {
        id: `${input.model}-${body.created ?? 0}-${index}`,
        url,
        revisedPrompt: entry.revised_prompt,
      }
    })
    .filter((item): item is NonNullable<typeof item> => item !== null)

  return { created: body.created ?? 0, items }
}