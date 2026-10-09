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
import { memo } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

import { getRouteMeta } from '../constants'
import type { PricingModel, TokenUnit } from '../types'

export interface RoutePriceCardProps {
  model: PricingModel
  group: string
  ratio: number
  tokenUnit: TokenUnit
  showRechargePrice?: boolean
  priceRate?: number
  usdExchangeRate?: number
}

function formatPriceValue(val: number): string {
  if (val === 0) return '$0'
  if (Number.isInteger(val)) return `$${val}`
  const s = val.toFixed(6)
  return `$${parseFloat(s).toString()}`
}

function formatSavingsPercent(ratio: number): string {
  const pct = ratio * 100
  if (pct < 10 && pct % 1 !== 0) {
    return pct.toFixed(1).replace(/\.0$/, '')
  }
  return Math.round(pct).toString()
}

export const RoutePriceCard = memo(function RoutePriceCard({
  model,
  group,
  ratio,
  tokenUnit,
  showRechargePrice = false,
  priceRate = 1,
  usdExchangeRate = 1,
}: RoutePriceCardProps) {
  const { t } = useTranslation()
  const meta = getRouteMeta(group)
  const isTokenBased = model.quota_type === 0
  const divisor = tokenUnit === 'K' ? 1000 : 1
  const unitLabel = tokenUnit === 'K' ? '/1K' : '/1M'

  // Official prices
  const baseRate = model.model_ratio * 2
  const officialInput = (baseRate / divisor)
  const officialOutput = ((baseRate * (model.completion_ratio || 1)) / divisor)

  // Route prices
  let routeInput = officialInput * ratio
  let routeOutput = officialOutput * ratio

  if (showRechargePrice && priceRate > 0 && usdExchangeRate > 0) {
    routeInput = (routeInput * priceRate) / usdExchangeRate
    routeOutput = (routeOutput * priceRate) / usdExchangeRate
  }

  const multiplierText = `${Number((ratio).toFixed(3)).toString()}×`
  const savingsText = `≈ ${formatSavingsPercent(ratio)}% of official price`

  return (
    <div
      role='group'
      aria-label={group}
      data-route-kind={meta.kind}
      className={cn(
        'group/route-card flex flex-col justify-between rounded-xl border border-border/70 p-3.5 transition-all shadow-xs',
        'bg-card/80 hover:border-primary/40 hover:shadow-md hover:bg-card',
        meta.kind === 'award' && 'hover:border-emerald-500/40',
        meta.kind === 'limited' && 'hover:border-amber-500/40'
      )}
    >
      {/* Heading */}
      <div className='flex items-center justify-between gap-2'>
        <span
          className='truncate font-mono text-xs font-semibold text-foreground'
          title={group}
        >
          {group}
        </span>
        <div className='flex shrink-0 items-center gap-1.5'>
          <Badge
            variant='outline'
            className='h-5 gap-1 rounded-full border-border/80 px-2 py-0 text-[10px] font-medium text-foreground bg-muted/30'
          >
            <span className={cn('size-1.5 shrink-0 rounded-full', meta.color)} />
            {meta.tier}
          </Badge>
          <span className='inline-flex h-5 items-center rounded-full border border-border/60 bg-background px-1.5 font-mono text-[10px] font-medium text-muted-foreground'>
            {multiplierText}
          </span>
        </div>
      </div>

      {/* Description */}
      <p className='mt-2 min-h-[2.4rem] line-clamp-2 text-[11px] leading-relaxed text-muted-foreground'>
        {meta.description}
      </p>

      {/* Metric Values */}
      <div className='mt-2.5 grid grid-cols-2 gap-2 rounded-lg bg-muted/40 p-2 font-mono dark:bg-muted/15'>
        {isTokenBased ? (
          <>
            <div className='flex flex-col min-w-0'>
              <span className='font-sans text-[10px] font-medium text-muted-foreground'>
                {t('Input')}
              </span>
              <div className='mt-0.5 flex flex-wrap items-baseline gap-1'>
                <del className='text-[11px] text-muted-foreground/70'>
                  {formatPriceValue(officialInput)}
                </del>
                <strong className='text-xs font-bold text-foreground'>
                  {formatPriceValue(routeInput)}
                </strong>
                <small className='text-[10px] text-muted-foreground'>{unitLabel}</small>
              </div>
            </div>
            <div className='flex flex-col min-w-0'>
              <span className='font-sans text-[10px] font-medium text-muted-foreground'>
                {t('Output')}
              </span>
              <div className='mt-0.5 flex flex-wrap items-baseline gap-1'>
                <del className='text-[11px] text-muted-foreground/70'>
                  {formatPriceValue(officialOutput)}
                </del>
                <strong className='text-xs font-bold text-foreground'>
                  {formatPriceValue(routeOutput)}
                </strong>
                <small className='text-[10px] text-muted-foreground'>{unitLabel}</small>
              </div>
            </div>
          </>
        ) : (
          <div className='col-span-2 flex flex-col min-w-0'>
            <span className='font-sans text-[10px] font-medium text-muted-foreground'>
              {t('Per request')}
            </span>
            <div className='mt-0.5 flex flex-wrap items-baseline gap-1.5'>
              <del className='text-[11px] text-muted-foreground/70'>
                {formatPriceValue(model.model_price ?? 0)}
              </del>
              <strong className='text-xs font-bold text-foreground'>
                {formatPriceValue((model.model_price ?? 0) * ratio)}
              </strong>
              <small className='text-[10px] text-muted-foreground'>/{t('request')}</small>
            </div>
          </div>
        )}
      </div>

      {/* Savings Percent Footer */}
      <div className='mt-3 flex items-center justify-between border-t border-border/50 pt-2 font-mono text-[11px]'>
        <span className='font-medium text-emerald-500 dark:text-emerald-400'>
          {savingsText}
        </span>
      </div>
    </div>
  )
})
