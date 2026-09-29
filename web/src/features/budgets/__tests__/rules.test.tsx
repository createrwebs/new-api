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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import type { BudgetRule } from '../api'
import { Budgets } from '../index'

const daily: BudgetRule = {
  id: 1,
  scope_type: 'user',
  scope_id: 7,
  period: 'daily',
  limit_quota: 100,
  used_quota: 150,
  remaining_quota: 0,
  period_start: '2026-09-29T00:00:00Z',
  enabled: true,
}
const monthly: BudgetRule = {
  ...daily,
  id: 2,
  scope_type: 'token',
  scope_id: 9,
  period: 'monthly',
  used_quota: 0,
  remaining_quota: 100,
  enabled: false,
}

afterEach(() => vi.restoreAllMocks())

function mount(rules: BudgetRule[] = [daily, monthly]) {
  const get = vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: rules },
  })
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <Budgets />
    </QueryClientProvider>
  )
  return { get }
}

describe('admin budget rules', () => {
  it('shows daily and monthly usage including over-limit and disabled rules without token keys', async () => {
    mount([daily, { ...monthly, key: 'secret-budget-token' } as BudgetRule])
    expect(await screen.findByText('User #7')).toBeInTheDocument()
    expect(screen.getByText('Token #9')).toBeInTheDocument()
    expect(screen.getByText('Daily')).toBeInTheDocument()
    expect(screen.getByText('Monthly')).toBeInTheDocument()
    expect(screen.getByText('150%')).toBeInTheDocument()
    expect(screen.getByText('Disabled')).toBeInTheDocument()
    expect(screen.getAllByRole('progressbar')[0]).toHaveAttribute(
      'aria-valuenow',
      '100'
    )
    expect(screen.queryByText(/secret-budget-token/i)).not.toBeInTheDocument()
  })

  it('keeps status, usage and actions visible in the mobile card layout', async () => {
    const desktopMatchMedia = window.matchMedia
    vi.stubGlobal('matchMedia', (query: string) => ({
      ...desktopMatchMedia(query),
      matches: query.includes('max-width: 640px'),
    }))
    try {
      mount([monthly])
      expect(await screen.findByText('Token #9')).toBeInTheDocument()
      expect(screen.getByText('Disabled')).toBeInTheDocument()
      expect(screen.getByText('Monthly')).toBeInTheDocument()
      expect(screen.getByRole('button', { name: 'Edit' })).toBeInTheDocument()
    } finally {
      vi.unstubAllGlobals()
    }
  })

  it('creates a rule with exact quota points and refreshes the list', async () => {
    const { get } = mount([])
    const post = vi
      .spyOn(api, 'post')
      .mockResolvedValue({ data: { success: true } })
    const user = userEvent.setup()
    expect(await screen.findByText('No budget rules yet')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Create budget rule' }))
    const dialog = await screen.findByRole('dialog')
    await user.type(
      within(dialog).getByRole('spinbutton', { name: 'Scope ID' }),
      '7'
    )
    await user.type(
      screen.getByRole('spinbutton', { name: 'Limit (quota points)' }),
      '120'
    )
    await user.click(within(dialog).getByRole('button', { name: 'Create' }))
    await waitFor(() =>
      expect(post).toHaveBeenCalledWith('/api/budget/rules', {
        scope_type: 'user',
        scope_id: 7,
        period: 'daily',
        limit_quota: 120,
        enabled: true,
      })
    )
    await waitFor(() => expect(get).toHaveBeenCalledTimes(2))
  })

  it('creates a monthly token rule without sending a token key', async () => {
    mount([])
    const post = vi
      .spyOn(api, 'post')
      .mockResolvedValue({ data: { success: true } })
    const user = userEvent.setup()
    await screen.findByText('No budget rules yet')
    await user.click(screen.getByRole('button', { name: 'Create budget rule' }))
    const dialog = await screen.findByRole('dialog')
    await user.selectOptions(
      within(dialog).getByRole('combobox', { name: 'Scope type' }),
      'token'
    )
    await user.selectOptions(
      within(dialog).getByRole('combobox', { name: 'Period' }),
      'monthly'
    )
    await user.type(
      within(dialog).getByRole('spinbutton', { name: 'Scope ID' }),
      '9'
    )
    await user.type(
      within(dialog).getByRole('spinbutton', { name: 'Limit (quota points)' }),
      '50'
    )
    await user.click(within(dialog).getByRole('button', { name: 'Create' }))
    await waitFor(() =>
      expect(post).toHaveBeenCalledWith('/api/budget/rules', {
        scope_type: 'token',
        scope_id: 9,
        period: 'monthly',
        limit_quota: 50,
        enabled: true,
      })
    )
  })

  it('updates limit and enabled state without changing immutable fields', async () => {
    mount([daily])
    const put = vi
      .spyOn(api, 'put')
      .mockResolvedValue({ data: { success: true } })
    const user = userEvent.setup()
    await screen.findByText('User #7')
    await user.click(screen.getByRole('button', { name: 'Edit' }))
    await screen.findByRole('dialog')
    expect(screen.getByRole('spinbutton', { name: 'Scope ID' })).toBeDisabled()
    expect(screen.getByRole('combobox', { name: 'Period' })).toBeDisabled()
    await user.clear(
      screen.getByRole('spinbutton', { name: 'Limit (quota points)' })
    )
    await user.type(
      screen.getByRole('spinbutton', { name: 'Limit (quota points)' }),
      '200'
    )
    await user.click(screen.getByRole('switch', { name: 'Enabled' }))
    await user.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() =>
      expect(put).toHaveBeenCalledWith('/api/budget/rules/1', {
        scope_type: 'user',
        scope_id: 7,
        period: 'daily',
        limit_quota: 200,
        enabled: false,
      })
    )
  })

  it('confirms disable instead of deleting history', async () => {
    mount([daily])
    const del = vi
      .spyOn(api, 'delete')
      .mockResolvedValue({ data: { success: true } })
    const user = userEvent.setup()
    await screen.findByText('User #7')
    await user.click(screen.getByRole('button', { name: 'Disable' }))
    expect(
      await screen.findByText(/preserves its usage history/)
    ).toBeInTheDocument()
    expect(del).not.toHaveBeenCalled()
    await user.click(
      within(screen.getByRole('alertdialog')).getByRole('button', {
        name: 'Disable',
      })
    )
    await waitFor(() => expect(del).toHaveBeenCalledWith('/api/budget/rules/1'))
  })

  it('shows backend validation errors and leaves the form open', async () => {
    mount([])
    vi.spyOn(api, 'post').mockResolvedValue({
      data: { success: false, message: 'budget rule already exists' },
    })
    const user = userEvent.setup()
    await screen.findByText('No budget rules yet')
    await user.click(screen.getByRole('button', { name: 'Create budget rule' }))
    const dialog = await screen.findByRole('dialog')
    await user.type(screen.getByRole('spinbutton', { name: 'Scope ID' }), '7')
    await user.type(
      screen.getByRole('spinbutton', { name: 'Limit (quota points)' }),
      '100'
    )
    await user.click(within(dialog).getByRole('button', { name: 'Create' }))
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'budget rule already exists'
    )
    expect(
      screen.getByRole('spinbutton', { name: 'Limit (quota points)' })
    ).toBeInTheDocument()
  })

  it.each(['user not found', 'token not found', 'permission denied'])(
    'shows the backend error %s without closing the create dialog',
    async (message) => {
      mount([])
      vi.spyOn(api, 'post').mockResolvedValue({
        data: { success: false, message },
      })
      const user = userEvent.setup()
      await screen.findByText('No budget rules yet')
      await user.click(
        screen.getByRole('button', { name: 'Create budget rule' })
      )
      const dialog = await screen.findByRole('dialog')
      await user.type(
        within(dialog).getByRole('spinbutton', { name: 'Scope ID' }),
        '7'
      )
      await user.type(
        within(dialog).getByRole('spinbutton', {
          name: 'Limit (quota points)',
        }),
        '100'
      )
      await user.click(within(dialog).getByRole('button', { name: 'Create' }))
      expect(await within(dialog).findByRole('alert')).toHaveTextContent(
        message
      )
      expect(dialog).toBeInTheDocument()
    }
  )

  it('rejects nonpositive and unsafe quota values before sending a request', async () => {
    mount([])
    const post = vi.spyOn(api, 'post')
    const user = userEvent.setup()
    await screen.findByText('No budget rules yet')
    await user.click(screen.getByRole('button', { name: 'Create budget rule' }))
    const dialog = await screen.findByRole('dialog')
    await user.type(
      within(dialog).getByRole('spinbutton', { name: 'Scope ID' }),
      '0'
    )
    await user.type(
      within(dialog).getByRole('spinbutton', { name: 'Limit (quota points)' }),
      '9007199254740992'
    )
    await user.click(within(dialog).getByRole('button', { name: 'Create' }))
    expect(
      await within(dialog).findAllByText('Enter a positive integer')
    ).toHaveLength(2)
    expect(post).not.toHaveBeenCalled()
  })

  it('allows a disabled rule to be re-enabled via edit', async () => {
    mount([monthly])
    const put = vi
      .spyOn(api, 'put')
      .mockResolvedValue({ data: { success: true } })
    const user = userEvent.setup()
    await screen.findByText('Token #9')
    expect(
      screen.queryByRole('button', { name: 'Disable' })
    ).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Edit' }))
    const dialog = await screen.findByRole('dialog')
    expect(
      within(dialog).getByRole('combobox', { name: 'Period' })
    ).toBeDisabled()
    await user.click(within(dialog).getByRole('switch', { name: 'Enabled' }))
    await user.click(within(dialog).getByRole('button', { name: 'Save' }))
    await waitFor(() =>
      expect(put).toHaveBeenCalledWith('/api/budget/rules/2', {
        scope_type: 'token',
        scope_id: 9,
        period: 'monthly',
        limit_quota: 100,
        enabled: true,
      })
    )
  })

  it('shows load errors and offers retry', async () => {
    vi.spyOn(api, 'get').mockResolvedValue({
      data: { success: false, message: 'Forbidden' },
    })
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    render(
      <QueryClientProvider client={client}>
        <Budgets />
      </QueryClientProvider>
    )
    expect(await screen.findByText('Forbidden')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument()
  })
})
