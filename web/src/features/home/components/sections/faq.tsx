import { HelpCircle, MessageCircle } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'

export function FAQ() {
  const { t } = useTranslation()

  const faqs = [
    {
      id: 'faq-credits',
      question: t('Tora Credits มีวันหมดอายุหรือไม่ และคิดคำนวณอย่างไร?'),
      answer: t(
        'เครดิตในกระเป๋า Tora ของคุณไม่มีวันหมดอายุ! ระบบคิดค่าบริการแบบจ่ายตามจริง (Pay-As-You-Go) โดย 1 Tora Credit = 1,000 Quota (มูลค่าประมาณ 0.07 บาท หรือ $0.002 USD) ไม่มีการเรียกเก็บค่าสมาชิกรายเดือนแอบแฝง เติมเท่าไหร่ ใช้ได้เท่านั้นตลอดไป'
      ),
    },
    {
      id: 'faq-commercial',
      question: t(
        'สามารถนำรูปภาพและวิดีโอที่สร้างจาก Tora Studio ไปใช้ในเชิงพาณิชย์ (Commercial Use) ได้หรือไม่?'
      ),
      answer: t(
        'สามารถนำไปใช้ในเชิงพาณิชย์ได้ 100%! สิทธิ์ในผลงานภาพและวิดีโอทั้งหมดที่สร้างผ่าน Tora Creative Studio เป็นของผู้ใช้งานโดยสมบูรณ์ สามารถนำไปใช้ยิงแอดโฆษณา ลงขายสินค้าบน Shopee, Lazada, TikTok Shop, ทำคอนเทนต์ลง YouTube หรือสื่อสิ่งพิมพ์ได้ทันที'
      ),
    },
    {
      id: 'faq-api-integration',
      question: t(
        'จะเริ่มต้นเชื่อมต่อ API เข้ากับโปรเจกต์ Next.js, Python หรือ Cursor ได้อย่างไร?'
      ),
      answer: t(
        'Tora API รองรับมาตรฐาน OpenAI Specification 100% เพียงเปลี่ยน baseURL เป็น https://api.toraapi.com/v1 และใช้ API Key จากหน้าแดชบอร์ด คุณสามารถเรียกใช้งานทั้ง GPT-4o, Claude 3.7 Sonnet, DeepSeek R1, หรือ Gemini 2.5 Pro ได้ทันทีโดยไม่ต้องแก้ไขโค้ด SDK เดิม'
      ),
    },
    {
      id: 'faq-failed-jobs',
      question: t('หากโมเดลปลายทางประมวลผลล้มเหลว จะถูกหักเครดิตฟรีหรือไม่?'),
      answer: t(
        'ไม่ถูกหักเครดิตแน่นอนครับ! ระบบ Tora ใช้สถาปัตยกรรม Atomic Safe Billing โดยระบบจะทำการจอง (Reserve) เครดิตไว้ชั่วคราว และจะตัดเงินจริง (Settle) เมื่อได้รับผลลัพธ์ที่สำเร็จเท่านั้น หากโมเดล Upstream ขัดข้อง เครดิตจะถูกคืนเข้ากระเป๋าของคุณอัตโนมัติทันที 100%'
      ),
    },
    {
      id: 'faq-assistant',
      question: t('Creator Assistant (ผู้ช่วย AI) ทำงานอย่างไร และคิดเงินอย่างไร?'),
      answer: t(
        'ผู้ช่วย AI รับคำสั่งภาษาไทย เช่น "ทำรูปสินค้านี้ลง IG แล้วทำวิดีโอ 5 วิ" แล้ววางแผนเป็น ToolPlan หลายขั้นตอน พร้อมดึงราคาจริงมาให้คุณกดยืนยันก่อนตัดเงินเสมอ การตัดเงินจะทำทีละขั้นตอน หากขั้นที่ 3 ขัดข้อง ระบบจะไม่คืนเงินของขั้นที่ 1-2 ที่ทำเสร็จแล้ว และคุณสามารถกดปุ่ม Retry เฉพาะขั้นตอนที่ล้มเหลวได้โดยไม่ต้องจ่ายซ้ำ'
      ),
    },
    {
      id: 'faq-invoice',
      question: t(
        'มีใบเสร็จรับเงิน หรือเอกสารสำหรับบริษัท / นิติบุคคลหรือไม่?'
      ),
      answer: t(
        'มีครับ ระบบรองรับการออกใบเสร็จรับเงินสำหรับทุกยอดการเติมเงินผ่าน PromptPay และบัตรเครดิต สำหรับลูกค้านิติบุคคลหรือบริษัทที่ต้องการใบเสร็จ/ใบกำกับภาษีเต็มรูปแบบ สามารถกรอกข้อมูลนิติบุคคลในหน้าตั้งค่าบัญชี หรือติดต่อทีมงาน Support เพื่อออกเอกสารได้ทันที'
      ),
    },
  ]

  return (
    <section className='relative border-t border-white/10 bg-[#090d18] px-4 py-20 sm:px-6 md:py-28'>
      <div className='mx-auto max-w-4xl'>
        {/* Header */}
        <div className='flex flex-col items-center text-center'>
          <div className='inline-flex items-center gap-2 rounded-full border border-sky-500/30 bg-sky-500/10 px-3.5 py-1.5 text-xs font-semibold text-sky-300'>
            <HelpCircle className='size-3.5' />
            {t('คำถามที่พบบ่อย (FAQ)')}
          </div>
          <h2 className='mt-4 text-3xl font-bold tracking-tight text-white sm:text-4xl'>
            {t('มีข้อสงสัย? เรามีคำตอบให้คุณ')}
          </h2>
          <p className='mt-4 max-w-xl text-base text-slate-300'>
            {t(
              'คำถามสำคัญเกี่ยวกับการใช้งาน Tora Credits, การเชื่อมต่อ API, ความปลอดภัย และสิทธิ์การใช้งานผลงาน'
            )}
          </p>
        </div>

        {/* Accordion Component */}
        <div className='mt-12 rounded-3xl border border-white/10 bg-[#0d1222] p-6 sm:p-8'>
          <Accordion defaultValue={['faq-credits']} className='space-y-4'>
            {faqs.map((faq) => (
              <AccordionItem
                key={faq.id}
                value={faq.id}
                className='rounded-2xl border border-white/10 bg-white/5 px-5 py-2 transition-colors hover:border-white/20'
              >
                <AccordionTrigger className='text-left text-base font-semibold text-white hover:text-sky-300 hover:no-underline'>
                  {faq.question}
                </AccordionTrigger>
                <AccordionContent className='pt-2 text-sm leading-relaxed text-slate-300'>
                  {faq.answer}
                </AccordionContent>
              </AccordionItem>
            ))}
          </Accordion>
        </div>

        {/* Contact Support Footer Box */}
        <div className='mt-10 flex flex-col items-center justify-between gap-4 rounded-2xl border border-white/10 bg-white/5 p-6 text-center sm:flex-row sm:text-left'>
          <div className='flex items-center gap-3.5'>
            <div className='flex size-10 items-center justify-center rounded-xl bg-blue-500/10 text-blue-400'>
              <MessageCircle className='size-5' />
            </div>
            <div>
              <p className='text-sm font-semibold text-white'>
                {t('ยังมีข้อสงสัยเพิ่มเติม?')}
              </p>
              <p className='text-xs text-slate-400'>
                {t('ทีมงาน Tora ยินดีให้คำปรึกษาและช่วยเหลือตลอดเวลา')}
              </p>
            </div>
          </div>
          <a
            href='https://t.me/toraapi'
            target='_blank'
            rel='noopener noreferrer'
            className='inline-flex items-center justify-center rounded-xl bg-white/10 px-4 py-2 text-xs font-semibold text-white transition-colors hover:bg-white/20'
          >
            {t('ติดต่อฝ่ายสนับสนุน')}
          </a>
        </div>
      </div>
    </section>
  )
}
