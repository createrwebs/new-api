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
import { describe, expect, it } from 'vitest'

import { renderDocMarkdown } from '../markdown'

describe('renderDocMarkdown', () => {
  it('drops the leading H1 so the page header is not duplicated', () => {
    expect(renderDocMarkdown('# Quickstart\n\nBody text.')).not.toContain('#')
    expect(renderDocMarkdown('# Quickstart\n\nBody text.')).toContain(
      'Body text.'
    )
  })

  it('rewrites internal docs links to their local /docs routes', () => {
    const out = renderDocMarkdown(
      'See [Which endpoint](/guides/endpoints/) and [Errors](/guides/errors/).'
    )
    expect(out).toContain('](/docs/endpoint-to-call)')
    expect(out).toContain('](/docs/errors)')
  })

  it('sends model-catalog links to the pricing page instead of a dead route', () => {
    const out = renderDocMarkdown('See [Video models](/models/video/).')
    expect(out).toContain('](/pricing)')
  })

  it('leaves external links untouched', () => {
    const out = renderDocMarkdown('Read [the docs](https://example.com/a).')
    expect(out).toContain('](https://example.com/a)')
  })

  it('expands StartGrid into action links and numbered step cards', () => {
    const out = renderDocMarkdown(
      [
        '# Intro',
        '',
        '<StartGrid',
        "  actions={[",
        "    { href: '/quickstart/', label: 'Quickstart', primary: true },",
        "    { href: '/guides/endpoints/', label: 'Which endpoint' }",
        '  ]}',
        '>',
        '  <StartStep number="01" title="Create an API key" description="Sign in." />',
        '</StartGrid>',
      ].join('\n')
    )

    expect(out).not.toContain('StartGrid')
    expect(out).not.toContain('StartStep')
    expect(out).toContain('docs-grid-action-primary')
    expect(out).toContain('href="/docs/quickstart"')
    expect(out).toContain('href="/docs/endpoint-to-call"')
    expect(out).toContain('docs-step-number')
    expect(out).toContain('Create an API key')
  })

  it('turns an indented label plus indented text into a card', () => {
    const out = renderDocMarkdown(
      ['## What it provides', '', '  One key', '    Shared across families', ''].join(
        '\n'
      )
    )
    expect(out).toContain('docs-card-title')
    expect(out).toContain('One key')
    expect(out).toContain('Shared across families')
  })

  it('keeps a fenced code block inside a card as markup', () => {
    const out = renderDocMarkdown(
      [
        '## Samples',
        '',
        '  **cURL**',
        '',
        '```bash',
        'curl https://example.com',
        '```',
        '',
        'Then run it.',
      ].join('\n')
    )
    expect(out).toContain('docs-card-title')
    expect(out).toContain('```bash')
    expect(out).toContain('curl https://example.com')
    // The paragraph after the card must not be swallowed into it.
    expect(out).toContain('Then run it.')
  })

  it('does not treat a normal indented list as a card', () => {
    const out = renderDocMarkdown('- one\n- two\n')
    expect(out).not.toContain('docs-card')
  })
})