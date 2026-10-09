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
import { ChevronDown } from 'lucide-react'
import { memo } from 'react'
import { useTranslation } from 'react-i18next'

import { Accordion } from '@/components/ui/accordion'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { getLobeIcon } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'

import type { PricingModel, TokenUnit } from '../types'
import { ModelflareModelCard } from './modelflare-model-card'

export interface VendorGroupSectionProps {
  vendorName: string
  vendorIconKey?: string
  models: PricingModel[]
  tokenUnit: TokenUnit
  showRechargePrice?: boolean
  priceRate?: number
  usdExchangeRate?: number
  groupRatio?: Record<string, number>
  onModelClick: (modelName: string) => void
  isCollapsed: boolean
  onToggleCollapse: () => void
  expandedModelNames: string[]
  onExpandedModelsChange: (names: string[]) => void
}

export const VendorGroupSection = memo(function VendorGroupSection({
  vendorName,
  vendorIconKey,
  models,
  tokenUnit,
  showRechargePrice,
  priceRate,
  usdExchangeRate,
  groupRatio,
  onModelClick,
  isCollapsed,
  onToggleCollapse,
  expandedModelNames,
  onExpandedModelsChange,
}: VendorGroupSectionProps) {
  const { t } = useTranslation()
  const vendorIcon = vendorIconKey ? getLobeIcon(vendorIconKey, 24) : null

  return (
    <section
      data-vendor-group={vendorName}
      className='flex flex-col gap-2 rounded-2xl border border-border/80 bg-card/40 p-3 sm:p-4 shadow-2xs transition-colors'
    >
      {/* Group Header (Clickable to collapse/expand) */}
      <div
        role='button'
        tabIndex={0}
        data-vendor-header={vendorName}
        aria-expanded={!isCollapsed}
        onClick={onToggleCollapse}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault()
            onToggleCollapse()
          }
        }}
        className='group/vendor-header flex items-center justify-between cursor-pointer select-none rounded-xl p-1.5 transition-colors hover:bg-muted/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring'
      >
        <div className='flex items-center gap-2.5 sm:gap-3'>
          <div
            aria-hidden
            className='size-8 shrink-0 rounded-lg bg-muted/80 flex items-center justify-center overflow-hidden border border-border/60'
          >
            {vendorIcon || (
              <span className='font-mono text-sm font-bold text-foreground'>
                {vendorName.charAt(0).toUpperCase()}
              </span>
            )}
          </div>
          <h2 className='text-base sm:text-lg font-bold tracking-tight text-foreground flex items-center gap-2 font-mono'>
            {vendorName}
            <Badge
              variant='secondary'
              className='font-mono text-xs font-semibold px-2 py-0.5 rounded-full'
            >
              {models.length}
            </Badge>
          </h2>
        </div>

        <div className='flex items-center gap-2'>
          <Button
            type='button'
            variant='ghost'
            size='sm'
            className='h-8 gap-1.5 px-2.5 text-xs font-medium text-muted-foreground group-hover/vendor-header:text-foreground'
            onClick={(e) => {
              e.stopPropagation()
              onToggleCollapse()
            }}
          >
            <span className='hidden sm:inline'>
              {isCollapsed ? t('Expand') : t('Collapse')}
            </span>
            <ChevronDown
              className={cn(
                'size-4 transition-transform duration-200 text-muted-foreground',
                !isCollapsed && 'rotate-180'
              )}
            />
          </Button>
        </div>
      </div>

      {/* Models List Accordion */}
      {!isCollapsed && (
        <div className='mt-1 flex flex-col'>
          <Accordion
            multiple
            value={expandedModelNames}
            onValueChange={onExpandedModelsChange}
          >
            {models.map((model) => (
              <ModelflareModelCard
                key={model.id ?? model.model_name}
                model={model}
                tokenUnit={tokenUnit}
                showRechargePrice={showRechargePrice}
                priceRate={priceRate}
                usdExchangeRate={usdExchangeRate}
                groupRatio={groupRatio}
                onModelClick={onModelClick}
              />
            ))}
          </Accordion>
        </div>
      )}
    </section>
  )
})
