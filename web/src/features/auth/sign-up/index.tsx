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
import { CheckCircle2, Clock, KeyRound } from 'lucide-react'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import { useStatus } from '@/hooks/use-status'

import { AuthLayout } from '../auth-layout'
import { TermsFooter } from '../components/terms-footer'
import { SignUpForm } from './components/sign-up-form'

const HIGHLIGHTS = [
  { Icon: KeyRound, label: 'One key for chat, images, and video' },
  { Icon: CheckCircle2, label: 'OpenAI and Anthropic compatible' },
  { Icon: Clock, label: 'Ready to call models in minutes' },
] as const

export function SignUp() {
  const { t } = useTranslation()
  const { status } = useStatus()

  return (
    <AuthLayout>
      <Card className='gap-0 overflow-hidden py-0 shadow-sm'>
        <CardHeader className='gap-2 pb-4'>
          <CardTitle className='text-2xl font-semibold tracking-tight'>
            {t('Create an account')}
          </CardTitle>
          <CardDescription className='text-sm'>
            {t('Sign up once and call every text, image, and video model.')}
          </CardDescription>
        </CardHeader>

        <CardContent className='space-y-5 pb-6'>
          <ul className='text-muted-foreground flex flex-wrap gap-2 text-xs'>
            {HIGHLIGHTS.map(({ Icon, label }) => (
              <li key={label}>
                <Badge
                  variant='secondary'
                  className='gap-1.5 rounded-full py-1 font-normal'
                >
                  <Icon aria-hidden='true' className='size-3.5' />
                  {t(label)}
                </Badge>
              </li>
            ))}
          </ul>

          <SignUpForm />

          <div className='flex items-center gap-3'>
            <Separator className='flex-1' />
            <span className='text-muted-foreground text-xs'>
              {t('Already have an account?')}
            </span>
            <Separator className='flex-1' />
          </div>

          <Link
            to='/sign-in'
            className='text-primary flex items-center justify-center text-sm font-medium underline underline-offset-4'
          >
            {t('Sign in')}
          </Link>
        </CardContent>

        <div className='bg-muted/40 border-t px-6 py-4'>
          <TermsFooter variant='sign-up' status={status} className='text-xs' />
        </div>
      </Card>
    </AuthLayout>
  )
}
