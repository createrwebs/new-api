import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
  RouterProvider,
} from '@tanstack/react-router'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it, vi } from 'vitest'

import { CreatorAssistant } from '../components/CreatorAssistant'

afterEach(() => vi.unstubAllGlobals())

it('opens the wallet from an insufficient credit plan using keyboard navigation', async () => {
  vi.stubGlobal('scrollTo', vi.fn())
  vi.stubGlobal(
    'fetch',
    vi
      .fn()
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            success: true,
            data: {
              plan: {
                id: 'plan-1',
                status: 'DRAFT',
                requires_consent: false,
                total_estimated_credits: 10,
              },
              steps: [
                {
                  step_index: 1,
                  tool_display_name: 'Product photo',
                  status: 'PENDING',
                  depends_on: [],
                  estimated_credits: 10,
                  parameters: {},
                },
              ],
            },
          })
        )
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            error_code: 'INSUFFICIENT_CREDITS',
            data: {
              required_credits: 10,
              available_credits: 0,
              missing_credits: 10,
            },
          }),
          { status: 402 }
        )
      )
  )
  const root = createRootRoute({ component: Outlet })
  const assistant = createRoute({
    getParentRoute: () => root,
    path: '/',
    component: CreatorAssistant,
  })
  const wallet = createRoute({
    getParentRoute: () => root,
    path: '/wallet',
    component: () => <h1>Wallet</h1>,
  })
  const router = createRouter({
    routeTree: root.addChildren([assistant, wallet]),
    history: createMemoryHistory(),
    scrollRestoration: false,
  })
  await router.load()
  render(<RouterProvider router={router} />)
  const user = userEvent.setup()

  await user.type(await screen.findByRole('textbox'), 'Create a product photo')
  await user.click(screen.getByRole('button', { name: /Generate ToolPlan/ }))
  await user.click(
    await screen.findByRole('button', { name: /Confirm & Execute/ })
  )
  const walletLink = await screen.findByRole('button', { name: /Buy Credits/ })
  expect(walletLink).toHaveAttribute('href', '/wallet')
  walletLink.focus()
  await user.keyboard('{Enter}')

  expect(await screen.findByRole('heading', { name: 'Wallet' })).toBeVisible()
})
