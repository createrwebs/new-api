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
import { BookOpen, ExternalLink, LifeBuoy, Mail, type LucideIcon } from 'lucide-react'
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

interface ContactChannelsCardProps {
  contactEmail: string
  docsLink: string
}

/**
 * Support channels the deployment advertises through `/api/status`.
 *
 * Renders nothing actionable when the operator configured neither an email nor
 * a docs link, rather than showing dead buttons.
 */
export function ContactChannelsCard({
  contactEmail,
  docsLink,
}: ContactChannelsCardProps) {
  const { t } = useTranslation()

  const channels: {
    key: string
    icon: LucideIcon
    label: string
    href: string
    action: string
    external: boolean
  }[] = [
    ...(contactEmail
      ? [
          {
            key: 'email',
            icon: Mail,
            label: contactEmail,
            href: `mailto:${contactEmail}`,
            action: t('Email'),
            external: false,
          },
        ]
      : []),
    ...(docsLink
      ? [
          {
            key: 'docs',
            icon: BookOpen,
            label: t('Documentation'),
            href: docsLink,
            action: t('Open'),
            external: true,
          },
        ]
      : []),
  ]

  return (
    <Card data-card-hover='false' className='gap-0 overflow-hidden py-0'>
      <CardHeader className='border-b p-3 !pb-3 sm:p-5 sm:!pb-5'>
        <div className='flex items-center gap-3'>
          <IconBadge tone='info' size='title'>
            <LifeBuoy />
          </IconBadge>
          <div className='min-w-0'>
            <CardTitle className='text-base tracking-tight sm:text-lg'>
              {t('Contact support')}
            </CardTitle>
            <CardDescription className='text-xs'>
              {t(
                'Reach the operator of this deployment when something is not working as expected.'
              )}
            </CardDescription>
          </div>
        </div>
      </CardHeader>
      <CardContent className='p-3 sm:p-5'>
        {channels.length === 0 ? (
          <p className='text-muted-foreground text-sm'>
            {t(
              'This deployment has not configured a support contact yet. Ask the operator directly.'
            )}
          </p>
        ) : (
          <div className='grid gap-3 sm:grid-cols-2'>
            {channels.map((channel) => (
              <div
                key={channel.key}
                className='bg-background/60 flex items-center justify-between gap-3 rounded-xl border p-3'
              >
                <span className='flex min-w-0 items-center gap-2'>
                  <channel.icon
                    className='size-4 shrink-0'
                    aria-hidden='true'
                  />
                  <span className='truncate text-sm font-medium'>
                    {channel.label}
                  </span>
                </span>
                <Button
                  variant='outline'
                  size='sm'
                  render={
                    <a
                      href={channel.href}
                      target={channel.external ? '_blank' : undefined}
                      rel={channel.external ? 'noreferrer noopener' : 'noreferrer'}
                    />
                  }
                >
                  {channel.action}
                  <ExternalLink data-icon='inline-end' />
                </Button>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}