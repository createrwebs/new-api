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
import { ChevronsDown, ChevronsUp } from 'lucide-react'
import { memo, useCallback, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

import { DEFAULT_TOKEN_UNIT } from '../constants'
import type { PricingModel, PricingVendor, TokenUnit } from '../types'
import { VendorGroupSection } from './vendor-group-section'

export interface GroupedPricingCatalogProps {
  models: PricingModel[]
  onModelClick: (modelName: string) => void
  priceRate?: number
  usdExchangeRate?: number
  tokenUnit?: TokenUnit
  showRechargePrice?: boolean
  selectedGroup?: string
  vendors?: PricingVendor[]
  groupRatio?: Record<string, number>
}

const VENDOR_ORDER = [
  'OpenAI',
  'Anthropic',
  'Google',
  'TypeSafe',
  'Meta',
  'xAI',
  'DeepSeek',
  'Zhipu AI',
  'Alibaba Cloud',
  'Moonshot AI',
]

function getVendorSortRank(name: string): number {
  const idx = VENDOR_ORDER.indexOf(name)
  return idx === -1 ? 999 : idx
}

export const GroupedPricingCatalog = memo(function GroupedPricingCatalog(
  props: GroupedPricingCatalogProps
) {
  const { t } = useTranslation()
  const tokenUnit = props.tokenUnit ?? DEFAULT_TOKEN_UNIT
  const [selectedVendorFilter, setSelectedVendorFilter] = useState<string>('all')
  const [collapsedVendors, setCollapsedVendors] = useState<Set<string>>(new Set())

  // Default: first model of the first vendor expanded
  const [expandedModelNames, setExpandedModelNames] = useState<string[]>(() => {
    return props.models.length > 0 ? [props.models[0].model_name] : []
  })

  // Map of vendor id/name to vendor icon
  const vendorIconMap = useMemo(() => {
    const map = new Map<string, string>()
    for (const v of props.vendors ?? []) {
      if (v.name && v.icon) {
        map.set(v.name.toLowerCase(), v.icon)
      }
    }
    return map
  }, [props.vendors])

  // Group models by vendor
  const vendorGroups = useMemo(() => {
    const groupsMap = new Map<string, PricingModel[]>()

    for (const model of props.models) {
      const vendorName = model.vendor_name || 'Other'
      if (!groupsMap.has(vendorName)) {
        groupsMap.set(vendorName, [])
      }
      groupsMap.get(vendorName)!.push(model)
    }

    const groupsList: {
      name: string
      iconKey?: string
      models: PricingModel[]
    }[] = []

    for (const [name, modelList] of groupsMap.entries()) {
      const iconKey =
        vendorIconMap.get(name.toLowerCase()) ||
        modelList[0]?.vendor_icon ||
        modelList[0]?.icon ||
        name
      groupsList.push({
        name,
        iconKey,
        models: modelList,
      })
    }

    groupsList.sort((a, b) => {
      const rankA = getVendorSortRank(a.name)
      const rankB = getVendorSortRank(b.name)
      if (rankA !== rankB) return rankA - rankB
      return a.name.localeCompare(b.name)
    })

    return groupsList
  }, [props.models, vendorIconMap])

  // Filtered groups based on quick vendor filter
  const displayedGroups = useMemo(() => {
    if (selectedVendorFilter === 'all') {
      return vendorGroups
    }
    return vendorGroups.filter(
      (g) => g.name.toLowerCase() === selectedVendorFilter.toLowerCase()
    )
  }, [vendorGroups, selectedVendorFilter])

  // Toggle collapse for single vendor
  const toggleVendorCollapse = useCallback((vendorName: string) => {
    setCollapsedVendors((prev) => {
      const next = new Set(prev)
      if (next.has(vendorName)) {
        next.delete(vendorName)
      } else {
        next.add(vendorName)
      }
      return next
    })
  }, [])

  // Check if all displayed vendors are collapsed
  const allVendorsCollapsed = useMemo(() => {
    if (displayedGroups.length === 0) return false
    return displayedGroups.every((g) => collapsedVendors.has(g.name))
  }, [displayedGroups, collapsedVendors])

  // Expand / Collapse all vendors & models
  const toggleAllVendorsAndModels = useCallback(() => {
    if (allVendorsCollapsed) {
      // Expand everything
      setCollapsedVendors(new Set())
      const allModelNames = props.models.map((m) => m.model_name)
      setExpandedModelNames(allModelNames)
    } else {
      // Collapse everything
      const allVendorNames = new Set(displayedGroups.map((g) => g.name))
      setCollapsedVendors(allVendorNames)
      setExpandedModelNames([])
    }
  }, [allVendorsCollapsed, displayedGroups, props.models])

  if (props.models.length === 0) {
    return null
  }

  return (
    <div className='flex flex-col gap-4 sm:gap-5'>
      {/* Quick Vendor Selector & Expand All Bar */}
      <div className='flex flex-wrap items-center justify-between gap-2.5 rounded-xl border border-border/70 bg-card/50 p-2 sm:px-3'>
        {/* Vendor Filter Pills */}
        <div className='flex flex-wrap items-center gap-1.5 overflow-x-auto py-0.5 text-xs'>
          <button
            type='button'
            data-vendor-filter='all'
            onClick={() => setSelectedVendorFilter('all')}
            className={cn(
              'inline-flex items-center gap-1 rounded-lg px-2.5 py-1 font-mono text-xs font-medium transition-colors select-none',
              selectedVendorFilter === 'all'
                ? 'bg-foreground text-background font-semibold'
                : 'bg-muted/50 text-muted-foreground hover:bg-muted hover:text-foreground'
            )}
          >
            <span>{t('All')}</span>
            <span
              className={cn(
                'ml-0.5 text-[10px] opacity-75',
                selectedVendorFilter === 'all' ? 'text-background' : 'text-muted-foreground'
              )}
            >
              ({props.models.length})
            </span>
          </button>

          {vendorGroups.map((g) => {
            const isSelected =
              selectedVendorFilter.toLowerCase() === g.name.toLowerCase()
            return (
              <button
                key={g.name}
                type='button'
                data-vendor-filter={g.name}
                onClick={() => setSelectedVendorFilter(g.name)}
                className={cn(
                  'inline-flex items-center gap-1 rounded-lg px-2.5 py-1 font-mono text-xs font-medium transition-colors select-none',
                  isSelected
                    ? 'bg-foreground text-background font-semibold'
                    : 'bg-muted/50 text-muted-foreground hover:bg-muted hover:text-foreground'
                )}
              >
                <span>{g.name}</span>
                <span
                  className={cn(
                    'ml-0.5 text-[10px] opacity-75',
                    isSelected ? 'text-background' : 'text-muted-foreground'
                  )}
                >
                  ({g.models.length})
                </span>
              </button>
            )
          })}
        </div>

        {/* Global Expand / Collapse Control */}
        <div className='shrink-0 ml-auto'>
          <Button
            type='button'
            variant='outline'
            size='sm'
            onClick={toggleAllVendorsAndModels}
            className='h-7 gap-1.5 rounded-lg border-border/80 px-2.5 font-mono text-xs font-medium text-muted-foreground hover:text-foreground'
          >
            {allVendorsCollapsed ? (
              <>
                <ChevronsDown className='size-3.5' />
                <span>{t('Expand all')}</span>
              </>
            ) : (
              <>
                <ChevronsUp className='size-3.5' />
                <span>{t('Collapse all')}</span>
              </>
            )}
          </Button>
        </div>
      </div>

      {/* Render Vendor Sections */}
      <div className='flex flex-col gap-5'>
        {displayedGroups.map((group) => {
          const isCollapsed = collapsedVendors.has(group.name)
          return (
            <VendorGroupSection
              key={group.name}
              vendorName={group.name}
              vendorIconKey={group.iconKey}
              models={group.models}
              tokenUnit={tokenUnit}
              showRechargePrice={props.showRechargePrice}
              priceRate={props.priceRate}
              usdExchangeRate={props.usdExchangeRate}
              groupRatio={props.groupRatio}
              onModelClick={props.onModelClick}
              isCollapsed={isCollapsed}
              onToggleCollapse={() => toggleVendorCollapse(group.name)}
              expandedModelNames={expandedModelNames}
              onExpandedModelsChange={setExpandedModelNames}
            />
          )
        })}
      </div>
    </div>
  )
})
