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
import { Download, Images, Loader2 } from 'lucide-react'
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
import { Skeleton } from '@/components/ui/skeleton'

import type { GeneratedMedia } from '../types'

interface ResultGalleryProps {
  items: GeneratedMedia[]
  loading: boolean
  hasGenerated: boolean
}

/** Title + description shared by every state of the results card. */
function ResultHeader() {
  const { t } = useTranslation()
  return (
    <CardHeader className='border-b p-3 !pb-3 sm:p-5 sm:!pb-5'>
      <div className='flex items-center gap-3'>
        <IconBadge tone='chart-1' size='title'>
          <Images />
        </IconBadge>
        <div className='min-w-0'>
          <CardTitle className='text-base tracking-tight sm:text-lg'>
            {t('Results')}
          </CardTitle>
          <CardDescription className='text-xs'>
            {t('Images from your most recent generation.')}
          </CardDescription>
        </div>
      </div>
    </CardHeader>
  )
}

/** Renders the media returned by the most recent generation. */
export function ResultGallery({
  items,
  loading,
  hasGenerated,
}: ResultGalleryProps) {
  const { t } = useTranslation()

  if (loading) {
    return (
      <Card data-card-hover='false' className='gap-0 overflow-hidden py-0'>
        <ResultHeader />
        <CardContent className='p-3 sm:p-5'>
          <div className='grid gap-3 sm:grid-cols-2'>
            <Skeleton className='aspect-square w-full' />
            <Skeleton className='aspect-square w-full' />
          </div>
        </CardContent>
      </Card>
    )
  }

  if (items.length === 0) {
    return (
      <Card data-card-hover='false' className='gap-0 overflow-hidden py-0'>
        <ResultHeader />
        <CardContent className='p-3 sm:p-5'>
          <div className='bg-background/60 flex flex-col items-center gap-1 rounded-xl border p-8 text-center'>
            {hasGenerated ? (
              <>
                <p className='text-sm font-medium'>
                  {t('The request returned no images')}
                </p>
                <p className='text-muted-foreground text-xs'>
                  {t('Try a different model or prompt.')}
                </p>
              </>
            ) : (
              <>
                <Loader2
                  className='text-muted-foreground mb-1 size-5'
                  aria-hidden='true'
                />
                <p className='text-sm font-medium'>{t('Nothing here yet')}</p>
                <p className='text-muted-foreground text-xs'>
                  {t('Fill in a prompt and select Generate to see results.')}
                </p>
              </>
            )}
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card data-card-hover='false' className='gap-0 overflow-hidden py-0'>
      <ResultHeader />
      <CardContent className='p-3 sm:p-5'>
        <ul className='grid gap-3 sm:grid-cols-2'>
          {items.map((item) => (
            <li key={item.id} className='space-y-2'>
              <img
                src={item.url}
                alt={item.revisedPrompt ?? t('Generated image')}
                className='aspect-square w-full rounded-xl border object-cover'
              />
              {item.revisedPrompt && (
                <p className='text-muted-foreground line-clamp-2 text-xs'>
                  {item.revisedPrompt}
                </p>
              )}
              <Button
                variant='outline'
                size='sm'
                render={
                  <a
                    href={item.url}
                    download
                    target='_blank'
                    rel='noreferrer noopener'
                  />
                }
              >
                <Download data-icon='inline-start' />
                {t('Download')}
              </Button>
            </li>
          ))}
        </ul>
      </CardContent>
    </Card>
  )
}