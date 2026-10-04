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
 * Markdown sources in `docs/pages` are written for a richer docs tool and use
 * two constructs plain CommonMark does not understand:
 *
 *   1. `<StartGrid actions={…}><StartStep … /></StartGrid>` — numbered
 *      onboarding cards plus action buttons.
 *   2. Two-space-indented blocks — a bold label on a 2-space line with an
 *      indented description or fenced code block underneath (used for the
 *      cURL / Python / Node.js samples and the feature grids).
 *
 * Both are rewritten to plain HTML here so the shared `Markdown` component can
 * render them without inventing React components for one-off markup.
 */

const START_GRID_RE =
  /<StartGrid\s+actions=\{([\s\S]*?)\}\s*>([\s\S]*?)<\/StartGrid>/g
const START_STEP_RE = /<StartStep\s+([^>]*?)\/>/g
const ATTR_RE = /([a-zA-Z][\w-]*)\s*=\s*"([^"]*)"/g

/**
 * Links written against the original docs site's routes.
 *
 * Anything with a local page is repointed at `/docs/<slug>`; the rest fall back
 * to the closest local surface so a reader never lands on a 404.
 */
const INTERNAL_LINK_MAP: Record<string, string> = {
  '/quickstart/': '/docs/quickstart',
  '/guides/endpoints/': '/docs/endpoint-to-call',
  '/guides/authentication/': '/docs/authentication',
  '/guides/errors/': '/docs/errors',
  '/guides/models-and-groups/': '/docs/models-and-groups',
  '/guides/usage-and-cost/': '/docs/usage-and-cost',
  '/guides/clients/codex/': '/docs/codex-setup',
  '/guides/clients/codex/start': '/docs/codex-setup',
  '/guides/clients/claude-code/': '/docs/claude-code-setup',
  '/guides/clients/claude-code/start': '/docs/claude-code-setup',
  '/guides/clients/gemini-cli/': '/docs/gemini-cli-setup',
  '/guides/clients/grok-build/': '/docs/grok-build-setup',
  '/guides/clients/opencode/': '/docs/opencode-setup',
  '/models/': '/pricing',
  '/models/image/': '/pricing',
  '/models/video/': '/pricing',
  '/guides/features/': '/docs/introduction',
  '/guides/clients/media-skill/': '/workbench',
  '/guides/video/seedance-asset-management/': '/docs/endpoint-to-call',
}

function escapeText(value: string): string {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
}

function resolveInternalHref(href: string): string {
  return INTERNAL_LINK_MAP[href] ?? href
}

function rewriteInternalLinks(markdown: string): string {
  return markdown.replaceAll(
    /\]\((\/[^)\s]*)\)/g,
    (_match, href: string) => `](${resolveInternalHref(href)})`
  )
}

interface StartAction {
  href: string
  label: string
  primary: boolean
}

/** Parses `[{ href: '/a/', label: 'A', primary: true }, …]` from the prop. */
function parseStartActions(raw: string): StartAction[] {
  const actions: StartAction[] = []
  for (const match of raw.matchAll(/\{([^{}]*)\}/g)) {
    const body = match[1]
    const href = /href:\s*'([^']*)'/.exec(body)?.[1]
    if (!href) continue
    actions.push({
      href,
      label: /label:\s*'([^']*)'/.exec(body)?.[1] ?? href,
      primary: /primary:\s*true/.test(body),
    })
  }
  return actions
}

/** Renders one `<StartGrid>` block into buttons plus numbered step cards. */
function renderStartGrid(actionsRaw: string, bodyRaw: string): string {
  const buttons = parseStartActions(actionsRaw)
    .map((action) => {
      const className = action.primary
        ? 'docs-grid-action docs-grid-action-primary'
        : 'docs-grid-action'
      return `<a class="${className}" href="${escapeText(
        resolveInternalHref(action.href)
      )}">${escapeText(action.label)}</a>`
    })
    .join('')

  const steps = [...bodyRaw.matchAll(START_STEP_RE)]
    .map((match) => {
      const attrs: Record<string, string> = {}
      for (const attr of match[1].matchAll(ATTR_RE)) attrs[attr[1]] = attr[2]
      return [
        '<div class="docs-step">',
        attrs.number
          ? `<span class="docs-step-number">${escapeText(attrs.number)}</span>`
          : '',
        '<div class="docs-step-body">',
        `<p class="docs-step-title">${escapeText(attrs.title ?? '')}</p>`,
        `<p class="docs-step-text">${escapeText(attrs.description ?? '')}</p>`,
        '</div>',
        '</div>',
      ].join('')
    })
    .join('')

  return `<div class="docs-grid">${buttons}<div class="docs-steps">${steps}</div></div>`
}
/**
 * Rewrites indented blocks into titled cards.
 *
 * The sources use two shapes, both handled here:
 *   - A 2-space label followed by a top-level fenced block (the cURL /
 *     Python / Node.js samples in `quickstart.md`).
 *   - Runs of 4-space lines, where the first line is the card title and the
 *     rest is its body (the feature grids in `introduction.md`).
 *
 * Card boundaries matter: in `introduction.md` consecutive cards are separated
 * by a line holding exactly two spaces, while a genuinely empty line is only
 * a soft break inside a card. Treating both the same merges every feature card
 * into one, so the two are distinguished explicitly.
 *
 * An indented line that begins a markdown list stays a list — folding it into
 * a card would emit the raw `- [text](url): note` source as paragraph text.
 *
 * Fence state is tracked so blank lines and column-zero lines inside a code
 * sample are never mistaken for card boundaries.
 */
const INDENTED_RE = /^\s{2,}\S/
const FENCE_RE = /^\s*(~{3,}|```)/
const LIST_ITEM_RE = /^\s*([-*+]|\d+[.)])\s/
/** A line of one or two spaces: the card separator used by these sources. */
const SEPARATOR_RE = /^\s{1,2}$/

function renderIndentedCards(markdown: string): string {
  const lines = markdown.split('\n')
  const output: string[] = []
  let index = 0
  let inFence = false

  while (index < lines.length) {
    if (FENCE_RE.test(lines[index])) inFence = !inFence

    const line = lines[index]
    // Indented lists stay lists so the markdown parser can render them.
    if (inFence || !INDENTED_RE.test(line) || LIST_ITEM_RE.test(line)) {
      output.push(line)
      index += 1
      continue
    }

    // The card must actually have content: a deeper-indented line or a fence.
    let lookahead = index + 1
    while (lookahead < lines.length && lines[lookahead].trim() === '') {
      lookahead += 1
    }
    const next = lines[lookahead]
    if (
      next === undefined ||
      (!INDENTED_RE.test(next) && !FENCE_RE.test(next))
    ) {
      output.push(line)
      index += 1
      continue
    }

    const title = line.trim()
    const body: string[] = []
    let fenceDepth = 0
    index += 1

    while (index < lines.length) {
      const candidate = lines[index]

      // Two-space separator: closes the card and is not part of it.
      if (SEPARATOR_RE.test(candidate)) break

      if (FENCE_RE.test(candidate)) {
        fenceDepth += 1
        body.push(candidate.replace(/^\s{0,2}/, ''))
        index += 1
        continue
      }
      if (fenceDepth > 0) {
        body.push(candidate)
        index += 1
        continue
      }

      if (candidate === '') {
        // Genuinely empty line: keep it only when more card content follows.
        let after = index + 1
        while (after < lines.length && lines[after] === '') after += 1
        const upcoming = lines[after]
        if (
          upcoming === undefined ||
          (!INDENTED_RE.test(upcoming) && !FENCE_RE.test(upcoming))
        ) {
          break
        }
        body.push('')
        index = after
        continue
      }

      // Dedent back to column zero ends the card.
      if (!INDENTED_RE.test(candidate)) break

      body.push(candidate.replace(/^ {2}/, ''))
      index += 1
    }

    const hasFence = body.some((entry) => FENCE_RE.test(entry))
    const bodyHtml = hasFence
      ? body.join('\n')
      : `<p>${escapeText(body.join(' ').trim())}</p>`

    output.push(
      `<div class="docs-card"><p class="docs-card-title">${escapeText(
        title
      )}</p><div class="docs-card-body">${bodyHtml}</div></div>`
    )
  }

  return output.join('\n')
}

/**
 * Normalizes a `docs/pages` markdown file for the shared `Markdown` renderer.
 *
 * The leading H1 is dropped because the page renders its own heading from the
 * registry, avoiding a duplicated title.
 */
export function renderDocMarkdown(source: string): string {
  let markdown = source.replaceAll('\r\n', '\n').trim()
  markdown = markdown.replace(/^#\s+.*\n+/, '')
  markdown = rewriteInternalLinks(markdown)
  markdown = markdown.replace(START_GRID_RE, (_raw, actions, body) =>
    renderStartGrid(actions, body)
  )
  return renderIndentedCards(markdown)
}