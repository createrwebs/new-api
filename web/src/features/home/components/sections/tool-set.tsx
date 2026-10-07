import { Link } from '@tanstack/react-router'
import {
  ArrowRight,
  Bot,
  Crop,
  Layers,
  Palette,
  Sparkles,
  Video,
  Wand2,
  ZoomIn,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'

interface ToolItem {
  id: string
  title: string
  subtitle: string
  description: string
  image: string
  tag: string
  tagColor: string
  cost: string
  features: string[]
  studioUrl: string
  ctaLabel: string
}

export function ToolSet() {
  const { t } = useTranslation()

  const tools: ToolItem[] = [
    {
      id: 'bg-remove',
      title: t('AI Background Removal'),
      subtitle: t('ลบพื้นหลัง ไดคัทสินค้าเนียนกริบใน 1 วินาที'),
      description: t(
        'เทคโนโลยี BiRefNet AI แยกวัตถุออกจากฉากหลังอย่างแม่นยำ เก็บครบทุกรายละเอียดเส้นผมและขอบวัตถุโปร่งแสง พร้อมดาวน์โหลด PNG โปร่งใสทันที'
      ),
      image: '/images/tool-bg-remove.svg',
      tag: 'ไดคัทสินค้า e-Commerce',
      tagColor: 'border-cyan-500/30 bg-cyan-500/10 text-cyan-300',
      cost: '10 Credits ($0.02)',
      features: [
        'เก็บรายละเอียดขอบวัตถุและเส้นผมเนียนสนิท',
        'ได้ไฟล์ PNG พื้นหลังโปร่งแสงความละเอียดสูง',
        'รองรับ Batch อัปโหลดหลายภาพพร้อมกัน',
      ],
      studioUrl: '/studio',
      ctaLabel: t('ทดลองลบพื้นหลัง'),
    },
    {
      id: 'product-photo',
      title: t('Product Studio'),
      subtitle: t('สตูดิโอจัดฉากถ่ายภาพสินค้าออนไลน์แบบมืออาชีพ'),
      description: t(
        'เปลี่ยนรูปสินค้าถ่ายจากมือถือให้กลายเป็นภาพโฆษณาระดับแบรนด์ดัง ด้วย Preset สำหรับ Shopee, Lazada, TikTok Shop และ IG พร้อมแม่แบบ Luxury, Minimal Beige, Studio White'
      ),
      image: '/images/tool-product.jpg',
      tag: 'ยอดนิยมสำหรับพ่อค้าแม่ค้า',
      tagColor: 'border-amber-500/30 bg-amber-500/10 text-amber-300',
      cost: '50 Credits ($0.10)',
      features: [
        'สัดส่วนสำเร็จรูป Shopee 1:1, TikTok 9:16, IG 4:5',
        'แม่แบบ Luxury Black, Minimal Beige, Kitchen, Cosmetic',
        'รักษารายละเอียดโลโก้และรูปทรงสินค้าแท้ 100%',
      ],
      studioUrl: '/studio',
      ctaLabel: t('สร้างภาพสินค้าทันที'),
    },
    {
      id: 'image-upscale',
      title: t('AI Image Upscaler (4K)'),
      subtitle: t('ขยายภาพ 4x คมชัดสูง ไม่แตก ไม่เบลอ'),
      description: t(
        'ยกระดับภาพความละเอียดต่ำ ภาพแตก หรือภาพเก่า ให้กลายเป็นภาพคมกริบระดับ 4K Ultra HD ด้วย Clarity และ Real-ESRGAN ฟื้นฟูรายละเอียดพื้นผิวและดวงตาอย่างสมบูรณ์แบบ'
      ),
      image: '/images/tool-upscale.svg',
      tag: 'ขยายความละเอียด 4K',
      tagColor: 'border-purple-500/30 bg-purple-500/10 text-purple-300',
      cost: '15 - 25 Credits ($0.03 - $0.05)',
      features: [
        'ขยายความละเอียดได้สูงสุด 4 เท่าแบบไม่สูญเสียคุณภาพ',
        'Face Enhancement ฟื้นฟูใบหน้าและผิวเนียนคมชัด',
        'พร้อมสำหรับนำไปพิมพ์ป้ายโฆษณาหรือขึ้นหน้าเว็บ',
      ],
      studioUrl: '/studio',
      ctaLabel: t('ลองขยายความละเอียดภาพ'),
    },
    {
      id: 'image-to-video',
      title: t('Image to Video (Motion AI)'),
      subtitle: t('แปลงภาพนิ่งให้เคลื่อนไหวเป็นคลิป 5 วินาทีระดับภาพยนตร์'),
      description: t(
        'ปลุกภาพสินค้าหรือผลงานกราฟิกของคุณให้มีชีวิต ด้วยโมเดล Wan 2.2 และ Kling วิดีโอ 1080p 24fps พร้อมการเคลื่อนไหวมุมกล้องที่นุ่มนวล สมจริง น่าดึงดูดใจ'
      ),
      image: '/images/tool-video.svg',
      tag: 'สร้างคลิปโฆษณาโซเชียล',
      tagColor: 'border-indigo-500/30 bg-indigo-500/10 text-indigo-300',
      cost: '125 Credits ($0.25)',
      features: [
        'ความยาว 5 - 10 วินาที คมชัด 1080p Full HD',
        'มุมกล้องภาพยนตร์ Camera Zoom / Pan / Orbit',
        'เหมาะสำหรับทำ Reels, TikTok, และ Shorts',
      ],
      studioUrl: '/studio',
      ctaLabel: t('สร้างวิดีโอจากภาพ'),
    },
    {
      id: 'creator-assistant',
      title: t('Creator Assistant (ผู้ช่วย AI)'),
      subtitle: t('สั่งงานด้วยภาษาไทย วางแผนงานอัตโนมัติ ToolPlan'),
      description: t(
        'พิมพ์สั่งง่ายๆ เช่น "ทำรูปสินค้านี้ลง IG แล้วทำวิดีโอ 5 วิ" ผู้ช่วยจะจัดเรียงไดคัท จัดฉากสินค้า และแปลงเป็นคลิปวิดีโอให้จบครบในครั้งเดียว ปลอดภัยด้วย Step-Billing'
      ),
      image: '/images/tool-assistant.svg',
      tag: 'อัจฉริยะภาษาไทย',
      tagColor: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-300',
      cost: 'คิดตามจริงแยกตาม Step',
      features: [
        'แปลงคำสั่งภาษาไทยเป็น ToolPlan หลายขั้นตอนอัตโนมัติ',
        'คำนวณราคาโปร่งใส และยืนยันก่อนตัดเงินจริงเสมอ',
        'ตัดเงินทีละ Step หากขั้นไหนติดขัด คืนเงินเฉพาะขั้นนั้น',
      ],
      studioUrl: '/assistant',
      ctaLabel: t('ปรึกษาผู้ช่วย AI ทันที'),
    },
  ]

  return (
    <section className='relative border-t border-white/10 bg-[#080c16] px-4 py-20 sm:px-6 md:py-28'>
      <div className='mx-auto max-w-7xl'>
        {/* Section Header */}
        <div className='flex flex-col items-center text-center'>
          <div className='inline-flex items-center gap-2 rounded-full border border-blue-500/30 bg-blue-500/10 px-3.5 py-1.5 text-xs font-semibold text-blue-300'>
            <Wand2 className='size-3.5' />
            {t('Tora Creative Toolset (ชุดเครื่องมือสร้างสรรค์)')}
          </div>
          <h2 className='mt-4 text-3xl font-bold tracking-tight text-white sm:text-4xl md:text-5xl'>
            {t('ครบทุกเครื่องมือตกแต่งภาพและวิดีโอ จบในที่เดียว')}
          </h2>
          <p className='mt-4 max-w-2xl text-base text-slate-300'>
            {t(
              'ยกระดับผลงานของคุณด้วยเครื่องมือปัญญาประดิษฐ์ที่ถูกปรับแต่งมาเพื่อ Content Creator, พ่อค้าแม่ค้าออนไลน์ และ Thai SME โดยเฉพาะ'
            )}
          </p>
        </div>

        {/* Tools Showcase */}
        <div className='mt-16 space-y-12'>
          {tools.map((tool, index) => {
            const isReversed = index % 2 === 1
            return (
              <div
                key={tool.id}
                className='group overflow-hidden rounded-3xl border border-white/10 bg-[#0d1220] transition-all duration-300 hover:border-white/25 hover:shadow-2xl hover:shadow-blue-500/5'
              >
                <div
                  className={`flex flex-col lg:flex-row ${
                    isReversed ? 'lg:flex-row-reverse' : ''
                  }`}
                >
                  {/* Visual Preview Half */}
                  <div className='relative flex items-center justify-center overflow-hidden bg-black/40 p-6 lg:w-1/2 lg:p-10'>
                    <div className='relative aspect-[16/9] w-full overflow-hidden rounded-2xl border border-white/10 shadow-2xl'>
                      <img
                        src={tool.image}
                        alt={tool.title}
                        loading='lazy'
                        className='size-full object-cover transition-transform duration-500 group-hover:scale-102'
                      />
                    </div>
                  </div>

                  {/* Information Half */}
                  <div className='flex flex-col justify-between p-8 sm:p-10 lg:w-1/2'>
                    <div>
                      <div className='flex flex-wrap items-center gap-2.5'>
                        <span
                          className={`rounded-full border px-3 py-1 text-xs font-semibold ${tool.tagColor}`}
                        >
                          {tool.tag}
                        </span>
                        <span className='rounded-full border border-white/10 bg-white/5 px-3 py-1 text-xs font-mono text-slate-300'>
                          {tool.cost}
                        </span>
                      </div>

                      <h3 className='mt-4 text-2xl font-bold text-white sm:text-3xl'>
                        {tool.title}
                      </h3>
                      <p className='mt-2 text-sm font-medium text-sky-400'>
                        {tool.subtitle}
                      </p>
                      <p className='mt-3 text-sm leading-relaxed text-slate-300'>
                        {tool.description}
                      </p>

                      {/* Feature Bullet Points */}
                      <ul className='mt-6 space-y-2.5'>
                        {tool.features.map((feat) => (
                          <li
                            key={feat}
                            className='flex items-center gap-2.5 text-xs text-slate-200'
                          >
                            <span className='size-1.5 rounded-full bg-blue-400' />
                            <span>{feat}</span>
                          </li>
                        ))}
                      </ul>
                    </div>

                    {/* Action Button */}
                    <div className='mt-8 pt-4 border-t border-white/10'>
                      <Button
                        className='h-11 rounded-xl bg-blue-600 px-6 text-xs font-semibold text-white shadow-lg shadow-blue-500/25 hover:bg-blue-500'
                        render={<Link to={tool.studioUrl} />}
                      >
                        <span>{tool.ctaLabel}</span>
                        <ArrowRight className='size-4 ml-2' />
                      </Button>
                    </div>
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      </div>
    </section>
  )
}
