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
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import {
  CardStaggerContainer,
  CardStaggerItem,
} from '@/components/page-transition'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { useStatus } from '@/hooks/use-status'

import { ContactChannelsCard } from './components/contact-channels-card'

/**
 * `/tickets` — operator contact entry point.
 *
 * No ticket backend exists yet, so instead of a non-functional form this page
 * surfaces the support channels the deployment actually advertises and keeps
 * the ticket slot obvious for when a ticketing system lands.
 */
export function SupportTickets() {
  const { t } = useTranslation()
  const { status } = useStatus()

  const docsLink = (status?.docs_link as string | undefined) ?? ''
  const contactEmail =
    (status?.contact as string | undefined) ||
    (status?.email as string | undefined) ||
    ''

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Support Tickets')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='mx-auto flex w-full max-w-5xl flex-col gap-4 sm:gap-5'>
          <CardStaggerContainer className='flex flex-col gap-4 sm:gap-5'>
            <CardStaggerItem>
              <ContactChannelsCard
                contactEmail={contactEmail}
                docsLink={docsLink}
              />
            </CardStaggerItem>

            <CardStaggerItem>
              <Card
                data-card-hover='false'
                className='gap-0 overflow-hidden py-0'
              >
                <CardHeader className='border-b p-3 !pb-3 sm:p-5 sm:!pb-5'>
                  <CardTitle className='text-base tracking-tight'>
                    {t('Your tickets')}
                  </CardTitle>
                  <CardDescription className='text-xs'>
                    {t('Submitted tickets and their status appear here.')}
                  </CardDescription>
                </CardHeader>
                <CardContent className='p-3 sm:p-5'>
                  <div className='bg-background/60 flex flex-col items-center gap-1 rounded-xl border p-6 text-center'>
                    <p className='text-sm font-medium'>
                      {t('No support tickets yet')}
                    </p>
                    <p className='text-muted-foreground text-xs'>
                      {t(
                        'Ticketing is not enabled on this deployment. Use the contact channels above to reach the operator.'
                      )}
                    </p>
                  </div>
                </CardContent>
              </Card>
            </CardStaggerItem>
          </CardStaggerContainer>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}