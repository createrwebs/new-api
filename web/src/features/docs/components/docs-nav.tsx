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
import { Link } from '@tanstack/react-router'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { DOC_ENTRIES, DOC_SECTIONS, getDocNeighbours } from '../constants'

/**
 * Sticky section index for the docs, grouped by `section`.
 *
 * Rendered next to the article rather than inside the app sidebar so it stays
 * usable on the public (unauthenticated) layout.
 */
export function DocsNav(props: { activeSlug: string }) {
  const { t } = useTranslation()

  return (
    <nav aria-label={t('Documentation')} className='text-sm'>
      {DOC_SECTIONS.map((section) => {
        const entries = DOC_ENTRIES.filter(
          (entry) => entry.section === section.id
        )
        if (entries.length === 0) return null

        return (
          <div key={section.id} className='mb-5'>
            <p className='text-muted-foreground mb-2 text-[11px] font-semibold tracking-wider uppercase'>
              {t(section.title)}
            </p>
            <ul className='space-y-0.5'>
              {entries.map((entry) => {
                const isActive = entry.slug === props.activeSlug
                return (
                  <li key={entry.slug}>
                    <Link
                      to="/docs/$slug"
                      params={{ slug: entry.slug }}
                      aria-current={isActive ? 'page' : undefined}
                      className={
                        isActive
                          ? 'bg-primary/10 text-primary block rounded-md px-2 py-1.5 font-medium'
                          : 'text-muted-foreground hover:text-foreground block rounded-md px-2 py-1.5 transition-colors hover:bg-muted'
                      }
                    >
                      {entry.title}
                    </Link>
                  </li>
                )
              })}
            </ul>
          </div>
        )
      })}
    </nav>
  )
}

/** Previous/next links driven by the registry order. */
export function DocsPager(props: { activeSlug: string }) {
  const { t } = useTranslation()
  const { previous, next } = getDocNeighbours(props.activeSlug)

  if (!previous && !next) return null

  return (
    <nav
      aria-label={t('Documentation pages')}
      className='mt-8 flex flex-col gap-3 border-t pt-6 sm:flex-row sm:justify-between'
    >
      {previous ? (
        <Link
          to='/docs/$slug'
          params={{ slug: previous.slug }}
          className='group hover:bg-muted/60 flex min-w-0 flex-1 items-center gap-2 rounded-lg border p-3 transition-colors'
        >
          <ChevronLeft
            className='text-muted-foreground size-4 shrink-0'
            aria-hidden='true'
          />
          <span className='min-w-0'>
            <span className='text-muted-foreground block text-xs'>
              {t('Previous')}
            </span>
            <span className='block truncate text-sm font-medium'>
              {previous.title}
            </span>
          </span>
        </Link>
      ) : (
        <span className='hidden flex-1 sm:block' />
      )}

      {next ? (
        <Link
          to='/docs/$slug'
          params={{ slug: next.slug }}
          className='group hover:bg-muted/60 flex min-w-0 flex-1 items-center justify-end gap-2 rounded-lg border p-3 text-right transition-colors'
        >
          <span className='min-w-0'>
            <span className='text-muted-foreground block text-xs'>
              {t('Next')}
            </span>
            <span className='block truncate text-sm font-medium'>
              {next.title}
            </span>
          </span>
          <ChevronRight
            className='text-muted-foreground size-4 shrink-0'
            aria-hidden='true'
          />
        </Link>
      ) : (
        <span className='hidden flex-1 sm:block' />
      )}
    </nav>
  )
}