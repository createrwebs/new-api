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
import { Coins, History, Users } from 'lucide-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { Card, CardContent } from '@/components/ui/card'
import { IconBadge } from '@/components/ui/icon-badge'
import { Skeleton } from '@/components/ui/skeleton'
import { formatCompactNumber, formatQuota } from '@/lib/format'

interface ReferralRewardsCardProps {
  pendingQuota: number
  historyQuota: number
  inviteCount: number
  loading: boolean
  /** Rendered below the counters, e.g. the "Transfer to Balance" action. */
  action?: ReactNode
}

/**
 * Invite / pending-reward / lifetime-reward counters for the referral program.
 */
export function ReferralRewardsCard({
  pendingQuota,
  historyQuota,
  inviteCount,
  loading,
  action,
}: ReferralRewardsCardProps) {
  const { t } = useTranslation()

  const stats = [
    {
      label: t('Invites'),
      value: formatCompactNumber(inviteCount),
      icon: Users,
    },
    {
      label: t('Pending'),
      value: formatQuota(pendingQuota),
      icon: Coins,
    },
    {
      label: t('Total Earned'),
      value: formatQuota(historyQuota),
      icon: History,
    },
  ]

  return (
    <Card data-card-hover='false' className='gap-0 overflow-hidden py-0'>
      <CardContent className='p-0'>
        <div className='divide-border/60 grid grid-cols-1 divide-y sm:grid-cols-3 sm:divide-x sm:divide-y-0'>
          {stats.map((stat) => (
            <div key={stat.label} className='px-3 py-3.5 sm:px-5 sm:py-4'>
              <div className='flex items-center gap-2'>
                <IconBadge tone='chart-3' size='stat'>
                  <stat.icon />
                </IconBadge>
                <div className='text-muted-foreground truncate text-xs font-medium tracking-wider uppercase'>
                  {stat.label}
                </div>
              </div>
              {loading ? (
                <Skeleton className='mt-2 h-7 w-24' />
              ) : (
                <div className='text-foreground mt-1.5 truncate font-mono text-lg font-bold tracking-tight tabular-nums sm:mt-2 sm:text-2xl'>
                  {stat.value}
                </div>
              )}
            </div>
          ))}
        </div>
        {action != null && (
          <div className='border-t p-3 sm:p-4'>{action}</div>
        )}
      </CardContent>
    </Card>
  )
}