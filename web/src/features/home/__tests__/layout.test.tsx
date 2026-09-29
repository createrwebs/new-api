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
import { act, render, screen } from '@testing-library/react'
import i18next from 'i18next'
import { afterEach, describe, expect, it, vi } from 'vitest'

import th from '@/i18n/locales/th.json'

import { Hero } from '../components/sections/hero'

vi.mock('@/hooks/use-status', () => ({
  useStatus: () => ({ status: { docs_link: 'https://docs.example.com' } }),
}))

vi.mock('@tanstack/react-router', () => ({
  Link: ({ to, ...props }: React.ComponentProps<'a'> & { to: string }) => (
    <a href={to} {...props} />
  ),
}))

describe('home hero layout', () => {
  afterEach(async () => {
    await i18next.changeLanguage('en')
  })

  it('stacks guest actions at narrow widths and restores a row on larger screens', () => {
    render(<Hero isAuthenticated={false} />)

    const actions = screen.getByRole('button', {
      name: /Get Started/,
    }).parentElement
    expect(actions).toHaveClass('flex-col', 'sm:flex-row')
    expect(screen.getByRole('button', { name: /Get Started/ })).toHaveClass(
      'max-sm:w-full'
    )
    expect(screen.getByRole('button', { name: /View Pricing/ })).toHaveClass(
      'max-sm:w-full'
    )
    expect(screen.getByRole('button', { name: /Docs/ })).toHaveClass(
      'max-sm:w-full'
    )
    expect(screen.getAllByRole('heading', { level: 3 })).toHaveLength(4)
  })

  it('fills both rows of the desktop category grid when four categories are shown', () => {
    render(<Hero isAuthenticated={false} />)

    const categoryHeadings = screen.getAllByRole('heading', { level: 3 })
    expect(categoryHeadings).toHaveLength(4)
    for (const heading of categoryHeadings) {
      expect(heading.parentElement?.parentElement).toHaveClass('lg:col-span-2')
    }
    expect(categoryHeadings[0].parentElement?.parentElement).toHaveClass(
      'lg:row-span-2'
    )
    expect(categoryHeadings[1].parentElement?.parentElement).toHaveClass(
      'lg:row-span-2'
    )
  })

  it('keeps dashboard and docs actions full width on narrow screens when signed in', () => {
    render(<Hero isAuthenticated />)

    expect(screen.getByRole('button', { name: /Go to Dashboard/ })).toHaveClass(
      'max-sm:w-full'
    )
    expect(screen.getByRole('button', { name: /Docs/ })).toHaveClass(
      'max-sm:w-full'
    )
  })

  it('shows Thai headline and category labels when Thai is selected', async () => {
    i18next.addResourceBundle('th', 'translation', th.translation, true, true)
    await act(() => i18next.changeLanguage('th'))

    render(<Hero isAuthenticated={false} />)

    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent(
      'API เดียวสำหรับทุก ไอเดีย AI'
    )
    expect(screen.getByRole('heading', { name: 'สร้างภาพ' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /เริ่มต้นใช้งาน/ })).toHaveClass(
      'max-sm:w-full'
    )
  })
})
