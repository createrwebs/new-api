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
import { useTranslation } from 'react-i18next'

import { PublicLayout } from '@/components/layout'
import { Markdown } from '@/components/ui/markdown'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'

import { DocsNav, DocsPager } from './components/docs-nav'
import { DEFAULT_DOC_SLUG, getDocEntry } from './constants'
import { useDocContent } from './hooks/use-doc-content'

/**
 * `/docs` — renders the markdown documents kept in the repository's
 * `docs/pages` directory.
 *
 * The article body is fetched at runtime from the static copy produced by
 * `rsbuild.config.ts`; see `hooks/use-doc-content.ts`. Titles, descriptions,
 * and ordering come from `constants.ts` so the header and sidebar render before
 * the body arrives.
 */
export function Docs(props: { slug?: string }) {
  const { t } = useTranslation()
  const slug = props.slug ?? DEFAULT_DOC_SLUG
  const entry = getDocEntry(slug)
  const { markdown, isLoading, error } = useDocContent(entry)

  return (
    <PublicLayout>
      <div className='mx-auto w-full max-w-6xl'>
        <div className='gap-8 lg:grid lg:grid-cols-[15rem_minmax(0,1fr)]'>
          <aside className='hidden lg:block'>
            <div className='sticky top-24'>
              <DocsNav activeSlug={slug} />
            </div>
          </aside>

          <main className='min-w-0'>
            {entry ? (
              <>
                <header className='mb-6'>
                  <h1 className='text-2xl font-semibold tracking-tight sm:text-3xl'>
                    {entry.title}
                  </h1>
                  <p className='text-muted-foreground mt-2 text-sm'>
                    {entry.description}
                  </p>
                </header>

                {error != null ? (
                  <div className='bg-card rounded-xl border p-6'>
                    <p className='text-sm font-medium'>
                      {t('This document could not be loaded.')}
                    </p>
                    <p className='text-muted-foreground mt-1 text-xs'>
                      {error}
                    </p>
                  </div>
                ) : null}

                {error == null && (isLoading || markdown === undefined) && (
                  <div className='space-y-3'>
                    <Skeleton className='h-5 w-2/3' />
                    <Skeleton className='h-4 w-full' />
                    <Skeleton className='h-4 w-5/6' />
                    <Skeleton className='mt-6 h-40 w-full' />
                  </div>
                )}

                {error == null &&
                  !isLoading &&
                  markdown !== undefined && (
                    <>
                      <Markdown className='docs-prose'>{markdown}</Markdown>
                      <DocsPager activeSlug={slug} />
                    </>
                  )}
              </>
            ) : (
              <div className='bg-card rounded-xl border p-6'>
                <p className='text-sm font-medium'>
                  {t('This document does not exist.')}
                </p>
                <Button
                  variant='outline'
                  size='sm'
                  className='mt-3'
                  render={<a href='/docs' />}
                >
                  {t('Back to documentation')}
                </Button>
              </div>
            )}
          </main>
        </div>
      </div>
    </PublicLayout>
  )
}