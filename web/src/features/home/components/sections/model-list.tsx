import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import { ArrowRight, Bot, Cpu, Image, Sparkles, Video, Zap } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'

type ModelCategory = 'all' | 'chat' | 'image' | 'video' | 'coding'

interface ModelItem {
  id: string
  name: string
  provider: string
  category: ModelCategory
  badge?: string
  context: string
  speed: string
  price: string
  tone: string
}

export function ModelList() {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState<ModelCategory>('all')

  const models: ModelItem[] = [
    {
      id: 'gpt-4o',
      name: 'GPT-4o',
      provider: 'OpenAI',
      category: 'chat',
      badge: 'Flagship Multimodal',
      context: '128K',
      speed: '95 tps',
      price: '1.25 Credits / 1k',
      tone: 'from-emerald-500/20 to-teal-500/5 text-emerald-400',
    },
    {
      id: 'claude-3-7-sonnet',
      name: 'Claude 3.7 Sonnet',
      provider: 'Anthropic',
      category: 'coding',
      badge: 'SOTA Reasoning',
      context: '200K',
      speed: '82 tps',
      price: '1.50 Credits / 1k',
      tone: 'from-orange-500/20 to-amber-500/5 text-orange-400',
    },
    {
      id: 'flux-dev',
      name: 'FLUX.1 [dev]',
      provider: 'Black Forest Labs',
      category: 'image',
      badge: 'Photorealistic Art',
      context: '1024x1024',
      speed: '2.1s',
      price: '12.5 Credits / img',
      tone: 'from-fuchsia-500/20 to-pink-500/5 text-fuchsia-400',
    },
    {
      id: 'deepseek-r1',
      name: 'DeepSeek R1',
      provider: 'DeepSeek',
      category: 'chat',
      badge: 'Deep Reasoning',
      context: '128K',
      speed: '65 tps',
      price: '0.28 Credits / 1k',
      tone: 'from-blue-500/20 to-indigo-500/5 text-blue-400',
    },
    {
      id: 'gemini-2-5-pro',
      name: 'Gemini 2.5 Pro',
      provider: 'Google',
      category: 'chat',
      badge: '2M Context Monster',
      context: '2,000K',
      speed: '110 tps',
      price: '0.80 Credits / 1k',
      tone: 'from-sky-500/20 to-cyan-500/5 text-sky-400',
    },
    {
      id: 'wan-2-2-video',
      name: 'Wan 2.2 / Kling Video',
      provider: 'Fal / Direct',
      category: 'video',
      badge: '1080p Cinema Motion',
      context: '5-10s',
      speed: '18s',
      price: '125 Credits / vid',
      tone: 'from-amber-500/20 to-yellow-500/5 text-amber-400',
    },
    {
      id: 'claude-3-5-haiku',
      name: 'Claude 3.5 Haiku',
      provider: 'Anthropic',
      category: 'chat',
      badge: 'Lightning Fast',
      context: '200K',
      speed: '140 tps',
      price: '0.40 Credits / 1k',
      tone: 'from-orange-500/20 to-amber-500/5 text-orange-300',
    },
    {
      id: 'o3-mini',
      name: 'o3-mini',
      provider: 'OpenAI',
      category: 'coding',
      badge: 'Math & STEM Master',
      context: '128K',
      speed: '78 tps',
      price: '0.55 Credits / 1k',
      tone: 'from-emerald-500/20 to-teal-500/5 text-emerald-300',
    },
    {
      id: 'flux-schnell',
      name: 'FLUX.1 [schnell]',
      provider: 'Black Forest Labs',
      category: 'image',
      badge: 'Instant 4-Step Gen',
      context: '1024x1024',
      speed: '0.8s',
      price: '8.0 Credits / img',
      tone: 'from-purple-500/20 to-pink-500/5 text-purple-300',
    },
  ]

  const filteredModels =
    activeTab === 'all'
      ? models
      : models.filter((m) => m.category === activeTab)

  return (
    <section className='relative border-t border-white/10 bg-[#090d16] px-4 py-20 sm:px-6 md:py-28'>
      <div className='mx-auto max-w-7xl'>
        {/* Header */}
        <div className='flex flex-col items-center text-center'>
          <div className='inline-flex items-center gap-2 rounded-full border border-sky-500/30 bg-sky-500/10 px-3.5 py-1.5 text-xs font-semibold text-sky-300'>
            <Cpu className='size-3.5' />
            {t('Universal AI Model Directory')}
          </div>
          <h2 className='mt-4 text-3xl font-bold tracking-tight text-white sm:text-4xl md:text-5xl'>
            {t('โมเดล AI ชั้นนำระดับโลก ทั้งหมดในที่เดียว')}
          </h2>
          <p className='mt-4 max-w-2xl text-base text-slate-300'>
            {t(
              'เชื่อมต่อโมเดลปัญญาประดิษฐ์ที่ดีที่สุดกว่า 240+ โมเดล ทั้ง Text, Image, Video และ Code ผ่าน OpenAI-compatible API เดียว'
            )}
          </p>

          {/* Filter Tabs */}
          <div className='mt-8 flex flex-wrap items-center justify-center gap-2 rounded-2xl border border-white/10 bg-white/5 p-1.5'>
            {[
              { id: 'all', label: t('ทั้งหมด (All)'), icon: Sparkles },
              { id: 'chat', label: t('แชท & เหตุผล (LLM)'), icon: Bot },
              { id: 'image', label: t('สร้างรูปภาพ (Image)'), icon: Image },
              { id: 'video', label: t('วิดีโอ (Video)'), icon: Video },
              { id: 'coding', label: t('โค้ดดิ้ง (Coding)'), icon: Zap },
            ].map((tab) => {
              const Icon = tab.icon
              const isActive = activeTab === tab.id
              return (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id as ModelCategory)}
                  className={`inline-flex items-center gap-2 rounded-xl px-4 py-2 text-xs font-medium transition-all ${
                    isActive
                      ? 'bg-blue-600 text-white shadow-lg shadow-blue-500/25'
                      : 'text-slate-400 hover:bg-white/5 hover:text-white'
                  }`}
                >
                  <Icon className='size-3.5' />
                  <span>{tab.label}</span>
                </button>
              )
            })}
          </div>
        </div>

        {/* Model Grid */}
        <div className='mt-12 grid gap-4 sm:grid-cols-2 lg:grid-cols-3'>
          {filteredModels.map((model) => (
            <div
              key={model.id}
              className='group relative flex flex-col justify-between overflow-hidden rounded-2xl border border-white/10 bg-[#0f1422] p-5.5 transition-all duration-300 hover:-translate-y-1 hover:border-white/20 hover:shadow-xl hover:shadow-blue-500/5'
            >
              <div
                aria-hidden='true'
                className={`pointer-events-none absolute -right-12 -top-12 size-36 rounded-full bg-gradient-to-br ${model.tone} opacity-30 blur-2xl transition-opacity group-hover:opacity-60`}
              />

              <div>
                <div className='flex items-center justify-between'>
                  <span className='text-xs font-semibold tracking-wider text-slate-400 uppercase'>
                    {model.provider}
                  </span>
                  {model.badge && (
                    <Badge
                      variant='outline'
                      className='border-white/15 bg-white/5 text-[10px] text-sky-300'
                    >
                      {model.badge}
                    </Badge>
                  )}
                </div>

                <h3 className='mt-3 text-lg font-bold text-white group-hover:text-sky-300 transition-colors'>
                  {model.name}
                </h3>

                <div className='mt-4 grid grid-cols-3 gap-2 border-y border-white/5 py-3 text-xs'>
                  <div>
                    <span className='block text-[10px] text-slate-500'>
                      Context / Res
                    </span>
                    <span className='font-mono font-medium text-slate-200'>
                      {model.context}
                    </span>
                  </div>
                  <div>
                    <span className='block text-[10px] text-slate-500'>
                      Speed
                    </span>
                    <span className='font-mono font-medium text-emerald-400'>
                      {model.speed}
                    </span>
                  </div>
                  <div>
                    <span className='block text-[10px] text-slate-500'>
                      Est. Quota
                    </span>
                    <span className='font-mono font-medium text-sky-300'>
                      {model.price}
                    </span>
                  </div>
                </div>
              </div>

              <div className='mt-4 flex items-center justify-between pt-1'>
                <span className='text-xs text-slate-400'>
                  OpenAI SDK Compatible
                </span>
                <Link
                  to='/playground'
                  className='inline-flex items-center gap-1.5 text-xs font-semibold text-blue-400 group-hover:text-blue-300 transition-colors'
                >
                  {t('ทดสอบโมเดล')}
                  <ArrowRight className='size-3 transition-transform group-hover:translate-x-0.5' />
                </Link>
              </div>
            </div>
          ))}
        </div>

        {/* Bottom CTA to view all models */}
        <div className='mt-10 flex flex-col items-center justify-center gap-3 text-center sm:flex-row'>
          <span className='text-sm text-slate-400'>
            {t('พร้อมรองรับโมเดลอื่นๆ อีกกว่า 200+ รายการตามราคาต้นทุนจริง')}
          </span>
          <Button
            variant='outline'
            className='rounded-xl border-white/20 bg-white/5 text-xs text-white hover:bg-white/10'
            render={<Link to='/pricing' />}
          >
            <span>{t('ดูตารางโมเดลและราคาทั้งหมด')}</span>
            <ArrowRight className='size-3.5 ml-1' />
          </Button>
        </div>
      </div>
    </section>
  )
}
