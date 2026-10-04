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
import fs from 'node:fs'
import path from 'node:path'

import { describe, expect, it } from 'vitest'

import { DOC_ENTRIES } from '../../constants'
import { renderDocMarkdown } from '../markdown'

const DOCS_DIR = path.resolve(import.meta.dirname, '../../../../../../docs/pages')

// Mirrors the source filename -> slug map in rsbuild.config.ts.
const SOURCE_FILES: Record<string, string> = {
  'introduction.md': 'introduction',
  'quickstart.md': 'quickstart',
  'endpoint_to_call.md': 'endpoint-to-call',
  'models_and_groups.md': 'models-and-groups',
  'Authentication.md': 'authentication',
  'Errors.md': 'errors',
  'Usage and cost.md': 'usage-and-cost',
  'Codex setup.md': 'codex-setup',
  'Claude Code setup.md': 'claude-code-setup',
  'Gemini CLI setup.md': 'gemini-cli-setup',
  'Grok Build setup.md': 'grok-build-setup',
  'OpenCode setup.md': 'opencode-setup',
}

const SOURCE_BY_SLUG = Object.fromEntries(
  Object.entries(SOURCE_FILES).map(([file, slug]) => [slug, file])
)

function readSource(slug: string): string {
  const file = SOURCE_BY_SLUG[slug]
  if (file === undefined) throw new Error(`no source file registered for ${slug}`)
  return fs.readFileSync(path.join(DOCS_DIR, file), 'utf8')
}

/**
 * Guards the real `docs/pages` sources against the registry and the transform.
 *
 * Synthetic fixtures cannot catch a source file whose name drifted out of the
 * rsbuild copy map, or a heading level the renderer mangles, so these run over
 * the actual repository content.
 */
describe('docs/pages sources', () => {
  it('registers every markdown file that exists on disk', () => {
    const onDisk = fs
      .readdirSync(DOCS_DIR)
      .filter((name) => name.endsWith('.md'))
      .sort()
    expect(Object.keys(SOURCE_FILES).sort()).toEqual(onDisk)
  })

  it('gives every registry entry a matching source file', () => {
    for (const entry of DOC_ENTRIES) {
      const file = SOURCE_BY_SLUG[entry.slug]
      expect(file, `${entry.slug} has no source file registered`).toBeDefined()
      expect(fs.existsSync(path.join(DOCS_DIR, file as string))).toBe(true)
    }
  })

  it.each(DOC_ENTRIES.map((entry) => [entry.slug, entry] as const))(
    'renders %s without leaking custom component tags or dead links',
    (_slug, entry) => {
      const out = renderDocMarkdown(readSource(entry.slug))

      // Custom components must be fully expanded, not left as raw tags.
      expect(out).not.toMatch(/<Start(Grid|Step)/)

      // No link may still point at the original docs site's routes.
      expect(out).not.toMatch(/\]\(\/guides\//)
      expect(out).not.toMatch(/\]\(\/models\//)

      // The page renders its own H1, so the body must not duplicate it.
      expect(out.trimStart().startsWith('#')).toBe(false)

      expect(out.length).toBeGreaterThan(0)
    }
  )

  it('expands the onboarding grid in introduction.md', () => {
    const out = renderDocMarkdown(readSource('introduction'))
    expect(out).toContain('docs-grid')
    expect(out).toContain('docs-steps')
    expect(out).toContain('Create an API key')
    expect(out).toContain('href="/docs/quickstart"')
  })

  it('keeps fenced code blocks balanced in every document', () => {
    for (const entry of DOC_ENTRIES) {
      const out = renderDocMarkdown(readSource(entry.slug))
      const fences = out.match(/^(~{3,}|```)/gm) ?? []
      expect(
        fences.length % 2,
        `${entry.slug} has an unbalanced code fence`
      ).toBe(0)
    }
  })
})