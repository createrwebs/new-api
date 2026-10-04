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
import { useQuery } from '@tanstack/react-query'
import { useMemo } from 'react'

import { DOC_CONTENT_BASE_PATH, type DocEntry } from '../constants'
import { renderDocMarkdown } from '../lib/markdown'

/**
 * Loads one markdown document and returns it already normalized for rendering.
 *
 * The files are copied to a static path by `rsbuild.config.ts`; fetching them
 * instead of importing keeps the prose out of the JS bundle, so a reader only
 * downloads the page they actually open.
 */
export function useDocContent(entry: DocEntry | undefined) {
  const query = useQuery({
    queryKey: ['docs', 'content', entry?.contentId],
    enabled: Boolean(entry),
    staleTime: 10 * 60 * 1000,
    retry: false,
    queryFn: async () => {
      if (!entry) throw new Error('No documentation entry requested')
      const response = await fetch(
        `${DOC_CONTENT_BASE_PATH}/${entry.contentId}.md`,
        { cache: 'force-cache' }
      )
      if (!response.ok) {
        throw new Error(
          `Failed to load "${entry.contentId}.md" (HTTP ${response.status})`
        )
      }
      return response.text()
    },
  })

  const markdown = useMemo(
    () =>
      query.data === undefined ? undefined : renderDocMarkdown(query.data),
    [query.data]
  )

  return {
    markdown,
    isLoading: query.isLoading,
    error: query.error instanceof Error ? query.error.message : null,
  }
}