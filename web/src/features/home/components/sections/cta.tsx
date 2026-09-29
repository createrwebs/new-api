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
import { Link } from '@tanstack/react-router'
import { ArrowRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { AnimateInView } from '@/components/animate-in-view'
import { Button } from '@/components/ui/button'

interface CTAProps {
  className?: string
  isAuthenticated?: boolean
}

export function CTA(props: CTAProps) {
  const { t } = useTranslation()

  if (props.isAuthenticated) {
    return null
  }

  return (
    <section className='relative z-10 overflow-hidden bg-[#080b12] px-4 py-24 text-white sm:px-6 md:py-32'>
      {/* Gradient mesh background */}
      <div
        aria-hidden
        className='absolute inset-0 opacity-20'
        style={{
          background: [
            'radial-gradient(ellipse 50% 50% at 30% 50%, oklch(0.7 0.15 250 / 70%) 0%, transparent 70%)',
            'radial-gradient(ellipse 40% 40% at 70% 40%, oklch(0.65 0.12 200 / 50%) 0%, transparent 70%)',
          ].join(', '),
        }}
      />

      <AnimateInView
        className='relative mx-auto max-w-2xl text-center'
        animation='scale-in'
      >
        <h2 className='text-2xl leading-tight font-bold tracking-tight md:text-4xl'>
          {t('Ready to simplify')}
          <br />
          <span className='bg-gradient-to-r from-sky-300 via-blue-400 to-violet-300 bg-clip-text text-transparent'>
            {t('your AI integration?')}
          </span>
        </h2>
        <p className='mx-auto mt-5 max-w-md text-sm leading-relaxed text-slate-300 md:text-base'>
          {t(
            'Deploy your own gateway and start routing requests through your configured upstream services.'
          )}
        </p>
        <div className='mt-8 flex flex-col items-stretch justify-center gap-3 sm:flex-row sm:items-center'>
          <Button
            className='group rounded-xl bg-blue-500 text-white hover:bg-blue-400'
            render={<Link to='/sign-up' />}
          >
            {t('Get Started')}
            <ArrowRight className='ml-1 size-3.5 transition-transform duration-200 group-hover:translate-x-0.5' />
          </Button>
          <Button
            variant='outline'
            className='rounded-xl border-white/20 bg-white/5 text-white hover:bg-white/10'
            render={<Link to='/pricing' />}
          >
            {t('View Pricing')}
          </Button>
        </div>
      </AnimateInView>
    </section>
  )
}
