import { useNavigate } from '@tanstack/react-router'
import { Sparkles, UserPlus, LogIn, ArrowRight } from 'lucide-react'
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

interface AuthRequiredModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  toolName?: string
  actionDescription?: string
}

export function AuthRequiredModal({
  open,
  onOpenChange,
  toolName,
  actionDescription,
}: AuthRequiredModalProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const handleAuthRedirect = (target: '/sign-up' | '/sign-in') => {
    onOpenChange(false)
    const redirectUrl =
      typeof window !== 'undefined'
        ? window.location.pathname + window.location.search
        : '/studio'

    navigate({
      to: target,
      search: { redirect: redirectUrl },
    })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='max-w-md sm:max-w-lg'>
        <DialogHeader className='text-left'>
          <div className='flex items-center gap-3 mb-2'>
            <div className='flex size-11 items-center justify-center rounded-2xl bg-primary/10 text-primary'>
              <Sparkles className='size-6' />
            </div>
            <div>
              <DialogTitle className='text-lg font-bold tracking-tight text-foreground'>
                {t('เข้าสู่ระบบหรือสมัครสมาชิก')}
              </DialogTitle>
              <DialogDescription className='text-xs text-muted-foreground'>
                {toolName
                  ? t('เพื่อใช้งาน {{tool}} และสร้างผลงานด้วย AI', { tool: toolName })
                  : t('เพื่อเริ่มใช้งาน Tora Studio และสร้างผลงานด้วย AI')}
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <div className='space-y-4 py-2'>
          <p className='text-sm text-foreground/90 leading-relaxed'>
            {actionDescription ||
              t(
                'คุณยังไม่ได้เข้าสู่ระบบ สมัครสมาชิกตอนนี้เพื่อรับเครดิตและสร้างผลงาน AI ได้ทันที'
              )}
          </p>

          <div className='rounded-xl border border-border/70 bg-muted/40 p-3.5 space-y-2.5'>
            <div className='flex items-start gap-2.5 text-xs text-muted-foreground'>
              <span className='flex size-5 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary font-bold text-[10px]'>
                ✓
              </span>
              <span>
                <strong className='text-foreground font-semibold'>
                  {t('รับเครดิตทดลองใช้')}
                </strong>{' '}
                — {t('สมัครสมาชิกใหม่เพื่อรับเครดิตสร้างภาพและวิดีโอ')}
              </span>
            </div>
            <div className='flex items-start gap-2.5 text-xs text-muted-foreground'>
              <span className='flex size-5 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary font-bold text-[10px]'>
                ✓
              </span>
              <span>
                <strong className='text-foreground font-semibold'>
                  {t('บันทึกผลงาน')}
                </strong>{' '}
                — {t('ดูประวัติ ดาวน์โหลด และ Remix ผลงานย้อนหลังได้ตลอดเวลา')}
              </span>
            </div>
            <div className='flex items-start gap-2.5 text-xs text-muted-foreground'>
              <span className='flex size-5 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary font-bold text-[10px]'>
                ✓
              </span>
              <span>
                <strong className='text-foreground font-semibold'>
                  {t('ไม่มีค่าสมาชิกรายเดือน')}
                </strong>{' '}
                — {t('จ่ายตามการใช้งานจริงผ่าน PromptPay หรือบัตรเครดิต')}
              </span>
            </div>
          </div>
        </div>

        <DialogFooter className='flex flex-col-reverse gap-2 sm:flex-row sm:justify-end sm:gap-2 pt-2'>
          <Button
            variant='ghost'
            onClick={() => onOpenChange(false)}
            className='h-9 text-xs text-muted-foreground'
          >
            {t('ไว้คราวหน้า')}
          </Button>
          <Button
            variant='outline'
            onClick={() => handleAuthRedirect('/sign-in')}
            className='h-9 text-xs gap-1.5'
          >
            <LogIn className='size-3.5' />
            {t('เข้าสู่ระบบ')}
          </Button>
          <Button
            onClick={() => handleAuthRedirect('/sign-up')}
            className='h-9 text-xs font-semibold gap-1.5 shadow-sm'
          >
            <UserPlus className='size-3.5' />
            {t('สมัครสมาชิกใหม่')}
            <ArrowRight className='size-3.5 ml-0.5' />
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
