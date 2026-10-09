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
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import { GroupedPricingCatalog } from '../components/grouped-pricing-catalog'
import { RoutePriceCard } from '../components/route-price-card'
import type { PricingModel, PricingVendor } from '../types'

const mockVendors: PricingVendor[] = [
  { id: 3, name: 'OpenAI', icon: 'OpenAI' },
  { id: 2, name: 'Anthropic', icon: 'Anthropic' },
]

const mockModels: PricingModel[] = [
  {
    id: 1,
    model_name: 'gpt-5.6-sol',
    vendor_name: 'OpenAI',
    description: 'Context window: 1.05M',
    tags: 'Chat Completions,Responses',
    quota_type: 0,
    model_ratio: 2.5,
    completion_ratio: 6.0,
    enable_groups: [
      'openai-award',
      'openai-stable',
      'openai-premium',
      'openai-first-topup',
    ],
  },
  {
    id: 2,
    model_name: 'gpt-5.6-terra',
    vendor_name: 'OpenAI',
    description: 'Context window: 1.05M',
    tags: 'Chat Completions,Responses',
    quota_type: 0,
    model_ratio: 1.0,
    completion_ratio: 6.0,
    enable_groups: ['openai-award', 'openai-stable'],
  },
  {
    id: 3,
    model_name: 'claude-sonnet-5-5',
    vendor_name: 'Anthropic',
    description: 'Context window: 1M',
    tags: 'Anthropic Messages,Chat Completions',
    quota_type: 0,
    model_ratio: 1.5,
    completion_ratio: 5.0,
    enable_groups: ['claude-stable', 'claude-premium'],
  },
]

const mockGroupRatio: Record<string, number> = {
  'openai-award': 0.03,
  'openai-stable': 0.08,
  'openai-premium': 0.13,
  'openai-first-topup': 0.015,
  'claude-stable': 0.315,
  'claude-premium': 0.5,
}

describe('RoutePriceCard', () => {
  it('calculates official prices and discounted route prices with 100% accuracy', () => {
    render(
      <RoutePriceCard
        model={mockModels[0]}
        group='openai-award'
        ratio={0.03}
        tokenUnit='M'
      />
    )

    // Route title and badges
    expect(screen.getByText('openai-award')).toBeInTheDocument()
    expect(screen.getByText('High-value route')).toBeInTheDocument()
    expect(screen.getByText('0.03×')).toBeInTheDocument()

    // Official and discounted prices for gpt-5.6-sol (ratio 2.5, completion 6.0)
    // Official Input: 2.5 * 2 = $5. Discounted: 5 * 0.03 = $0.15
    expect(screen.getByText('$5')).toBeInTheDocument()
    expect(screen.getByText('$0.15')).toBeInTheDocument()

    // Official Output: 5 * 6 = $30. Discounted: 30 * 0.03 = $0.9
    expect(screen.getByText('$30')).toBeInTheDocument()
    expect(screen.getByText('$0.9')).toBeInTheDocument()

    // Savings footer: ≈ 3% of official price
    expect(screen.getByText('≈ 3% of official price')).toBeInTheDocument()
  })

  it('renders limited-time offer route correctly with fractional percent', () => {
    render(
      <RoutePriceCard
        model={mockModels[0]}
        group='openai-first-topup'
        ratio={0.015}
        tokenUnit='M'
      />
    )

    expect(screen.getByText('openai-first-topup')).toBeInTheDocument()
    expect(screen.getByText('Limited-time offer route')).toBeInTheDocument()
    expect(screen.getByText('0.015×')).toBeInTheDocument()
    expect(screen.getByText('$0.075')).toBeInTheDocument()
    expect(screen.getByText('$0.45')).toBeInTheDocument()
    expect(screen.getByText('≈ 1.5% of official price')).toBeInTheDocument()
  })
})

describe('GroupedPricingCatalog', () => {
  it('organizes models into vendor sections and allows collapsing/expanding', async () => {
    const user = userEvent.setup()
    const onModelClick = vi.fn()

    render(
      <GroupedPricingCatalog
        models={mockModels}
        vendors={mockVendors}
        groupRatio={mockGroupRatio}
        onModelClick={onModelClick}
      />
    )

    // Vendor group headers should be present
    expect(screen.getByRole('heading', { level: 2, name: /OpenAI/i })).toBeInTheDocument()
    expect(screen.getByRole('heading', { level: 2, name: /Anthropic/i })).toBeInTheDocument()

    // Models inside OpenAI should be rendered
    expect(screen.getByText('gpt-5.6-sol')).toBeInTheDocument()
    expect(screen.getByText('gpt-5.6-terra')).toBeInTheDocument()

    // Models inside Anthropic should be rendered
    expect(screen.getByText('claude-sonnet-5-5')).toBeInTheDocument()

    // Clicking the OpenAI header collapses its models
    const openAiHeader = document.querySelector('[data-vendor-header="OpenAI"]') as HTMLElement
    expect(openAiHeader).not.toBeNull()
    await user.click(openAiHeader)

    // Models under OpenAI should no longer be rendered
    expect(screen.queryByText('gpt-5.6-sol')).not.toBeInTheDocument()

    // Anthropic models should still be visible
    expect(screen.getByText('claude-sonnet-5-5')).toBeInTheDocument()

    // Clicking again expands it back
    await user.click(openAiHeader)
    expect(screen.getByText('gpt-5.6-sol')).toBeInTheDocument()
  })

  it('filters models when a vendor pill is clicked', async () => {
    const user = userEvent.setup()

    render(
      <GroupedPricingCatalog
        models={mockModels}
        vendors={mockVendors}
        groupRatio={mockGroupRatio}
        onModelClick={vi.fn()}
      />
    )

    // Click "Anthropic" pill
    const anthropicPill = document.querySelector('[data-vendor-filter="Anthropic"]') as HTMLElement
    expect(anthropicPill).not.toBeNull()
    await user.click(anthropicPill)

    // Only Anthropic models should be visible
    expect(screen.getByText('claude-sonnet-5-5')).toBeInTheDocument()
    expect(screen.queryByText('gpt-5.6-sol')).not.toBeInTheDocument()

    // Click "All" pill
    const allPill = document.querySelector('[data-vendor-filter="all"]') as HTMLElement
    expect(allPill).not.toBeNull()
    await user.click(allPill)

    // Both should be visible
    expect(screen.getByText('gpt-5.6-sol')).toBeInTheDocument()
    expect(screen.getByText('claude-sonnet-5-5')).toBeInTheDocument()
  })

  it('supports global expand all and collapse all', async () => {
    const user = userEvent.setup()

    render(
      <GroupedPricingCatalog
        models={mockModels}
        vendors={mockVendors}
        groupRatio={mockGroupRatio}
        onModelClick={vi.fn()}
      />
    )

    const toggleAllBtn = screen.getByRole('button', { name: /Collapse all|Expand all/i })
    await user.click(toggleAllBtn)

    // If collapsed, models are hidden
    expect(screen.queryByText('gpt-5.6-sol')).not.toBeInTheDocument()

    // Click again to expand all
    await user.click(screen.getByRole('button', { name: /Expand all/i }))
    expect(screen.getByText('gpt-5.6-sol')).toBeInTheDocument()
  })
})
