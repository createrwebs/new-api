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
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import {
  CardStaggerContainer,
  CardStaggerItem,
} from '@/components/page-transition'
import { handleServerError } from '@/lib/handle-server-error'

import { generateImage, getWorkbenchModels } from './api'
import { GenerationForm } from './components/generation-form'
import { ResultGallery } from './components/result-gallery'
import type { GeneratedMedia } from './types'

const DEFAULT_SIZE = '1024x1024'

/**
 * `/workbench` — drive the gateway's image models without leaving the console.
 *
 * The catalog comes from `/api/pricing` (the only source that reports
 * `supported_endpoint_types`), and generation runs through the real
 * `/v1/images/generations` relay endpoint so billing and rate limits behave
 * exactly as they do for an external client.
 */
export function Workbench() {
  const { t } = useTranslation()
  const [prompt, setPrompt] = useState('')
  const [model, setModel] = useState('')
  const [size, setSize] = useState(DEFAULT_SIZE)
  const [count, setCount] = useState(1)
  const [apiKey, setApiKey] = useState('')
  const [items, setItems] = useState<GeneratedMedia[]>([])
  const [hasGenerated, setHasGenerated] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  const modelsQuery = useQuery({
    queryKey: ['workbench', 'models'],
    queryFn: getWorkbenchModels,
    staleTime: 5 * 60 * 1000,
  })

  const imageModelIds = useMemo(
    () =>
      (modelsQuery.data ?? [])
        .filter((entry) => entry.kind === 'image')
        .map((entry) => entry.id),
    [modelsQuery.data]
  )

  const videoModelIds = useMemo(
    () =>
      (modelsQuery.data ?? [])
        .filter((entry) => entry.kind === 'video')
        .map((entry) => entry.id),
    [modelsQuery.data]
  )

  // Default to the first image model once the catalog resolves, but never
  // overwrite a selection the user already made.
  const selectedModel =
    imageModelIds.includes(model) && model
      ? model
      : (imageModelIds[0] ?? '')

  const canSubmit =
    Boolean(selectedModel) &&
    prompt.trim().length > 0 &&
    apiKey.trim().length > 0 &&
    !submitting

  const handleSubmit = async () => {
    if (!canSubmit) return
    setSubmitting(true)
    try {
      const result = await generateImage({
        apiKey: apiKey.trim(),
        model: selectedModel,
        prompt: prompt.trim(),
        count,
        size,
      })
      setItems(result.items)
      setHasGenerated(true)
    } catch (error) {
      handleServerError(error, t('Generation failed'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('Image/Video Workbench')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <p className='text-muted-foreground text-xs'>
          {videoModelIds.length > 0
            ? t('{{count}} video models available via the task API', {
                count: videoModelIds.length,
              })
            : t('Generate media with your own API key.')}
        </p>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='mx-auto w-full max-w-7xl'>
          <CardStaggerContainer className='grid gap-4 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]'>
            <CardStaggerItem>
              <GenerationForm
                modelIds={imageModelIds}
                model={selectedModel}
                onModelChange={setModel}
                prompt={prompt}
                onPromptChange={setPrompt}
                size={size}
                onSizeChange={setSize}
                count={count}
                onCountChange={setCount}
                apiKey={apiKey}
                onApiKeyChange={setApiKey}
                onSubmit={handleSubmit}
                loading={submitting}
                disabled={modelsQuery.isLoading}
              />
            </CardStaggerItem>

            <CardStaggerItem>
              <ResultGallery
                items={items}
                loading={submitting}
                hasGenerated={hasGenerated}
              />
            </CardStaggerItem>
          </CardStaggerContainer>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}