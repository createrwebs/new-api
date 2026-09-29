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
import {
  ArrowRight,
  BookOpen,
  Image,
  MessageSquare,
  Music2,
  Sparkles,
  Video,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { useStatus } from '@/hooks/use-status'

interface HeroProps {
  className?: string
  isAuthenticated?: boolean
}

export function Hero(props: HeroProps) {
  const { t } = useTranslation()
  const { status } = useStatus()
  const docsUrl =
    (status?.docs_link as string | undefined) || 'https://docs.newapi.pro'
  const categories = [
    {
      label: t('Image generation'),
      icon: Image,
      tone: 'from-amber-500/30 via-orange-500/10 to-transparent',
      accent: 'text-amber-300',
      size: 'lg:col-span-2 lg:row-span-2',
    },
    {
      label: t('Video generation'),
      icon: Video,
      tone: 'from-indigo-500/35 via-blue-500/10 to-transparent',
      accent: 'text-indigo-300',
      size: 'lg:col-span-2 lg:row-span-2',
    },
    {
      label: t('AI chat'),
      icon: MessageSquare,
      tone: 'from-emerald-500/30 via-teal-500/10 to-transparent',
      accent: 'text-emerald-300',
      size: 'lg:col-span-2',
    },
    {
      label: t('Music generation'),
      icon: Music2,
      tone: 'from-pink-500/30 via-rose-500/10 to-transparent',
      accent: 'text-pink-300',
      size: 'lg:col-span-2',
    },
  ]

  const renderDocsButton = () => {
    const isExternal = docsUrl.startsWith('http')
    if (isExternal) {
      return (
        <Button
          variant='outline'
          className='group h-11 rounded-xl border-white/20 bg-white/5 px-5 text-sm text-white hover:bg-white/10 max-sm:w-full'
          render={
            <a href={docsUrl} target='_blank' rel='noopener noreferrer' />
          }
        >
          <BookOpen className='size-4' />
          <span>{t('Docs')}</span>
        </Button>
      )
    }
    return (
      <Button
        variant='outline'
        className='group h-11 rounded-xl border-white/20 bg-white/5 px-5 text-sm text-white hover:bg-white/10 max-sm:w-full'
        render={<Link to={docsUrl} />}
      >
        <BookOpen className='size-4' />
        <span>{t('Docs')}</span>
      </Button>
    )
  }

  return (
    <section className='relative isolate overflow-hidden bg-[#080b12] px-4 pt-32 pb-20 text-white sm:px-6 md:pt-40 md:pb-28'>
      <div
        aria-hidden='true'
        className='pointer-events-none absolute inset-0 -z-10 bg-[radial-gradient(ellipse_60%_50%_at_50%_0%,rgba(42,65,147,0.36),transparent_80%)]'
      />
      <div
        aria-hidden='true'
        className='pointer-events-none absolute inset-0 -z-10 bg-[linear-gradient(rgba(255,255,255,0.03)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,0.03)_1px,transparent_1px)] [mask-image:linear-gradient(to_bottom,black,transparent_75%)] bg-[size:5rem_5rem]'
      />

      <div className='mx-auto max-w-7xl'>
        <div className='mx-auto flex max-w-4xl flex-col items-center text-center'>
          <div className='mb-7 inline-flex items-center gap-2 rounded-full border border-white/15 bg-white/5 px-4 py-2 text-xs font-medium tracking-wide text-slate-200'>
            <Sparkles aria-hidden='true' className='size-4 text-sky-300' />
            {t('One API. Endless possibilities.')}
          </div>
          <h1 className='text-[clamp(2.6rem,6vw,5.5rem)] leading-[1.08] font-bold tracking-[-0.045em]'>
            {t('One API for every')}{' '}
            <span className='bg-gradient-to-r from-sky-300 via-blue-400 to-violet-300 bg-clip-text text-transparent'>
              {t('AI idea')}
            </span>
          </h1>
          <p className='mt-6 max-w-2xl text-base leading-relaxed text-slate-300 sm:text-lg'>
            {t(
              'Connect leading AI models through one API. Create, integrate, and manage everything in one place.'
            )}
          </p>

          <div className='mt-9 flex w-full flex-col items-stretch justify-center gap-3 sm:w-auto sm:flex-row sm:items-center'>
            {props.isAuthenticated ? (
              <Button
                className='h-11 rounded-xl bg-blue-500 px-6 text-sm font-semibold text-white hover:bg-blue-400 max-sm:w-full'
                render={<Link to='/dashboard' />}
              >
                {t('Go to Dashboard')}{' '}
                <ArrowRight aria-hidden='true' className='size-4' />
              </Button>
            ) : (
              <Button
                className='h-11 rounded-xl bg-blue-500 px-6 text-sm font-semibold text-white hover:bg-blue-400 max-sm:w-full'
                render={<Link to='/sign-up' />}
              >
                {t('Get Started')}{' '}
                <ArrowRight aria-hidden='true' className='size-4' />
              </Button>
            )}
            <Button
              variant='outline'
              className='h-11 rounded-xl border-white/20 bg-white/5 px-5 text-sm text-white hover:bg-white/10 max-sm:w-full'
              render={<Link to='/pricing' />}
            >
              {t('View Pricing')}
            </Button>
            {renderDocsButton()}
          </div>
        </div>

        <div className='mt-16 flex items-center justify-between gap-4 border-b border-white/10 pb-5 md:mt-24'>
          <div>
            <p className='text-xs font-semibold tracking-[0.2em] text-sky-300 uppercase'>
              {t('Explore what you can create')}
            </p>
            <h2 className='mt-2 text-xl font-semibold tracking-tight sm:text-2xl'>
              {t('One gateway. More ways to build.')}
            </h2>
          </div>
          <Link
            to='/pricing'
            className='hidden items-center gap-2 text-sm font-medium text-slate-300 transition-colors hover:text-white sm:inline-flex'
          >
            {t('Explore models')}{' '}
            <ArrowRight aria-hidden='true' className='size-4' />
          </Link>
        </div>

        <div className='mt-5 grid gap-4 sm:grid-cols-2 lg:grid-cols-6 lg:grid-rows-2'>
          {categories.map((category, index) => (
            <div
              key={category.label}
              className={`group relative flex min-h-56 flex-col justify-between overflow-hidden rounded-2xl border border-white/10 bg-[#111827] p-6 transition-transform duration-300 hover:-translate-y-1 hover:border-white/25 lg:min-h-64 ${category.size}`}
            >
              {/* Replace the decorative layer with an image when artwork is available. */}
              <div
                aria-hidden='true'
                className={`absolute inset-0 bg-gradient-to-br ${category.tone}`}
              />
              <div
                aria-hidden='true'
                className='absolute -right-8 -bottom-12 size-48 rounded-full border border-white/15 shadow-[0_0_0_28px_rgba(255,255,255,0.025),0_0_0_56px_rgba(255,255,255,0.02)] transition-transform duration-500 group-hover:scale-110'
              />
              <div className='relative flex items-center justify-between'>
                <span
                  className={`flex size-11 items-center justify-center rounded-xl border border-white/15 bg-white/10 ${category.accent}`}
                >
                  <category.icon aria-hidden='true' className='size-5' />
                </span>
                <span className='text-xs font-medium text-slate-400'>
                  0{index + 1} / 04
                </span>
              </div>
              <div className='relative'>
                <p className='text-xs font-medium text-slate-300'>
                  {t('Powered by a unified API')}
                </p>
                <h3 className='mt-2 text-2xl font-semibold tracking-tight'>
                  {category.label}
                </h3>
              </div>
            </div>
          ))}
        </div>
        <div className='mt-6 flex flex-wrap items-center justify-center gap-x-6 gap-y-3 text-xs font-medium text-slate-400'>
          <span>API</span>
          <span>OpenAI</span>
          <span>Claude</span>
          <span>Gemini</span>
          <span>DeepSeek</span>
        </div>
        <p className='mt-5 text-center text-xs text-slate-400'>
          {t(
            'Supports one-click configuration and perfectly adapts to NewAPI multi-protocol configuration.'
          )}
        </p>
      </div>
    </section>
  )
}
