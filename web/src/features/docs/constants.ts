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

/**
 * Public path the markdown files are served from.
 *
 * `rsbuild.config.ts` copies `docs/pages/*.md` into `docs-content/<slug>.md`
 * for both `rsbuild dev` and `rsbuild build`. Keep the two in sync when
 * adding a document.
 */
export const DOC_CONTENT_BASE_PATH = '/docs-content'

export type DocSectionId = 'getting-started' | 'guides' | 'reference'

export interface DocEntry {
  /** URL segment under `/docs/`. */
  slug: string
  /** File basename under `docs-content/` (no extension). */
  contentId: string
  title: string
  description: string
  section: DocSectionId
  order: number
}

/**
 * Ordered document registry.
 *
 * Titles and descriptions are duplicated from each markdown file's H1 and
 * opening paragraph so the sidebar and page header can render before the
 * document body is fetched.
 */
export const DOC_ENTRIES: readonly DocEntry[] = [
  {
    slug: 'introduction',
    contentId: 'introduction',
    title: 'Introduction',
    description:
      'One key for chat, images, and video, with setup guides for coding agents.',
    section: 'getting-started',
    order: 1,
  },
  {
    slug: 'quickstart',
    contentId: 'quickstart',
    title: 'Quickstart',
    description: 'Create an API key and send a first request.',
    section: 'getting-started',
    order: 2,
  },
  {
    slug: 'models-and-groups',
    contentId: 'models-and-groups',
    title: 'Models and groups',
    description: 'Choose a model ID and a routing group that can serve it.',
    section: 'getting-started',
    order: 3,
  },
  {
    slug: 'authentication',
    contentId: 'authentication',
    title: 'Authentication',
    description: 'Use an API key as a Bearer token on every request.',
    section: 'reference',
    order: 4,
  },
  {
    slug: 'endpoint-to-call',
    contentId: 'endpoint-to-call',
    title: 'Which endpoint to call',
    description: 'Base URL, host choice, and which API family to use.',
    section: 'reference',
    order: 5,
  },
  {
    slug: 'errors',
    contentId: 'errors',
    title: 'Errors',
    description: 'Read status codes and the JSON error object.',
    section: 'reference',
    order: 6,
  },
  {
    slug: 'usage-and-cost',
    contentId: 'usage-and-cost',
    title: 'Usage and cost',
    description: 'Where charges are recorded and how to read live prices.',
    section: 'reference',
    order: 7,
  },
  {
    slug: 'codex-setup',
    contentId: 'codex-setup',
    title: 'Codex setup',
    description: 'Connect Codex through the Responses endpoint.',
    section: 'guides',
    order: 8,
  },
  {
    slug: 'claude-code-setup',
    contentId: 'claude-code-setup',
    title: 'Claude Code setup',
    description: 'Connect Claude Code through Anthropic Messages.',
    section: 'guides',
    order: 9,
  },
  {
    slug: 'gemini-cli-setup',
    contentId: 'gemini-cli-setup',
    title: 'Gemini CLI setup',
    description: 'Connect the official Gemini CLI via the native Gemini API.',
    section: 'guides',
    order: 10,
  },
  {
    slug: 'grok-build-setup',
    contentId: 'grok-build-setup',
    title: 'Grok Build setup',
    description: 'Configure the Grok Build CLI with the Responses endpoint.',
    section: 'guides',
    order: 11,
  },
  {
    slug: 'opencode-setup',
    contentId: 'opencode-setup',
    title: 'OpenCode setup',
    description: 'Configure the gateway as an OpenAI-compatible provider.',
    section: 'guides',
    order: 12,
  },
]

export const DOC_SECTIONS: readonly {
  id: DocSectionId
  title: string
}[] = [
  { id: 'getting-started', title: 'Getting started' },
  { id: 'guides', title: 'Coding agents' },
  { id: 'reference', title: 'Reference' },
]

export const DEFAULT_DOC_SLUG = 'introduction'

export function getDocEntry(slug: string): DocEntry | undefined {
  return DOC_ENTRIES.find((entry) => entry.slug === slug)
}

/** Previous/next neighbours for the pager, or `null` at either end. */
export function getDocNeighbours(
  slug: string
): { previous: DocEntry | null; next: DocEntry | null } {
  const index = DOC_ENTRIES.findIndex((entry) => entry.slug === slug)
  if (index === -1) return { previous: null, next: null }
  return {
    previous: DOC_ENTRIES[index - 1] ?? null,
    next: DOC_ENTRIES[index + 1] ?? null,
  }
}