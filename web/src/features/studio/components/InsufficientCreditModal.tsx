import { useEffect } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { AlertCircle, Coins, Sparkles } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  trackStudioBuyCreditClick,
  trackStudioInsufficientCredit,
} from '@/lib/analytics'

import { recordStudioAttribution } from '../api'
import type { InsufficientCreditData } from '../types'

interface InsufficientCreditModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  data: InsufficientCreditData | null
  onPreserveStateAndTopUp: () => void
}

export function InsufficientCreditModal({
  open,
  onOpenChange,
  data,
  onPreserveStateAndTopUp,
}: InsufficientCreditModalProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()

  useEffect(() => {
    if (open && data) {
      const toolId = data.tool_id || 'studio'
      trackStudioInsufficientCredit(toolId, data.required_credits, data.current_credits)
      recordStudioAttribution('insufficient_credit', toolId, data.required_credits)
    }
  }, [open, data])

  if (!data) return null

  const handleTopUp = () => {
    const toolId = data.tool_id || 'studio'
    trackStudioBuyCreditClick(toolId, data.required_credits)
    recordStudioAttribution('buy_credit_click', toolId, data.required_credits)
    onPreserveStateAndTopUp()
    sessionStorage.setItem('tora_studio_purchase_origin', toolId)
    sessionStorage.setItem('tora_studio_purchase_needed', String(data.required_credits))
    onOpenChange(false)
    navigate({ to: '/wallet' })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='max-w-md sm:max-w-lg'>
        <DialogHeader>
          <div className='flex items-center gap-3'>
            <div className='flex size-10 items-center justify-center rounded-full bg-amber-500/15 text-amber-500'>
              <Coins className='size-5' />
            </div>
            <div>
              <DialogTitle className='text-lg font-semibold'>
                {t('Tora Credits ไม่เพียงพอ')}
              </DialogTitle>
              <DialogDescription className='text-sm text-muted-foreground'>
                {t('กรุณาเติมเครดิตเพื่อดำเนินการสร้างผลงานใน Tora Studio')}
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <div className='space-y-4 py-2'>
          <div className='grid grid-cols-3 gap-2 rounded-xl bg-muted/50 p-3 text-center'>
            <div className='rounded-lg bg-background p-2 shadow-xs'>
              <div className='text-xs text-muted-foreground'>{t('ต้องใช้')}</div>
              <div className='text-lg font-bold text-foreground'>
                {data.required_credits}{' '}
                <span className='text-xs font-normal text-muted-foreground'>
                  Cr
                </span>
              </div>
            </div>

            <div className='rounded-lg bg-background p-2 shadow-xs'>
              <div className='text-xs text-muted-foreground'>{t('คงเหลือ')}</div>
              <div className='text-lg font-bold text-foreground'>
                {data.current_credits}{' '}
                <span className='text-xs font-normal text-muted-foreground'>
                  Cr
                </span>
              </div>
            </div>

            <div className='rounded-lg bg-rose-500/10 p-2 shadow-xs dark:bg-rose-950/20'>
              <div className='text-xs text-rose-600 dark:text-rose-400'>
                {t('ขาดอีก')}
              </div>
              <div className='text-lg font-bold text-rose-600 dark:text-rose-400'>
                {data.missing_credits}{' '}
                <span className='text-xs font-normal text-rose-600/70'>
                  Cr
                </span>
              </div>
            </div>
          </div>

          <div className='rounded-lg border border-border/60 bg-background/50 p-3 text-xs leading-relaxed text-muted-foreground'>
            <div className='mb-1 flex items-center gap-1.5 font-medium text-foreground'>
              <Sparkles className='size-3.5 text-primary' />
              <span>{t('หลักการ Universal Tora Credits')}</span>
            </div>
            <p>
              {t(
                '1 Tora Credit = 1,000 Quota units (~0.07 THB) เติมเพียงกระเป๋าเดียว สามารถใช้งานร่วมกันได้ทั้งระบบ Chat, Image, Video และ Creator Studio ไม่ต้องซื้อแพ็กเกจแยก'
              )}
            </p>
          </div>

          <div className='flex items-center gap-2 text-xs text-muted-foreground'>
            <AlertCircle className='size-4 shrink-0 text-blue-500' />
            <span>
              {t(
                'ระบบได้บันทึกคำสั่งและพารามิเตอร์ที่คุณตั้งไว้เรียบร้อยแล้ว เมื่อเติมเครดิตเสร็จจะสามารถกดสร้างต่อได้ทันที'
              )}
            </span>
          </div>
        </div>

        <DialogFooter className='flex-col gap-2 sm:flex-row'>
          <Button
            variant='outline'
            onClick={() => onOpenChange(false)}
            className='w-full sm:w-auto'
          >
            {t('ยกเลิก')}
          </Button>
          <Button
            onClick={handleTopUp}
            className='w-full bg-primary font-medium sm:w-auto'
          >
            <Coins className='mr-1.5 size-4' />
            {t('เติมเครดิตที่ Wallet')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
