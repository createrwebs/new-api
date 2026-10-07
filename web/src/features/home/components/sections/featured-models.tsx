import { Link } from '@tanstack/react-router'
import {
  ArrowRight,
  ExternalLink,
  Flame,
  MessageSquare,
  Play,
  Sparkles,
  Zap,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'

interface FeaturedModel {
  id: string
  name: string
  provider: string
  tag: string
  description: string
  image: string
  badgeColor: string
  stats: { label: string; value: string }[]
  playgroundUrl: string
  playgroundLabel: string
  detailsUrl: string
}

export function FeaturedModels() {
  const { t } = useTranslation()

  const featured: FeaturedModel[] = [
    {
      id: 'flux-dev-featured',
      name: 'FLUX.1 [dev] & [schnell]',
      provider: 'Black Forest Labs',
      tag: '✨ โมเดลสร้างภาพยอดนิยม',
      description:
        'สร้างสรรค์ผลงานภาพระดับ Masterpiece คมชัดสมจริงทุกรายละเอียด จัดการแสงเงา รายละเอียดผิว และตัวอักษรได้อย่างสมบูรณ์แบบ',
      image: '/images/model-flux.jpg',
      badgeColor: 'border-fuchsia-500/30 bg-fuchsia-500/10 text-fuchsia-300',
      stats: [
        { label: 'Resolution', value: 'Up to 2048x1152' },
        { label: 'Generation Speed', value: '2.1s (Schnell 0.8s)' },
        { label: 'Prompt Support', value: 'Thai & English Native' },
      ],
      playgroundUrl: '/studio',
      playgroundLabel: t('ทดลองสร้างภาพใน Studio'),
      detailsUrl: '/pricing',
    },
    {
      id: 'claude-3-7-featured',
      name: 'Claude 3.7 Sonnet & DeepSeek R1',
      provider: 'Anthropic / DeepSeek',
      tag: '🧠 ให้เหตุผลขั้นสูง & เขียนโค้ด',
      description:
        'สุดยอดโมเดลคิดวิเคราะห์เชิงลึกระดับแข่งขันคณิตศาสตร์และโค้ดดิ้ง สถาปัตยกรรม Hybrid Reasoning เข้าใจภาษาไทยลึกซึ้งเป็นธรรมชาติ',
      image: '/images/model-claude.svg',
      badgeColor: 'border-amber-500/30 bg-amber-500/10 text-amber-300',
      stats: [
        { label: 'Context Window', value: '200,000 Tokens' },
        { label: 'Reasoning Mode', value: 'Step-by-Step Thinking' },
        { label: 'Coding Benchmark', value: '92.4% HumanEval' },
      ],
      playgroundUrl: '/playground',
      playgroundLabel: t('เปิดทดสอบใน Playground'),
      detailsUrl: '/pricing',
    },
    {
      id: 'wan-video-featured',
      name: 'Wan 2.2 & Kling 1.5 Video',
      provider: 'Fal & Video Engines',
      tag: '🎬 เสกภาพเป็นคลิปวิดีโอ 1080p',
      description:
        'เปลี่ยนภาพนิ่งสินค้าหรือภาพวาดศิลปะให้กลายเป็นวิดีโอเคลื่อนไหว 5 วินาทีระดับภาพยนตร์ คมชัด 1080p 24fps พร้อมมุมกล้องไดนามิก',
      image: '/images/tool-video.svg',
      badgeColor: 'border-indigo-500/30 bg-indigo-500/10 text-indigo-300',
      stats: [
        { label: 'Video Quality', value: '1080p Full HD' },
        { label: 'Duration', value: '5 - 10 วินาที' },
        { label: 'Motion Control', value: 'Cinematic Camera Flow' },
      ],
      playgroundUrl: '/studio',
      playgroundLabel: t('ลองสร้าง Video ใน Studio'),
      detailsUrl: '/pricing',
    },
    {
      id: 'gemini-pro-featured',
      name: 'Gemini 2.5 Pro & Flash',
      provider: 'Google DeepMind',
      tag: '⚡ 2,000,000 Multimodal Tokens',
      description:
        'ประมวลผลเอกสาร PDF ขนาดยักษ์ วิเคราะห์คลิปวิดีโอยาว 1 ชั่วโมง และเสียงประชุมได้อย่างไร้ขีดจำกัด พร้อมโหมด Flash ความเร็วแสง',
      image: '/images/model-gemini.svg',
      badgeColor: 'border-sky-500/30 bg-sky-500/10 text-sky-300',
      stats: [
        { label: 'Context Length', value: '2,000,000 Tokens' },
        { label: 'Multimodal Input', value: 'Video, Audio, PDF, Code' },
        { label: 'Edge Latency', value: '< 250ms TTFT' },
      ],
      playgroundUrl: '/playground',
      playgroundLabel: t('เปิดทดสอบใน Playground'),
      detailsUrl: '/pricing',
    },
  ]

  return (
    <section className='relative border-t border-white/10 bg-[#070a12] px-4 py-20 sm:px-6 md:py-28'>
      {/* Background glow */}
      <div
        aria-hidden='true'
        className='pointer-events-none absolute inset-0 -z-10 bg-[radial-gradient(ellipse_60%_50%_at_50%_20%,rgba(56,189,248,0.12),transparent_70%)]'
      />

      <div className='mx-auto max-w-7xl'>
        {/* Header */}
        <div className='flex flex-col items-center text-center'>
          <div className='inline-flex items-center gap-2 rounded-full border border-amber-500/30 bg-amber-500/10 px-3.5 py-1.5 text-xs font-semibold text-amber-300'>
            <Flame className='size-3.5' />
            {t('โมเดล AI แนะนำ (Featured AI Models)')}
          </div>
          <h2 className='mt-4 text-3xl font-bold tracking-tight text-white sm:text-4xl md:text-5xl'>
            {t('โมเดลประสิทธิภาพสูงสุด พร้อมให้คุณลองทันที')}
          </h2>
          <p className='mt-4 max-w-2xl text-base text-slate-300'>
            {t(
              'สำรวจความสามารถของโมเดล AI ตัวท็อป คลิกดูรายละเอียด และกดเข้าใช้งาน Playground หรือ Creative Studio ได้ในคลิกเดียว'
            )}
          </p>
        </div>

        {/* Featured Cards Grid */}
        <div className='mt-14 grid gap-8 lg:grid-cols-2'>
          {featured.map((item) => (
            <div
              key={item.id}
              className='group flex flex-col justify-between overflow-hidden rounded-3xl border border-white/10 bg-[#0c101d] transition-all duration-300 hover:border-white/25 hover:shadow-2xl hover:shadow-blue-500/10'
            >
              {/* Image Preview Container */}
              <div className='relative aspect-[16/9] w-full overflow-hidden bg-black/60'>
                <img
                  src={item.image}
                  alt={item.name}
                  loading='lazy'
                  className='size-full object-cover transition-transform duration-700 group-hover:scale-105'
                />
                <div
                  aria-hidden='true'
                  className='pointer-events-none absolute inset-0 bg-gradient-to-t from-[#0c101d] via-[#0c101d]/20 to-transparent'
                />
                <div className='absolute top-4 left-4'>
                  <span
                    className={`inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-semibold backdrop-blur-md ${item.badgeColor}`}
                  >
                    {item.tag}
                  </span>
                </div>
                <div className='absolute top-4 right-4'>
                  <span className='rounded-full border border-white/20 bg-black/60 px-3 py-1 text-xs font-medium text-slate-300 backdrop-blur-md'>
                    {item.provider}
                  </span>
                </div>
              </div>

              {/* Card Body */}
              <div className='flex flex-1 flex-col justify-between p-6 sm:p-8'>
                <div>
                  <h3 className='text-2xl font-bold tracking-tight text-white group-hover:text-sky-300 transition-colors'>
                    {item.name}
                  </h3>
                  <p className='mt-3 text-sm leading-relaxed text-slate-300'>
                    {item.description}
                  </p>

                  {/* Highlights Bar */}
                  <div className='mt-6 grid grid-cols-3 gap-2 rounded-2xl border border-white/10 bg-white/5 p-3.5 text-xs'>
                    {item.stats.map((s) => (
                      <div key={s.label}>
                        <span className='block text-[10px] font-medium text-slate-400'>
                          {s.label}
                        </span>
                        <span className='mt-0.5 block font-semibold text-slate-100'>
                          {s.value}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>

                {/* Actions */}
                <div className='mt-8 flex flex-wrap items-center gap-3 pt-2'>
                  <Button
                    className='h-10.5 rounded-xl bg-blue-600 px-5 text-xs font-semibold text-white shadow-lg shadow-blue-500/25 hover:bg-blue-500'
                    render={<Link to={item.playgroundUrl} />}
                  >
                    <Play className='size-3.5 mr-1 fill-white' />
                    <span>{item.playgroundLabel}</span>
                  </Button>
                  <Button
                    variant='outline'
                    className='h-10.5 rounded-xl border-white/15 bg-white/5 px-4 text-xs font-medium text-slate-200 hover:bg-white/10 hover:text-white'
                    render={<Link to={item.detailsUrl} />}
                  >
                    <span>{t('ดูรายละเอียดและราคา')}</span>
                    <ExternalLink className='size-3 ml-1.5' />
                  </Button>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
