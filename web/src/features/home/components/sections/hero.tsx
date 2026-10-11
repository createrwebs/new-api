import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import {
  ArrowRight,
  BookOpen,
  Bot,
  Copy,
  Cpu,
  Flame,
  Image as ImageIcon,
  Play,
  ShieldCheck,
  Sparkles,
  Video,
  Wand2,
  Zap,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { useStatus } from '@/hooks/use-status'

interface HeroProps {
  className?: string
  isAuthenticated?: boolean
}

type ShowcaseMode = 'image' | 'video' | 'reasoning' | 'product'

export function Hero(props: HeroProps) {
  const { t } = useTranslation()
  const { status } = useStatus()
  const [activeMode, setActiveMode] = useState<ShowcaseMode>('image')

  const docsUrl =
    (status?.docs_link as string | undefined) || 'https://docs.newapi.pro'

  const showcaseData = {
    image: {
      tag: 'FLUX.1 [dev] • 2.1s',
      prompt:
        'A sleek futuristic cybernetic crystal flower blossoming with radiant neon cyan & violet energy, 8k resolution, cinematic lighting',
      image: '/images/hero-showcase.jpg',
      badge: '✨ AI Image Synthesis',
      actionUrl: '/studio',
      actionLabel: 'เปิดสร้างภาพใน Studio',
    },
    video: {
      tag: 'Wan 2.2 Video • 1080p 24fps',
      prompt:
        'Cinematic 5-second dynamic motion reel of futuristic gold vehicle racing at neon dusk, high-speed camera orbit, photorealistic',
      image: '/images/tool-video.svg',
      badge: '🎬 Motion Video 1080p',
      actionUrl: '/studio',
      actionLabel: 'ลองสร้างวิดีโอใน Studio',
    },
    reasoning: {
      tag: 'Claude 3.7 & DeepSeek R1',
      prompt:
        'Analyze full-stack e-commerce logistics, synthesize Thai VAT tax calculation rules, and generate type-safe React code',
      image: '/images/model-claude.svg',
      badge: '🧠 Deep Reasoning & Code',
      actionUrl: '/playground',
      actionLabel: 'เปิดทดสอบใน Playground',
    },
    product: {
      tag: 'Product Studio • Luxury Preset',
      prompt:
        'Amber crystal perfume bottle on black obsidian stone pedestal, water ripples, commercial lighting --template Luxury Black',
      image: '/images/tool-product.jpg',
      badge: '🛍️ Commercial e-Commerce',
      actionUrl: '/studio',
      actionLabel: 'จัดฉากสินค้าใน Studio',
    },
  }

  const currentShowcase = showcaseData[activeMode]

  const renderDocsButton = () => {
    const isExternal = docsUrl.startsWith('http')
    if (isExternal) {
      return (
        <Button
          variant='outline'
          className='h-11 rounded-xl border-white/20 bg-white/5 px-5 text-sm text-white hover:bg-white/10 max-sm:w-full'
          render={
            <a href={docsUrl} target='_blank' rel='noopener noreferrer' />
          }
        >
          <BookOpen className='size-4 mr-1.5' />
          <span>{t('API Docs')}</span>
        </Button>
      )
    }
    return (
      <Button
        variant='outline'
        className='h-11 rounded-xl border-white/20 bg-white/5 px-5 text-sm text-white hover:bg-white/10 max-sm:w-full'
        render={<Link to={docsUrl} />}
      >
        <BookOpen className='size-4 mr-1.5' />
        <span>{t('API Docs')}</span>
      </Button>
    )
  }

  return (
    <section className='relative isolate overflow-hidden bg-[#06080f] px-4 pt-32 pb-20 text-white sm:px-6 md:pt-40 md:pb-28'>
      {/* Radiant Aura Ambient Gradients */}
      <div
        aria-hidden='true'
        className='pointer-events-none absolute inset-0 -z-10 bg-[radial-gradient(ellipse_70%_50%_at_50%_0%,rgba(37,99,235,0.28),transparent_80%)]'
      />
      <div
        aria-hidden='true'
        className='pointer-events-none absolute inset-0 -z-10 bg-[linear-gradient(rgba(255,255,255,0.03)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,0.03)_1px,transparent_1px)] [mask-image:linear-gradient(to_bottom,black,transparent_75%)] bg-[size:4rem_4rem]'
      />

      <div className='mx-auto max-w-7xl'>
        {/* Main Hero Header */}
        <div className='mx-auto flex max-w-4xl flex-col items-center text-center'>
          <div className='mb-7 inline-flex items-center gap-2 rounded-full border border-sky-400/30 bg-sky-500/10 px-4 py-1.5 text-xs font-semibold tracking-wide text-sky-300 backdrop-blur-md'>
            <span className='flex size-2 rounded-full bg-emerald-400 animate-pulse' />
            <Sparkles aria-hidden='true' className='size-3.5 text-sky-300' />
            {t('Tora AI 2.0 — All-in-One Gateway & Creative Studio')}
          </div>

          <h1 className='text-[clamp(2.5rem,5.8vw,5.2rem)] leading-[1.08] font-extrabold tracking-[-0.04em] text-white'>
            {t('เชื่อมต่อทุกโมเดล AI ระดับโลก')}{' '}
            <span className='bg-gradient-to-r from-sky-300 via-blue-400 to-violet-300 bg-clip-text text-transparent'>
              {t('ใน API และกระเป๋าเดียว')}
            </span>
          </h1>

          <p className='mt-6 max-w-3xl text-base leading-relaxed text-slate-300 sm:text-lg'>
            {t(
              'เกตเวย์ AI รวมกว่า 240+ โมเดลชั้นนำ (OpenAI, Claude, Gemini, DeepSeek, Flux, Kling) พร้อม Creative Studio ลบพื้นหลังสินค้า แต่งรูปโฆษณา และเสกวิดีโอ 1080p อัตโนมัติ จบในกระเป๋าเดียว'
            )}
          </p>

          {/* Action CTAs */}
          <div className='mt-9 flex w-full flex-col items-stretch justify-center gap-3 sm:w-auto sm:flex-row sm:items-center'>
            {props.isAuthenticated ? (
              <Button
                className='h-11 rounded-xl bg-blue-600 px-6 text-sm font-semibold text-white shadow-lg shadow-blue-500/30 hover:bg-blue-500 max-sm:w-full'
                render={<Link to='/dashboard' />}
              >
                <span>{t('เข้าสู่แดชบอร์ด (Dashboard)')}</span>
                <ArrowRight aria-hidden='true' className='size-4 ml-1.5' />
              </Button>
            ) : (
              <Button
                className='h-11 rounded-xl bg-blue-600 px-6 text-sm font-semibold text-white shadow-lg shadow-blue-500/30 hover:bg-blue-500 max-sm:w-full'
                render={<Link to='/sign-up' />}
              >
                <span>{t('เริ่มต้นใช้งานฟรี (Get Started)')}</span>
                <ArrowRight aria-hidden='true' className='size-4 ml-1.5' />
              </Button>
            )}

            <Button
              className='h-11 rounded-xl border border-sky-400/30 bg-sky-500/10 px-5 text-sm font-semibold text-sky-200 hover:bg-sky-500/20 max-sm:w-full'
              render={<Link to='/studio' />}
            >
              <Wand2 className='size-4 mr-1.5 text-sky-300' />
              <span>{t('เปิด Creative Studio')}</span>
            </Button>

            <Button
              variant='outline'
              className='h-11 rounded-xl border-white/20 bg-white/5 px-5 text-sm text-white hover:bg-white/10 max-sm:w-full'
              render={<Link to='/pricing' />}
            >
              <span>{t('ดูอัตราค่าบริการ')}</span>
            </Button>

            {renderDocsButton()}
          </div>
        </div>

        {/* ============================================================ */}
        {/* FLAQ-STYLE HERO SHOWCASE BANNER WITH ARTWORK & INTERACTION   */}
        {/* ============================================================ */}
        <div className='mt-16 sm:mt-20'>
          {/* Mode Switcher Tabs */}
          <div className='mx-auto flex max-w-xl flex-wrap items-center justify-center gap-2 rounded-2xl border border-white/10 bg-white/5 p-1.5 backdrop-blur-xl'>
            {[
              { id: 'image', label: t('🎨 AI Image (FLUX)'), icon: ImageIcon },
              { id: 'video', label: t('🎬 1080p Video'), icon: Video },
              { id: 'product', label: t('🛍️ Product Studio'), icon: Sparkles },
              { id: 'reasoning', label: t('🧠 Claude & R1'), icon: Bot },
            ].map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveMode(tab.id as ShowcaseMode)}
                className={`rounded-xl px-4 py-2 text-xs font-semibold transition-all ${
                  activeMode === tab.id
                    ? 'bg-blue-600 text-white shadow-md shadow-blue-500/30'
                    : 'text-slate-400 hover:bg-white/5 hover:text-white'
                }`}
              >
                {tab.label}
              </button>
            ))}
          </div>

          {/* Hero Showcase Display Frame */}
          <div className='relative mt-6 overflow-hidden rounded-3xl border border-white/15 bg-[#0a0e1a]/90 shadow-2xl shadow-blue-500/10 backdrop-blur-2xl'>
            {/* Top Prompt HUD Bar */}
            <div className='flex flex-col gap-3 border-b border-white/10 bg-white/5 px-6 py-4 sm:flex-row sm:items-center sm:justify-between'>
              <div className='flex items-center gap-3 overflow-hidden'>
                <div className='flex size-8 shrink-0 items-center justify-center rounded-lg bg-sky-500/20 text-sky-400'>
                  <Wand2 className='size-4' />
                </div>
                <div className='overflow-hidden text-left'>
                  <span className='block text-[10px] font-semibold text-slate-400 uppercase tracking-wider'>
                    Prompt Input
                  </span>
                  <p className='truncate text-xs font-mono text-slate-200'>
                    "{currentShowcase.prompt}"
                  </p>
                </div>
              </div>

              <div className='flex items-center gap-2 shrink-0'>
                <span className='rounded-full border border-white/15 bg-black/40 px-3 py-1 text-xs font-mono text-emerald-400'>
                  {currentShowcase.tag}
                </span>
                <Button
                  size='sm'
                  className='h-8 rounded-lg bg-blue-600 px-3 text-xs font-semibold text-white hover:bg-blue-500'
                  render={<Link to={currentShowcase.actionUrl} />}
                >
                  <span>{currentShowcase.actionLabel}</span>
                  <ArrowRight className='size-3 ml-1' />
                </Button>
              </div>
            </div>

            {/* Showcase Visual Art Container */}
            <div className='relative aspect-[16/9] w-full overflow-hidden bg-black/80 md:aspect-[21/9]'>
              <img
                src={currentShowcase.image}
                alt='Tora AI Hero Showcase'
                loading='eager'
                className='size-full object-cover transition-transform duration-700 hover:scale-101'
              />
              <div
                aria-hidden='true'
                className='pointer-events-none absolute inset-0 bg-gradient-to-t from-[#06080f] via-transparent to-transparent opacity-80'
              />

              {/* Floating Badge on Visual */}
              <div className='absolute bottom-6 left-6 flex flex-wrap items-center gap-2.5'>
                <span className='rounded-full border border-white/20 bg-black/60 px-3.5 py-1.5 text-xs font-bold text-white backdrop-blur-md'>
                  {currentShowcase.badge}
                </span>
                <span className='rounded-full border border-sky-400/30 bg-sky-500/20 px-3 py-1.5 text-xs font-semibold text-sky-300 backdrop-blur-md'>
                  ⚡ Single Tora Wallet
                </span>
              </div>
            </div>

            {/* Live Metrics Floating Bar at Bottom */}
            <div className='grid grid-cols-2 gap-4 border-t border-white/10 bg-white/5 p-4 sm:grid-cols-4 sm:p-5 text-center'>
              <div className='flex items-center justify-center gap-3'>
                <Cpu className='size-5 text-sky-400' />
                <div className='text-left'>
                  <span className='block text-sm font-bold text-white'>
                    240+ Models
                  </span>
                  <span className='block text-[10px] text-slate-400'>
                    OpenAI, Claude, Gemini, Flux
                  </span>
                </div>
              </div>

              <div className='flex items-center justify-center gap-3'>
                <ShieldCheck className='size-5 text-emerald-400' />
                <div className='text-left'>
                  <span className='block text-sm font-bold text-white'>
                    99.9% Uptime
                  </span>
                  <span className='block text-[10px] text-slate-400'>
                    Auto Multi-Provider Fallback
                  </span>
                </div>
              </div>

              <div className='flex items-center justify-center gap-3'>
                <Zap className='size-5 text-amber-400' />
                <div className='text-left'>
                  <span className='block text-sm font-bold text-white'>
                    &lt; 45ms Edge
                  </span>
                  <span className='block text-[10px] text-slate-400'>
                    Regional Fast Routing
                  </span>
                </div>
              </div>

              <div className='flex items-center justify-center gap-3'>
                <Sparkles className='size-5 text-fuchsia-400' />
                <div className='text-left'>
                  <span className='block text-sm font-bold text-white'>
                    PromptPay &amp; Cards
                  </span>
                  <span className='block text-[10px] text-slate-400'>
                    Instant Thai Top-up
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
