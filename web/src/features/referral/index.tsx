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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import {
  CardStaggerContainer,
  CardStaggerItem,
} from '@/components/page-transition'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { TransferDialog } from '@/features/wallet/components/dialogs/transfer-dialog'
import { useAffiliate } from '@/features/wallet/hooks/use-affiliate'
import { useTopupInfo } from '@/features/wallet/hooks/use-topup-info'
import { useAuthStore } from '@/stores/auth-store'

import { ReferralLinkCard } from './components/referral-link-card'
import { ReferralRewardsCard } from './components/referral-rewards-card'

const HOW_IT_WORKS_STEPS = [
  {
    title: 'Share your link',
    description: 'Send your referral link to friends, teammates, or a community.',
  },
  {
    title: 'They sign up',
    description: 'New accounts created through your link are linked to you.',
  },
  {
    title: 'Collect rewards',
    description: 'Rewards accumulate and can be moved to your main balance.',
  },
]

/**
 * `/invite` — the sharing-focused entry point for the referral program.
 *
 * Shows the link plus a short explainer; the money side lives on
 * {@link AffiliatePartners}.
 */
export function InviteFriends() {
  const { t } = useTranslation()
  const { affiliateLink, loading } = useAffiliate()
  const user = useAuthStore((state) => state.auth.user)

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Invite friends')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='mx-auto flex w-full max-w-5xl flex-col gap-4 sm:gap-5'>
          <CardStaggerContainer className='flex flex-col gap-4 sm:gap-5'>
            <CardStaggerItem>
              <ReferralLinkCard
                affiliateLink={affiliateLink}
                loading={loading}
              />
            </CardStaggerItem>

            <CardStaggerItem>
              <ReferralRewardsCard
                pendingQuota={Number(user?.aff_quota ?? 0)}
                historyQuota={Number(user?.aff_history_quota ?? 0)}
                inviteCount={Number(user?.aff_count ?? 0)}
                loading={loading}
              />
            </CardStaggerItem>

            <CardStaggerItem>
              <Card
                data-card-hover='false'
                className='gap-0 overflow-hidden py-0'
              >
                <CardHeader className='border-b p-3 !pb-3 sm:p-5 sm:!pb-5'>
                  <CardTitle className='text-base tracking-tight'>
                    {t('How it works')}
                  </CardTitle>
                  <CardDescription className='text-xs'>
                    {t('Three steps to start earning referral rewards.')}
                  </CardDescription>
                </CardHeader>
                <CardContent className='p-3 sm:p-5'>
                  <ol className='grid gap-3 sm:grid-cols-3'>
                    {HOW_IT_WORKS_STEPS.map((step, index) => (
                      <li
                        key={step.title}
                        className='bg-background/60 flex flex-col gap-1 rounded-xl border p-3'
                      >
                        <span className='text-muted-foreground font-mono text-xs'>
                          {index + 1}
                        </span>
                        <span className='text-sm font-medium'>{step.title}</span>
                        <span className='text-muted-foreground text-xs leading-relaxed'>
                          {step.description}
                        </span>
                      </li>
                    ))}
                  </ol>
                </CardContent>
              </Card>
            </CardStaggerItem>
          </CardStaggerContainer>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

/**
 * `/affiliate` — the earnings side of the referral program.
 *
 * Reuses the wallet transfer dialog so rewards land on the main balance using
 * the existing `/api/user/aff_transfer` flow.
 */
export function AffiliatePartners() {
  const { t } = useTranslation()
  const { affiliateLink, loading, transferQuota, transferring } = useAffiliate()
  const { topupInfo } = useTopupInfo()
  const user = useAuthStore((state) => state.auth.user)
  const [transferDialogOpen, setTransferDialogOpen] = useState(false)

  const pendingQuota = Number(user?.aff_quota ?? 0)
  const complianceConfirmed =
    topupInfo?.payment_compliance_confirmed !== false

  return (
    <>
      <SectionPageLayout>
        <SectionPageLayout.Title>
          {t('Affiliate Partners')}
        </SectionPageLayout.Title>
        <SectionPageLayout.Content>
          <div className='mx-auto flex w-full max-w-5xl flex-col gap-4 sm:gap-5'>
            <CardStaggerContainer className='flex flex-col gap-4 sm:gap-5'>
              <CardStaggerItem>
                <ReferralRewardsCard
                  pendingQuota={pendingQuota}
                  historyQuota={Number(user?.aff_history_quota ?? 0)}
                  inviteCount={Number(user?.aff_count ?? 0)}
                  loading={loading}
                  action={
                    pendingQuota > 0 ? (
                      <div className='flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between'>
                        <p className='text-muted-foreground text-xs'>
                          {complianceConfirmed
                            ? t(
                                'Move accumulated rewards to your main balance any time.'
                              )
                            : t(
                                'Referral reward transfer is disabled until the administrator confirms compliance terms.'
                              )}
                        </p>
                        <Button
                          size='sm'
                          disabled={!complianceConfirmed}
                          onClick={() => setTransferDialogOpen(true)}
                        >
                          {t('Transfer to Balance')}
                        </Button>
                      </div>
                    ) : null
                  }
                />
              </CardStaggerItem>

              <CardStaggerItem>
                <ReferralLinkCard
                  affiliateLink={affiliateLink}
                  loading={loading}
                />
              </CardStaggerItem>
            </CardStaggerContainer>
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <TransferDialog
        open={transferDialogOpen}
        onOpenChange={setTransferDialogOpen}
        onConfirm={transferQuota}
        availableQuota={pendingQuota}
        transferring={transferring}
      />
    </>
  )
}