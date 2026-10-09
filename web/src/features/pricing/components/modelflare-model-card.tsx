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
import { ArrowUpRight, Check, Copy } from 'lucide-react'
import { memo, useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import {
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'
import { getLobeIcon } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'

import { parseTags } from '../lib/filters'
import type { PricingModel, TokenUnit } from '../types'
import { RoutePriceCard } from './route-price-card'

export interface ModelflareModelCardProps {
  model: PricingModel
  tokenUnit: TokenUnit
  showRechargePrice?: boolean
  priceRate?: number
  usdExchangeRate?: number
  groupRatio?: Record<string, number>
  onModelClick: (modelName: string) => void
}

export const ModelflareModelCard = memo(function ModelflareModelCard({
  model,
  tokenUnit,
  showRechargePrice = false,
  priceRate = 1,
  usdExchangeRate = 1,
  groupRatio = {},
  onModelClick,
}: ModelflareModelCardProps) {
  const { t } = useTranslation()
  const { copiedText, copyToClipboard } = useCopyToClipboard({ notify: false })
  const isCopied = copiedText === model.model_name
  const groups = model.enable_groups?.length ? model.enable_groups : ['default']
  const tags = useMemo(() => parseTags(model.tags), [model.tags])
  const modelIconKey = model.icon || model.vendor_icon
  const modelIcon = modelIconKey ? getLobeIcon(modelIconKey, 24) : null
  const initial = model.model_name?.charAt(0).toUpperCase() || '?'

  // Context window parsing
  const contextWindow = useMemo(() => {
    if (model.description?.toLowerCase().includes('context window:')) {
      const match = model.description.match(/context window:\s*([^,\.]+)/i)
      return match ? match[1].trim() : null
    }
    if (model.context_length) {
      if (model.context_length >= 1000000) {
        return `${(model.context_length / 1000000).toFixed(2).replace(/\.?0+$/, '')}M`
      }
      return `${Math.round(model.context_length / 1000)}K`
    }
    return null
  }, [model.description, model.context_length])

  // Filter out any tag that is already shown as context window
  const displayTags = useMemo(() => {
    return tags.filter(
      (tag) => !tag.toLowerCase().startsWith('context window')
    )
  }, [tags])

  const visibleTags = displayTags.slice(0, 2)
  const remainingTagsCount = displayTags.length - visibleTags.length

  return (
    <AccordionItem
      value={model.model_name}
      className='mb-2 overflow-hidden rounded-xl border border-border/70 bg-card/60 transition-all hover:border-border'
    >
      <AccordionTrigger className='group/trigger px-3.5 py-2.5 hover:bg-muted/40 hover:no-underline sm:px-4 sm:py-3'>
        <div className='flex flex-1 items-center justify-between gap-3 min-w-0 pr-2'>
          {/* Left: Model Identity */}
          <div className='flex items-center gap-2.5 min-w-0 max-w-[50%] sm:max-w-none'>
            <div
              aria-hidden
              className='size-7 shrink-0 rounded-full bg-muted/70 flex items-center justify-center overflow-hidden border border-border/40'
            >
              {modelIcon || (
                <span className='font-mono text-xs font-bold text-muted-foreground'>
                  {initial}
                </span>
              )}
            </div>
            <span
              className='truncate font-mono text-sm font-semibold text-foreground'
              title={model.model_name}
            >
              {model.model_name}
            </span>
            <span
              role='button'
              tabIndex={0}
              title={isCopied ? t('Copied!') : t('Copy model name')}
              onClick={(e) => {
                e.stopPropagation()
                copyToClipboard(model.model_name)
              }}
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.stopPropagation()
                  copyToClipboard(model.model_name)
                }
              }}
              className='inline-flex size-6 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:text-foreground opacity-60 hover:opacity-100 hover:bg-muted/50 cursor-pointer'
            >
              {isCopied ? (
                <Check className='size-3 text-emerald-500' />
              ) : (
                <Copy className='size-3' />
              )}
            </span>
          </div>

          {/* Middle: Capabilities / Tags */}
          <div className='hidden sm:flex flex-wrap items-center gap-1.5 min-w-0'>
            {contextWindow && (
              <Badge
                variant='outline'
                className='h-5 rounded-md border-border/70 bg-background/50 px-2 font-mono text-[11px] font-normal text-muted-foreground'
              >
                Context window: {contextWindow}
              </Badge>
            )}
            {visibleTags.map((tag) => (
              <Badge
                key={tag}
                variant='outline'
                className='h-5 rounded-md border-border/70 bg-background/50 px-2 font-mono text-[11px] font-normal text-muted-foreground'
              >
                {tag}
              </Badge>
            ))}
            {remainingTagsCount > 0 && (
              <Badge
                variant='outline'
                className='h-5 rounded-md border-border/70 bg-background/50 px-1.5 font-mono text-[11px] font-normal text-muted-foreground'
              >
                +{remainingTagsCount}
              </Badge>
            )}
          </div>

          {/* Right: Route count badge */}
          <div className='flex items-center gap-1.5 shrink-0 ml-auto sm:ml-0'>
            <Badge
              variant='secondary'
              className='h-5 rounded-md px-1.5 font-mono text-xs font-semibold'
            >
              {groups.length}
            </Badge>
            <span className='font-mono text-xs text-muted-foreground hidden md:inline'>
              {groups.length === 1 ? t('Routing group') : t('Routing groups')}
            </span>
          </div>
        </div>
      </AccordionTrigger>

      <AccordionContent className='border-t border-border/60 bg-muted/20 pb-0'>
        {/* Intro Legend Bar */}
        <div className='flex flex-wrap items-center justify-between gap-2 border-b border-border/50 bg-muted/30 px-3.5 py-2 sm:px-4'>
          <div className='flex flex-wrap items-baseline gap-3 text-xs'>
            <strong className='font-mono text-[11px] font-bold uppercase tracking-wider text-foreground'>
              {t('Routing groups')}
            </strong>
            <span className='inline-flex items-center gap-2 font-mono text-[11px] text-muted-foreground'>
              <del className='text-muted-foreground/70'>
                {t('Official API list price')}
              </del>
              <span className='font-semibold text-foreground'>
                {t('Pay vs official')}
              </span>
            </span>
          </div>

          <Button
            type='button'
            variant='ghost'
            size='sm'
            className='h-7 gap-1 px-2 text-xs font-medium text-muted-foreground hover:text-foreground'
            onClick={() => onModelClick(model.model_name)}
          >
            <span>{t('View details')}</span>
            <ArrowUpRight className='size-3.5' />
          </Button>
        </div>

        {/* Route Cards Grid */}
        <div className='grid grid-cols-1 gap-3 p-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 sm:p-4'>
          {groups.map((group) => (
            <RoutePriceCard
              key={group}
              model={model}
              group={group}
              ratio={groupRatio[group] ?? 1}
              tokenUnit={tokenUnit}
              showRechargePrice={showRechargePrice}
              priceRate={priceRate}
              usdExchangeRate={usdExchangeRate}
            />
          ))}
        </div>
      </AccordionContent>
    </AccordionItem>
  )
})
