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
import { Loader2, Sparkles, Wand2 } from 'lucide-react'
import { useId } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { IconBadge } from '@/components/ui/icon-badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  NativeSelect,
  NativeSelectOption,
} from '@/components/ui/native-select'
import { Textarea } from '@/components/ui/textarea'

const IMAGE_SIZES = ['1024x1024', '1024x1792', '1792x1024', '512x512']
const IMAGE_COUNTS = [1, 2, 3, 4]

export interface GenerationFormProps {
  modelIds: string[]
  model: string
  onModelChange: (value: string) => void
  prompt: string
  onPromptChange: (value: string) => void
  size: string
  onSizeChange: (value: string) => void
  count: number
  onCountChange: (value: number) => void
  apiKey: string
  onApiKeyChange: (value: string) => void
  onSubmit: () => void
  loading: boolean
  disabled: boolean
}

/**
 * Prompt form for `/v1/images/generations`.
 *
 * Only image generation is exposed: `/v1/videos` returns a task handle rather
 * than media, so it does not belong on a page that renders a result grid.
 */
export function GenerationForm(props: GenerationFormProps) {
  const { t } = useTranslation()
  const id = useId()
  const disabled = props.disabled || props.loading

  return (
    <Card data-card-hover='false' className='gap-0 overflow-hidden py-0'>
      <CardHeader className='border-b p-3 !pb-3 sm:p-5 sm:!pb-5'>
        <div className='flex items-center gap-3'>
          <IconBadge tone='chart-2' size='title'>
            <Wand2 />
          </IconBadge>
          <div className='min-w-0'>
            <CardTitle className='text-base tracking-tight sm:text-lg'>
              {t('Generate an image')}
            </CardTitle>
            <CardDescription className='text-xs'>
              {t('Runs against /v1/images/generations using your API key.')}
            </CardDescription>
          </div>
        </div>
      </CardHeader>

      <CardContent className='space-y-4 p-3 sm:p-5'>
        <div className='space-y-1.5'>
          <Label htmlFor={`${id}-prompt`}>{t('Prompt')}</Label>
          <Textarea
            id={`${id}-prompt`}
            value={props.prompt}
            onChange={(event) => props.onPromptChange(event.target.value)}
            placeholder={t('A neon-lit skyline above the clouds')}
            rows={4}
            maxLength={4000}
            disabled={disabled}
          />
        </div>

        <div className='grid gap-3 sm:grid-cols-3'>
          <div className='space-y-1.5'>
            <Label htmlFor={`${id}-model`}>{t('Model')}</Label>
            <NativeSelect
              id={`${id}-model`}
              className='w-full'
              value={props.model}
              disabled={disabled || props.modelIds.length === 0}
              onChange={(event) => props.onModelChange(event.target.value)}
            >
              {props.modelIds.length === 0 ? (
                <NativeSelectOption value='' disabled>
                  {t('No image models available')}
                </NativeSelectOption>
              ) : (
                props.modelIds.map((modelId) => (
                  <NativeSelectOption key={modelId} value={modelId}>
                    {modelId}
                  </NativeSelectOption>
                ))
              )}
            </NativeSelect>
          </div>

          <div className='space-y-1.5'>
            <Label htmlFor={`${id}-size`}>{t('Size')}</Label>
            <NativeSelect
              id={`${id}-size`}
              className='w-full'
              value={props.size}
              disabled={disabled}
              onChange={(event) => props.onSizeChange(event.target.value)}
            >
              {IMAGE_SIZES.map((size) => (
                <NativeSelectOption key={size} value={size}>
                  {size}
                </NativeSelectOption>
              ))}
            </NativeSelect>
          </div>

          <div className='space-y-1.5'>
            <Label htmlFor={`${id}-count`}>{t('Count')}</Label>
            <NativeSelect
              id={`${id}-count`}
              className='w-full'
              value={String(props.count)}
              disabled={disabled}
              onChange={(event) =>
                props.onCountChange(Number(event.target.value))
              }
            >
              {IMAGE_COUNTS.map((count) => (
                <NativeSelectOption key={count} value={String(count)}>
                  {String(count)}
                </NativeSelectOption>
              ))}
            </NativeSelect>
          </div>
        </div>
<div className='space-y-1.5'>
          <Label htmlFor={`${id}-key`}>{t('API key')}</Label>
          <Input
            id={`${id}-key`}
            type='password'
            autoComplete='off'
            spellCheck={false}
            placeholder='sk-...'
            value={props.apiKey}
            onChange={(event) => props.onApiKeyChange(event.target.value)}
            disabled={disabled}
          />
          <p className='text-muted-foreground text-xs'>
            {t('Used for this request only and never stored by the workbench.')}
          </p>
        </div>

        <Button onClick={props.onSubmit} disabled={disabled}>
          {props.loading ? (
            <Loader2 className='animate-spin' data-icon='inline-start' />
          ) : (
            <Sparkles data-icon='inline-start' />
          )}
          {props.loading ? t('Generating...') : t('Generate')}
        </Button>
      </CardContent>
    </Card>
  )
}