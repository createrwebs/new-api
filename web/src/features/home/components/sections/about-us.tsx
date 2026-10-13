import { Link } from '@tanstack/react-router'
import {
  CreditCard,
  Globe2,
  Lock,
  QrCode,
  Server,
  ShieldCheck,
  Sparkles,
  Users2,
  Wallet,
  Zap,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

export function AboutUs() {
  const { t } = useTranslation()

  const pillars = [
    {
      icon: Wallet,
      title: t('กระเป๋าเงินเดียว เข้าถึงทุกโมเดล (Universal Wallet)'),
      description: t(
        'หมดปัญหาการผูกบัตรเครดิตหลายสิบบัญชี หรือต้องจ่ายค่า Subscription รายเดือนทิ้งไว้ เติมเครดิต Tora Credits เพียงกระเป๋าเดียว ใช้งานได้ทั้ง Text, Image, Video และ Code'
      ),
      accent: 'from-blue-500/20 to-sky-500/5 text-blue-400 border-blue-500/20',
    },
    {
      icon: QrCode,
      title: t('รองรับ PromptPay & ธุรกิจไทย (Thai SME Friendly)'),
      description: t(
        'ออกแบบมาเพื่อคนไทยอย่างแท้จริง สแกนจ่ายผ่าน QR PromptPay หรือบัตรเครดิตไทย เครดิตเข้าบัญชีทันทีแบบ Real-time พร้อมเอกสารใบเสร็จที่นำไปใช้ในทางบัญชีได้'
      ),
      accent:
        'from-emerald-500/20 to-teal-500/5 text-emerald-400 border-emerald-500/20',
    },
    {
      icon: Server,
      title: t('เสถียรภาพ 99.9% ด้วยระบบ Auto Route Fallback'),
      description: t(
        'ระบบกระจายโหลดอัจฉริยะแบบ Multi-Provider (Fal, Replicate, Direct) หากผู้ให้บริการหลักเกิดข้อผิดพลาด ระบบจะสลับเส้นทางสำรองอัตโนมัติ การันตีงานของคุณไม่สะดุด'
      ),
      accent:
        'from-purple-500/20 to-indigo-500/5 text-purple-400 border-purple-500/20',
    },
    {
      icon: ShieldCheck,
      title: t('ความปลอดภัยระดับองค์กร & Latency ต่ำพิเศษ'),
      description: t(
        'เซิร์ฟเวอร์ Edge Gateway ในภูมิภาคเอเชีย ตอบสนองเร็วกว่า พร้อมระบบจัดการคีย์ API ระดับมืออาชีพ (Subnet IP Restriction, Quota Limiter, และ Audit Logs)'
      ),
      accent: 'from-amber-500/20 to-orange-500/5 text-amber-400 border-amber-500/20',
    },
  ]

  return (
    <section className='relative border-t border-white/10 bg-[#070b14] px-4 py-20 sm:px-6 md:py-28'>
      <div className='mx-auto max-w-7xl'>
        {/* Header */}
        <div className='flex flex-col items-center text-center'>
          <div className='inline-flex items-center gap-2 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-3.5 py-1.5 text-xs font-semibold text-emerald-300'>
            <Users2 className='size-3.5' />
            {t('เกี่ยวกับเรา (About Tora AI)')}
          </div>
          <h2 className='mt-4 text-3xl font-bold tracking-tight text-white sm:text-4xl md:text-5xl'>
            {t('โครงสร้างพื้นฐาน AI ที่สร้างขึ้นเพื่อคุณ')}
          </h2>
          <p className='mt-4 max-w-2xl text-base text-slate-300'>
            {t(
              'Tora AI เป็นแพลตฟอร์มเกตเวย์และสตูดิโอสร้างสรรค์คอนเทนต์ ที่รวมโมเดล AI ระดับโลกมาไว้ในที่เดียว ให้คุณสร้างสรรค์ผลงาน พัฒนาแอปพลิเคชัน และขยายธุรกิจได้อย่างไร้ขีดจำกัด'
            )}
          </p>
        </div>

        {/* Pillars Grid */}
        <div className='mt-16 grid gap-6 sm:grid-cols-2 lg:grid-cols-4'>
          {pillars.map((item) => {
            const Icon = item.icon
            return (
              <div
                key={item.title}
                className='group relative flex flex-col justify-between overflow-hidden rounded-2xl border border-white/10 bg-[#0c101c] p-6 transition-all duration-300 hover:-translate-y-1 hover:border-white/20 hover:shadow-xl'
              >
                <div>
                  <div
                    className={`inline-flex size-12 items-center justify-center rounded-xl border bg-gradient-to-br ${item.accent}`}
                  >
                    <Icon className='size-6' />
                  </div>
                  <h3 className='mt-5 text-lg font-bold text-white'>
                    {item.title}
                  </h3>
                  <p className='mt-3 text-xs leading-relaxed text-slate-300'>
                    {item.description}
                  </p>
                </div>
              </div>
            )
          })}
        </div>

        {/* Community & Enterprise Stats Banner */}
        <div className='mt-16 rounded-3xl border border-white/10 bg-gradient-to-r from-blue-900/30 via-indigo-900/20 to-purple-900/30 p-8 sm:p-12'>
          <div className='grid gap-8 sm:grid-cols-2 lg:grid-cols-4 text-center'>
            <div>
              <span className='block text-3xl font-extrabold text-white sm:text-4xl'>
                240+
              </span>
              <span className='mt-1 block text-xs font-medium text-slate-300'>
                {t('โมเดล AI ในระบบ')}
              </span>
            </div>
            <div>
              <span className='block text-3xl font-extrabold text-sky-400 sm:text-4xl'>
                99.9%
              </span>
              <span className='mt-1 block text-xs font-medium text-slate-300'>
                {t('ความเสถียร Uptime SLA')}
              </span>
            </div>
            <div>
              <span className='block text-3xl font-extrabold text-emerald-400 sm:text-4xl'>
                &lt; 45ms
              </span>
              <span className='mt-1 block text-xs font-medium text-slate-300'>
                {t('Edge Latency ในภูมิภาค')}
              </span>
            </div>
            <div>
              <span className='block text-3xl font-extrabold text-amber-400 sm:text-4xl'>
                1 Wallet
              </span>
              <span className='mt-1 block text-xs font-medium text-slate-300'>
                {t('กระเป๋าเดียวใช้ได้ทุกเครื่องมือ')}
              </span>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
