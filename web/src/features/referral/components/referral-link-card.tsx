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
import { UserPlus } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Card, CardContent } from '@/components/ui/card'
import { IconBadge } from '@/components/ui/icon-badge'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'

interface ReferralLinkCardProps {
  affiliateLink: string
  loading: boolean
}

/**
 * Shows the user's referral link with a copy affordance.
 *
 * Shared by the Invite friends and Affiliate Partners pages, which both render
 * the same `/api/user/aff`-derived link.
 */
export function ReferralLinkCard({
  affiliateLink,
  loading,
}: ReferralLinkCardProps) {
  const { t } = useTranslation()

  return (
    <Card data-card-hover='false' className='gap-0 overflow-hidden py-0'>
      <CardContent className='flex flex-col gap-3 p-3 sm:p-5'>
        <div className='flex items-center gap-3'>
          <IconBadge tone='chart-3'>
            <UserPlus />
          </IconBadge>
          <div className='min-w-0'>
            <h3 className='truncate text-sm font-semibold sm:text-base'>
              {t('Your referral link')}
            </h3>
            <p className='text-muted-foreground text-xs'>
              {t(
                'Accounts that sign up through this link are linked to you automatically.'
              )}
            </p>
          </div>
        </div>

        <div className='flex items-center gap-2'>
          {loading ? (
            <Skeleton className='h-9 flex-1' />
          ) : (
            <Input
              value={affiliateLink}
              readOnly
              aria-label={t('Your referral link')}
              className='bg-background/70 h-9 min-w-0 flex-1 font-mono text-xs'
            />
          )}
          <CopyButton
            value={affiliateLink}
            variant='outline'
            className='bg-background size-9'
            iconClassName='size-4'
            tooltip={t('Copy referral link')}
            aria-label={t('Copy referral link')}
          />
        </div>
      </CardContent>
    </Card>
  )
}