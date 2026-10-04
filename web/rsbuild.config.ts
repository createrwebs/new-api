import path from 'node:path'
import { fileURLToPath } from 'node:url'

import { defineConfig, loadEnv } from '@rsbuild/core'
import { pluginReact } from '@rsbuild/plugin-react'
import { pluginTailwindcss } from '@rsbuild/plugin-tailwindcss'
import { tanstackRouter } from '@tanstack/router-plugin/rspack'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

/**
 * Markdown documents rendered by the `/docs` page.
 *
 * Source of truth is the sibling `docs/pages` directory in the repository
 * root. Rspack's copy plugin cannot flatten a glob, so each file is mapped
 * explicitly: that is also what lets the awkward source names (`Usage and
 * cost.md`, `Codex setup.md`, ...) become clean, URL-safe slugs that the
 * frontend registry can reference without encoding.
 *
 * Add a new document by appending to `DOC_SOURCE_FILES` and to
 * `src/features/docs/constants.ts`.
 */
const DOC_SOURCE_DIR = path.resolve(__dirname, '../docs/pages')

const DOC_SOURCE_FILES: Record<string, string> = {
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

const docCopyEntries = Object.entries(DOC_SOURCE_FILES).map(
  ([fileName, slug]) => ({
    from: path.join(DOC_SOURCE_DIR, fileName),
    to: `docs-content/${slug}.md`,
    noErrorOnMissing: true,
  })
)

export default defineConfig(({ envMode }) => {
  const env = loadEnv({ mode: envMode, prefixes: ['VITE_'] })
  const serverUrl =
    process.env.VITE_REACT_APP_SERVER_URL ||
    env.rawPublicVars.VITE_REACT_APP_SERVER_URL ||
    'http://localhost:3000'

  const isProd = envMode === 'production'
  const devProxy = Object.fromEntries(
    (['/api', '/v1', '/mj', '/pg'] as const).map((key) => [
      key,
      { target: serverUrl, changeOrigin: true },
    ])
  ) as Record<string, { target: string; changeOrigin: boolean }>

  return {
    plugins: [pluginReact(), pluginTailwindcss({ optimize: false })],
    // Rsbuild 2: replaces deprecated `performance.chunkSplit` (RSPack 2 aligned)
    splitChunks: {
      preset: 'default',
      cacheGroups: {
        'vendor-react': {
          test: /node_modules[\\/](react|react-dom)[\\/]/,
          name: 'vendor-react',
          chunks: 'all',
          priority: 0,
          enforce: true,
        },
        'vendor-ui-primitives': {
          test: /node_modules[\\/](@base-ui|@radix-ui)[\\/]/,
          name: 'vendor-ui-primitives',
          chunks: 'all',
          priority: 0,
          enforce: true,
        },
        'vendor-tanstack': {
          test: /node_modules[\\/]@tanstack[\\/]/,
          name: 'vendor-tanstack',
          chunks: 'all',
          priority: 0,
          enforce: true,
        },
      },
    },
    source: {
      entry: {
        index: './src/main.tsx',
      },
    },
    resolve: {
      alias: {
        '@': path.resolve(__dirname, './src'),
      },
    },
    html: {
      template: './index.html',
      favicon: './public/favicon.ico',
    },
    server: {
      host: '0.0.0.0',
      strictPort: false,
      proxy: devProxy,
      copy: docCopyEntries,
    },
    output: {
      // Production optimizations
      minify: isProd,
      target: 'web',
      distPath: {
        root: 'dist',
      },
      copy: docCopyEntries,
      // Rely on Rsbuild default legalComments ("linked" → per-chunk *.LICENSE.txt) in all modes.
      // Do not set "none" in production: that strips minifier-preserved third-party notices and
      // extracted license files, which some distributions require for open-source compliance.
    },
    performance: {
      // Remove console in production
      removeConsole: isProd ? ['log'] : false,
      buildCache: false,
    },
    tools: {
      rspack: {
        plugins: [
          tanstackRouter({
            target: 'react',
            // Dev: avoid per-route async chunks (reduces white flash on navigation + faster HMR feedback).
            // Prod: keep route-based code splitting.
            autoCodeSplitting: isProd,
          }),
        ],
      },
    },
  }
})
