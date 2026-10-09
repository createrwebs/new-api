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
import { type TFunction } from 'i18next'

import type { TokenUnit } from './types'

// ----------------------------------------------------------------------------
// Pricing Constants
// ----------------------------------------------------------------------------

/** Sort options for pricing models */
export const SORT_OPTIONS = {
  NAME: 'name',
  PRICE_LOW: 'price-low',
  PRICE_HIGH: 'price-high',
} as const

export type SortOption = (typeof SORT_OPTIONS)[keyof typeof SORT_OPTIONS]

export function getSortLabels(t: TFunction): Record<SortOption, string> {
  return {
    [SORT_OPTIONS.NAME]: t('Name'),
    [SORT_OPTIONS.PRICE_LOW]: t('Price: Low to High'),
    [SORT_OPTIONS.PRICE_HIGH]: t('Price: High to Low'),
  }
}

/** Filter values */
export const FILTER_ALL = 'all'

/** Quota type options */
export const QUOTA_TYPES = {
  ALL: 'all',
  TOKEN: 'token',
  REQUEST: 'request',
  TASK: 'task',
} as const

export type QuotaTypeOption = (typeof QUOTA_TYPES)[keyof typeof QUOTA_TYPES]

/** Quota type labels */
export function getQuotaTypeLabels(
  t: TFunction
): Record<QuotaTypeOption, string> {
  return {
    [QUOTA_TYPES.ALL]: t('All Models'),
    [QUOTA_TYPES.TOKEN]: t('Token-based'),
    [QUOTA_TYPES.REQUEST]: t('Per Request'),
    [QUOTA_TYPES.TASK]: t('Task billing'),
  }
}

/** Endpoint type options */
export const ENDPOINT_TYPES = {
  ALL: 'all',
  OPENAI: 'openai',
  OPENAI_RESPONSE: 'openai-response',
  ANTHROPIC: 'anthropic',
  GEMINI: 'gemini',
  JINA_RERANK: 'jina-rerank',
  IMAGE_GENERATION: 'image-generation',
  EMBEDDINGS: 'embeddings',
  OPENAI_VIDEO: 'openai-video',
} as const

export type EndpointTypeOption =
  (typeof ENDPOINT_TYPES)[keyof typeof ENDPOINT_TYPES]

/** Endpoint type labels */
export function getEndpointTypeLabels(
  t: TFunction
): Record<EndpointTypeOption, string> {
  return {
    [ENDPOINT_TYPES.ALL]: t('All Types'),
    [ENDPOINT_TYPES.OPENAI]: 'Chat',
    [ENDPOINT_TYPES.OPENAI_RESPONSE]: 'Response',
    [ENDPOINT_TYPES.ANTHROPIC]: 'Anthropic',
    [ENDPOINT_TYPES.GEMINI]: 'Gemini',
    [ENDPOINT_TYPES.JINA_RERANK]: 'Rerank',
    [ENDPOINT_TYPES.IMAGE_GENERATION]: t('Image'),
    [ENDPOINT_TYPES.EMBEDDINGS]: t('Embeddings'),
    [ENDPOINT_TYPES.OPENAI_VIDEO]: t('Video'),
  }
}

/** Filter section keys */
export const FILTER_SECTIONS = {
  PRICING_TYPE: 'pricingType',
  ENDPOINT_TYPE: 'endpointType',
  VENDOR: 'vendor',
  GROUP: 'group',
  TAG: 'tag',
} as const

/** Maximum number of tags to display in model row */
export const MAX_TAGS_DISPLAY = 5

/** Maximum number of filter items to display before showing "More..." */
export const MAX_FILTER_ITEMS = 5

/** Sidebar width */
export const SIDEBAR_WIDTH = 'w-64'

/** Excluded groups */
export const EXCLUDED_GROUPS = ['', 'auto']

/** Quota type values */
export const QUOTA_TYPE_VALUES = {
  TOKEN: 0,
  REQUEST: 1,
} as const

/** Token unit divisors */
export const TOKEN_UNIT_DIVISORS = {
  M: 1,
  K: 1000,
} as const

/** Default token unit for pricing display */
export const DEFAULT_TOKEN_UNIT: TokenUnit = 'M'

/** View mode options */
export const VIEW_MODES = {
  CARD: 'card',
  TABLE: 'table',
} as const

export type ViewMode = (typeof VIEW_MODES)[keyof typeof VIEW_MODES]

/** Default page size for pricing table */
export const DEFAULT_PRICING_PAGE_SIZE = 20

export interface GroupRouteMeta {
  tier: string
  kind: 'award' | 'stable' | 'premium' | 'limited'
  description: string
  color: string
}

export const ROUTE_METADATA: Record<string, GroupRouteMeta> = {
  'openai-award': {
    tier: 'High-value route',
    kind: 'award',
    description:
      'Price-priority routing for budget-sensitive, retryable development, testing, and flexible workloads.',
    color: 'bg-emerald-500',
  },
  'openai-stable': {
    tier: 'High-stability route',
    kind: 'stable',
    description:
      'The go-to choice for individual developers who prioritize stability, ideal for everyday development, long-term projects, and production applications.',
    color: 'bg-blue-500',
  },
  'openai-premium': {
    tier: 'High-stability route',
    kind: 'premium',
    description:
      'Premium OpenAI route for workloads that prioritize continuity and request success, billed at 13% of official list pricing.',
    color: 'bg-purple-500',
  },
  'openai-first-topup': {
    tier: 'Limited-time offer route',
    kind: 'limited',
    description:
      'Exclusive pricing unlocked when any single top-up reaches US$20.',
    color: 'bg-amber-500',
  },
  'claude-award': {
    tier: 'High-value route',
    kind: 'award',
    description:
      'Price-priority routing for budget-sensitive, retryable development, testing, and flexible workloads.',
    color: 'bg-emerald-500',
  },
  'claude-stable': {
    tier: 'High-stability route',
    kind: 'stable',
    description:
      'The go-to choice for individual developers who prioritize stability, ideal for everyday development, long-term projects, and production applications.',
    color: 'bg-blue-500',
  },
  'claude-premium': {
    tier: 'High-stability route',
    kind: 'premium',
    description:
      'Premium Claude route for workloads that prioritize continuity and request success, billed at 50% of official list pricing.',
    color: 'bg-purple-500',
  },
  'claude-first-topup': {
    tier: 'Limited-time offer route',
    kind: 'limited',
    description:
      'Exclusive pricing unlocked when any single top-up reaches US$20.',
    color: 'bg-amber-500',
  },
  'claude-tmp-cheap': {
    tier: 'Limited-time offer route',
    kind: 'limited',
    description: 'Limited-time promotional route for Claude models.',
    color: 'bg-amber-500',
  },
  'grok-award': {
    tier: 'Limited-time offer route',
    kind: 'award',
    description:
      'Price-priority routing for budget-sensitive, retryable development, testing, and flexible workloads.',
    color: 'bg-emerald-500',
  },
  'grok-stable': {
    tier: 'High-stability route',
    kind: 'stable',
    description:
      'The go-to choice for individual developers who prioritize stability, ideal for everyday development, long-term projects, and production applications.',
    color: 'bg-blue-500',
  },
  'deepseek-stable': {
    tier: 'High-stability route',
    kind: 'stable',
    description:
      'The go-to choice for individual developers who prioritize stability, ideal for everyday development, long-term projects, and production applications.',
    color: 'bg-blue-500',
  },
  'glm-stable': {
    tier: 'High-stability route',
    kind: 'stable',
    description:
      'Official GLM access on a high-stability channel, suited to everyday development, long-term projects, and production.',
    color: 'bg-blue-500',
  },
  'qwen-stable': {
    tier: 'High-stability route',
    kind: 'stable',
    description:
      'Official Qwen access on a high-stability channel, suited to everyday development, long-term projects, and production.',
    color: 'bg-blue-500',
  },
  'kimi-stable': {
    tier: 'High-stability route',
    kind: 'stable',
    description:
      'Official Kimi access on a high-stability channel, suited to everyday development, long-term projects, and production.',
    color: 'bg-blue-500',
  },
  'gemini-stable': {
    tier: 'High-stability route',
    kind: 'stable',
    description:
      'The go-to choice for individual developers who prioritize stability, ideal for everyday development, long-term projects, and production applications.',
    color: 'bg-blue-500',
  },
  'spark-premium': {
    tier: 'High-stability route',
    kind: 'premium',
    description: 'Muse Spark route billed at 1.15× the official token price.',
    color: 'bg-purple-500',
  },
  'jev-premium': {
    tier: 'High-stability route',
    kind: 'premium',
    description: 'Jev decisions route billed at 1.15× the official token price.',
    color: 'bg-purple-500',
  },
}

export function getRouteMeta(groupName: string): GroupRouteMeta {
  if (ROUTE_METADATA[groupName]) {
    return ROUTE_METADATA[groupName]
  }
  const isAward = groupName.includes('award') || groupName.includes('cheap')
  const isPremium = groupName.includes('premium')
  const isLimited = groupName.includes('topup') || groupName.includes('tmp')
  return {
    tier: isAward
      ? 'High-value route'
      : isPremium
        ? 'Premium route'
        : isLimited
          ? 'Limited-time offer route'
          : 'High-stability route',
    kind: isAward
      ? 'award'
      : isPremium
        ? 'premium'
        : isLimited
          ? 'limited'
          : 'stable',
    description: `Routing group for ${groupName}.`,
    color: isAward
      ? 'bg-emerald-500'
      : isPremium
        ? 'bg-purple-500'
        : isLimited
          ? 'bg-amber-500'
          : 'bg-blue-500',
  }
}

