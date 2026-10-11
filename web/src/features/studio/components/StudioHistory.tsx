import { useQuery } from '@tanstack/react-query'
import {
  Coins,
  Download,
  Layers,
  RotateCcw,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardFooter, CardHeader } from '@/components/ui/card'
import { Spinner } from '@/components/ui/spinner'

import { getUserStudioJobs } from '../api'
import type { StudioJob, StudioTool } from '../types'

interface StudioHistoryProps {
  tools: StudioTool[]
  onRemix: (job: StudioJob) => void
}

export function StudioHistory({ tools, onRemix }: StudioHistoryProps) {
  const { t } = useTranslation()

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['studio', 'history'],
    queryFn: () => getUserStudioJobs(1, 30),
    staleTime: 10 * 1000,
  })

  const jobs = data?.jobs || []

  if (isLoading) {
    return (
      <div className='flex h-64 items-center justify-center'>
        <Spinner className='size-8 text-primary' />
      </div>
    )
  }

  if (jobs.length === 0) {
    return (
      <div className='flex min-h-[350px] flex-col items-center justify-center rounded-2xl border border-dashed border-border p-8 text-center'>
        <div className='flex size-14 items-center justify-center rounded-2xl bg-muted text-muted-foreground'>
          <Layers className='size-7' />
        </div>
        <h3 className='mt-4 text-sm font-semibold text-foreground'>
          {t('ยังไม่มีประวัติการสร้างผลงาน')}
        </h3>
        <p className='mt-1 max-w-sm text-xs text-muted-foreground'>
          {t(
            'เมื่อคุณสร้างภาพ วิดีโอ หรือใช้เครื่องมือตัดต่อ AI ใน Tora Studio ผลงานทั้งหมดของคุณจะแสดงที่นี่'
          )}
        </p>
      </div>
    )
  }

  const renderBadgeVariant = (status: string) => {
    if (status === 'SUCCEEDED') return 'default'
    if (status === 'FAILED') return 'destructive'
    return 'secondary'
  }

  return (
    <div className='space-y-6'>
      <div className='flex items-center justify-between'>
        <div>
          <h2 className='text-base font-semibold text-foreground'>
            {t('ประวัติผลงานของคุณ')} ({jobs.length})
          </h2>
          <p className='text-xs text-muted-foreground'>
            {t('ดูผลลัพธ์ย้อนหลัง ดาวน์โหลดไฟล์ หรือกด Remix เพื่อสร้างต่อยอด')}
          </p>
        </div>
        <Button size='sm' variant='outline' onClick={() => refetch()} className='h-8 text-xs'>
          {t('รีเฟรช')}
        </Button>
      </div>

      <div className='grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4'>
        {jobs.map((job) => {
          const tool = tools.find((t) => t.id === job.tool_id || t.slug === job.tool_id)
          let outputUrl: string | null = null
          let prompt = ''

          try {
            if (job.output_data) {
              const out = JSON.parse(job.output_data)
              outputUrl = out.output_url || out.image_url || out.video_url || null
            }
            if (job.input_params) {
              const inp = JSON.parse(job.input_params)
              prompt = inp.prompt || ''
            }
          } catch {
            // Ignore
          }

          const isVideo = tool?.category === 'video' || job.tool_id.includes('video')

          return (
            <Card
              key={job.id}
              className='group flex flex-col justify-between overflow-hidden border-border/60 transition-all hover:border-primary/40 hover:shadow-xs'
            >
              <CardHeader className='p-3 pb-2'>
                <div className='flex items-center justify-between gap-1 text-xs'>
                  <span className='font-semibold text-foreground line-clamp-1'>
                    {tool?.name_th || tool?.name || job.tool_id}
                  </span>
                  <Badge variant={renderBadgeVariant(job.status)} className='text-[9px]'>
                    {job.status}
                  </Badge>
                </div>
              </CardHeader>

              <CardContent className='p-3 pt-0'>
                {/* Media Preview Box */}
                <div className='relative aspect-square overflow-hidden rounded-lg bg-muted'>
                  {outputUrl && isVideo && (
                    <video
                      src={outputUrl}
                      className='size-full object-cover'
                      muted
                      loop
                      onMouseEnter={(e) => e.currentTarget.play()}
                      onMouseLeave={(e) => e.currentTarget.pause()}
                    />
                  )}
                  {outputUrl && !isVideo && (
                    <img
                      src={outputUrl}
                      alt='Output'
                      className='size-full object-cover transition-transform group-hover:scale-105'
                      loading='lazy'
                    />
                  )}
                  {!outputUrl && (
                    <div className='flex size-full items-center justify-center text-[10px] text-muted-foreground p-3 text-center'>
                      {job.status === 'FAILED'
                        ? t('สร้างไม่สำเร็จ (คืนเครดิตแล้ว)')
                        : t('กำลังประมวลผล...')}
                    </div>
                  )}
                </div>

                {prompt && (
                  <p className='mt-2 text-[11px] text-muted-foreground line-clamp-2'>
                    {prompt}
                  </p>
                )}

                <div className='mt-2 flex items-center justify-between text-[10px] text-muted-foreground'>
                  <span className='flex items-center gap-1'>
                    <Coins className='size-3' />
                    {job.credit_charged} Cr
                  </span>
                  <span>{new Date(job.created_at).toLocaleDateString()}</span>
                </div>
              </CardContent>

              <CardFooter className='flex items-center gap-1.5 p-3 pt-0'>
                <Button
                  size='sm'
                  variant='outline'
                  className='h-7 flex-1 text-xs'
                  onClick={() => onRemix(job)}
                >
                  <RotateCcw className='mr-1 size-3' />
                  {t('Remix')}
                </Button>
                {outputUrl && (
                  <Button
                    size='sm'
                    className='h-7 text-xs'
                    onClick={() => {
                      if (outputUrl) {
                        window.open(outputUrl, '_blank')
                      }
                    }}
                  >
                    <Download className='size-3' />
                  </Button>
                )}
              </CardFooter>
            </Card>
          )
        })}
      </div>
    </div>
  )
}
