import {
  AlertCircle,
  CheckCircle2,
  Clock,
  Coins,
  Copy,
  Download,
  ImageIcon,
  Loader2,
  RotateCcw,
  Sparkles,
  Upload,
  Video,
  Wand2,
} from 'lucide-react'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { NativeSelect } from '@/components/ui/native-select'
import { Progress } from '@/components/ui/progress'
import { Textarea } from '@/components/ui/textarea'
import { useAuthStore } from '@/stores/auth-store'

import {
  cancelStudioJob,
  createStudioJob,
  getStudioJob,
  InsufficientCreditException,
  quoteStudioJob,
  recordStudioAttribution,
} from '../api'
import {
  trackStudioGenerationAfterPurchase,
  trackStudioPurchaseReturn,
} from '@/lib/analytics'
import type {
  InsufficientCreditData,
  SavedPendingJob,
  StudioJob,
  StudioTemplate,
  StudioTool,
} from '../types'
import { InsufficientCreditModal } from './InsufficientCreditModal'

const STORAGE_KEY = 'tora_studio_pending_job'

interface StudioPlaygroundProps {
  tools: StudioTool[]
  templates: StudioTemplate[]
  selectedToolSlug: string
  initialTemplate?: StudioTemplate | null
  onToolChange: (toolSlug: string) => void
  onJobCompleted?: (job: StudioJob) => void
}

export function StudioPlayground({
  tools,
  templates,
  selectedToolSlug,
  initialTemplate,
  onToolChange,
  onJobCompleted,
}: StudioPlaygroundProps) {
  const { t } = useTranslation()
  const { auth } = useAuthStore()

  // Current selected tool
  const currentTool = useMemo(() => {
    return (
      tools.find((tool) => tool.slug === selectedToolSlug || tool.id === selectedToolSlug) ||
      tools[0]
    )
  }, [tools, selectedToolSlug])

  // Relevant templates
  const toolTemplates = useMemo(() => {
    if (!currentTool) return []
    return templates.filter((tpl) => tpl.tool_id === currentTool.id)
  }, [templates, currentTool])

  // Form states
  const [prompt, setPrompt] = useState('')
  const [negativePrompt, setNegativePrompt] = useState('')
  const [aspectRatio, setAspectRatio] = useState('1:1')
  const [imageUrl, setImageUrl] = useState('')
  const [scaleFactor, setScaleFactor] = useState(2)
  const [durationSec, setDurationSec] = useState(5)
  const [activeTemplateId, setActiveTemplateId] = useState<string | null>(null)
  const [providerOverride, setProviderOverride] = useState<'fal' | 'mock' | 'auto'>('auto')
  const [confirmedRights, setConfirmedRights] = useState(false)

  // Job execution state
  const [activeJob, setActiveJob] = useState<StudioJob | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [elapsedSec, setElapsedSec] = useState(0)
  const timerRef = useRef<NodeJS.Timeout | null>(null)
  const pollRef = useRef<NodeJS.Timeout | null>(null)

  // Insufficient credits modal
  const [insufficientModalOpen, setInsufficientModalOpen] = useState(false)
  const [insufficientData, setInsufficientData] =
    useState<InsufficientCreditData | null>(null)

  // Credit Conversion Funnel: Post-purchase return state (Queue 3)
  const [isPostPurchaseReturn, setIsPostPurchaseReturn] = useState(false)
  const [freshQuoteCredits, setFreshQuoteCredits] = useState<number | null>(null)

  // Calculate user credits
  const userCredits = Math.floor((auth.user?.quota || 0) / 1000)
  const requiredCredits = currentTool ? currentTool.credit_cost : 10
  const hasEnoughCredits = userCredits >= requiredCredits

  const applyTemplate = useCallback((tpl: StudioTemplate) => {
    setActiveTemplateId(tpl.id)
    try {
      const inputs = JSON.parse(tpl.sample_inputs || '{}')
      if (inputs.prompt) setPrompt(inputs.prompt)
      if (inputs.image_url) setImageUrl(inputs.image_url)
      if (inputs.aspect_ratio) setAspectRatio(inputs.aspect_ratio)
      toast.success(t('โหลดพรีเซ็ต {{title}} สำเร็จ', { title: tpl.title_th || tpl.title }))
    } catch {
      // Ignore
    }
  }, [t])

  // Restore state from localStorage if available (with 2-hour TTL enforcement)
  useEffect(() => {
    try {
      const saved = localStorage.getItem(STORAGE_KEY)
      if (saved) {
        const parsed: SavedPendingJob = JSON.parse(saved)
        const TWO_HOURS_MS = 2 * 60 * 60 * 1000
        if (parsed.timestamp && Date.now() - parsed.timestamp > TWO_HOURS_MS) {
          localStorage.removeItem(STORAGE_KEY)
          return
        }

        if (parsed.tool_id) {
          onToolChange(parsed.tool_id)
        }
        if (parsed.template_id) {
          setActiveTemplateId(parsed.template_id)
        }
        if (parsed.input_params) {
          if (typeof parsed.input_params.prompt === 'string') {
            setPrompt(parsed.input_params.prompt)
          }
          if (typeof parsed.input_params.image_url === 'string') {
            setImageUrl(parsed.input_params.image_url)
          }
          if (typeof parsed.input_params.aspect_ratio === 'string') {
            setAspectRatio(parsed.input_params.aspect_ratio)
          }
        }
        toast.info(t('กู้คืนข้อมูลคำสั่งที่บันทึกไว้เรียบร้อยแล้ว'))
      }

      // Detect post-purchase return from wallet top-up (Queue 3 Funnel)
      const purchaseOrigin = sessionStorage.getItem('tora_studio_purchase_origin')
      if (purchaseOrigin) {
        setIsPostPurchaseReturn(true)
        trackStudioPurchaseReturn(purchaseOrigin)
        recordStudioAttribution('purchase_return', purchaseOrigin)
      }
    } catch {
      // Ignore
    }
  }, [onToolChange, t])

  // Apply initial template if passed
  useEffect(() => {
    if (initialTemplate) {
      applyTemplate(initialTemplate)
    }
  }, [initialTemplate, applyTemplate])

  // Cleanup timers on unmount
  useEffect(() => {
    return () => {
      if (timerRef.current) clearInterval(timerRef.current)
      if (pollRef.current) clearInterval(pollRef.current)
    }
  }, [])

  // Elapsed time counter
  useEffect(() => {
    const isRunning =
      activeJob &&
      (activeJob.status === 'RESERVED' ||
        activeJob.status === 'SUBMITTING' ||
        activeJob.status === 'PROCESSING')

    if (isRunning) {
      if (!timerRef.current) {
        setElapsedSec(0)
        timerRef.current = setInterval(() => {
          setElapsedSec((prev) => prev + 1)
        }, 1000)
      }
    } else {
      if (timerRef.current) {
        clearInterval(timerRef.current)
        timerRef.current = null
      }
    }
  }, [activeJob])

  // Build input params payload according to tool needs
  const buildInputParams = useCallback((): Record<string, unknown> => {
    const params: Record<string, unknown> = {}
    if (prompt.trim()) params.prompt = prompt.trim()
    if (negativePrompt.trim()) params.negative_prompt = negativePrompt.trim()
    if (imageUrl.trim()) params.image_url = imageUrl.trim()
    if (aspectRatio) params.aspect_ratio = aspectRatio
    if (scaleFactor) params.scale_factor = scaleFactor
    if (durationSec) params.duration_sec = durationSec
    return params
  }, [prompt, negativePrompt, imageUrl, aspectRatio, scaleFactor, durationSec])

  // Requote when post-purchase return is active (Queue 3 Funnel)
  useEffect(() => {
    if (isPostPurchaseReturn && currentTool) {
      const params = buildInputParams()
      quoteStudioJob({
        tool_id: currentTool.id,
        template_id: activeTemplateId || undefined,
        input_params: params,
      })
        .then((q) => {
          if (q?.calculated_credits) {
            setFreshQuoteCredits(q.calculated_credits)
          }
        })
        .catch(() => {
          setFreshQuoteCredits(currentTool.credit_cost)
        })
    }
  }, [isPostPurchaseReturn, currentTool, activeTemplateId, buildInputParams])

  // Save pending state for top-up redirect (stripping base64 media to prevent storage leak)
  const handlePreserveState = () => {
    const rawParams = buildInputParams()
    const sanitizedParams: Record<string, unknown> = {}
    for (const [key, val] of Object.entries(rawParams)) {
      if (typeof val === 'string' && val.startsWith('data:')) {
        // Do not persist raw base64 media in localStorage
        continue
      }
      sanitizedParams[key] = val
    }

    const pendingJob: SavedPendingJob = {
      tool_id: currentTool.id,
      template_id: activeTemplateId || undefined,
      input_params: sanitizedParams,
      timestamp: Date.now(),
    }
    localStorage.setItem(STORAGE_KEY, JSON.stringify(pendingJob))
    sessionStorage.setItem('tora_studio_purchase_origin', currentTool.id)
  }

  // Handle image file upload to base64
  const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    if (!file.type.startsWith('image/')) {
      toast.error(t('กรุณาเลือกไฟล์รูปภาพเท่านั้น'))
      return
    }

    if (file.size > 10 * 1024 * 1024) {
      toast.error(t('ขนาดไฟล์ต้องไม่เกิน 10MB'))
      return
    }

    const reader = new FileReader()
    reader.addEventListener('load', () => {
      if (typeof reader.result === 'string') {
        setImageUrl(reader.result)
        toast.success(t('อัปโหลดรูปภาพสำเร็จ'))
      }
    })
    reader.readAsDataURL(file)
  }

  // Start polling an active job
  const pollJob = (jobId: string) => {
    if (pollRef.current) clearInterval(pollRef.current)

    pollRef.current = setInterval(async () => {
      try {
        const updated = await getStudioJob(jobId)
        setActiveJob(updated)

        if (
          updated.status === 'SUCCEEDED' ||
          updated.status === 'FAILED' ||
          updated.status === 'CANCELLED' ||
          updated.status === 'AMBIGUOUS_SUBMISSION'
        ) {
          if (pollRef.current) {
            clearInterval(pollRef.current)
            pollRef.current = null
          }

          if (updated.status === 'SUCCEEDED') {
            toast.success(t('สร้างผลงานสำเร็จ!'))
            localStorage.removeItem(STORAGE_KEY)
            if (onJobCompleted) onJobCompleted(updated)
          } else if (updated.status === 'FAILED') {
            toast.error(
              t('การสร้างผลงานไม่สำเร็จ: {{msg}} (ระบบคืนเครดิตเรียบร้อยแล้ว)', {
                msg: updated.error_message || 'Unknown error',
              })
            )
          }
        }
      } catch {
        // Polling error handled gracefully
      }
    }, 1500)
  }

  // Handle generation submission
  const handleGenerate = async () => {
    if (!currentTool) return

    // Require image for utility/transform tools
    const needsImage = [
      'image-upscale',
      'background-remove',
      'product-photo',
      'image-to-video',
      'image-extend',
      'object-erase',
    ].includes(currentTool.slug)

    if (needsImage && !imageUrl.trim()) {
      toast.error(t('กรุณาใส่ลิงก์รูปภาพหรืออัปโหลดรูปภาพต้นฉบับ'))
      return
    }

    if (!needsImage && !prompt.trim()) {
      toast.error(t('กรุณาระบุคำสั่ง Prompt เพื่อสร้างผลงาน'))
      return
    }

    // Require ownership & rights confirmation for Object Cleanup (Section 30)
    if (
      (currentTool.slug === 'object-erase' || currentTool.id === 'object-erase') &&
      !confirmedRights
    ) {
      toast.error(t('กรุณายืนยันสิทธิ์ในการแก้ไขรูปภาพก่อนเริ่มดำเนินการ'))
      return
    }

    setIsSubmitting(true)
    try {
      const payload = {
        tool_id: currentTool.id,
        template_id: activeTemplateId || undefined,
        provider: providerOverride === 'auto' ? undefined : providerOverride,
        input_params: buildInputParams(),
      }

      const job = await createStudioJob(payload)
      setActiveJob(job)
      toast.success(t('ส่งคำสั่งประมวลผลสำเร็จ กำลังเริ่มสร้างผลงาน'))
      pollJob(job.id)
    } catch (error: unknown) {
      if (error instanceof InsufficientCreditException) {
        setInsufficientData(error.data)
        setInsufficientModalOpen(true)
      } else {
        const msg = error instanceof Error ? error.message : String(error)
        toast.error(msg || t('เกิดข้อผิดพลาดในการส่งคำสั่ง'))
      }
    } finally {
      setIsSubmitting(false)
    }
  }

  // Explicit user confirmation after purchasing credits (Queue 3 - strictly non-auto)
  const handleConfirmAfterPurchase = () => {
    const toolId = currentTool?.id || 'studio'
    const credits = freshQuoteCredits || currentTool?.credit_cost || 10
    trackStudioGenerationAfterPurchase(toolId, credits)
    recordStudioAttribution('generation_after_purchase', toolId, credits)
    sessionStorage.removeItem('tora_studio_purchase_origin')
    sessionStorage.removeItem('tora_studio_purchase_needed')
    setIsPostPurchaseReturn(false)
    handleGenerate()
  }

  // Cancel job
  const handleCancel = async () => {
    if (!activeJob) return
    try {
      const cancelled = await cancelStudioJob(activeJob.id)
      setActiveJob(cancelled)
      toast.info(t('ยกเลิกงานเรียบร้อยแล้ว'))
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : t('ไม่สามารถยกเลิกงานได้'))
    }
  }

  // Parse result media URL
  const resultUrl = useMemo(() => {
    if (!activeJob?.output_data) return null
    try {
      const data = JSON.parse(activeJob.output_data)
      return data.output_url || data.image_url || data.video_url || null
    } catch {
      return null
    }
  }, [activeJob?.output_data])

  const isVideoTool = currentTool?.category === 'video' || currentTool?.slug.includes('video')

  const renderBadgeVariant = (status: string) => {
    if (status === 'SUCCEEDED') return 'default'
    if (status === 'FAILED') return 'destructive'
    return 'secondary'
  }

  return (
    <div className='space-y-6'>
      {/* Tool Header & Selector */}
      <div className='flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between'>
        <div className='flex items-center gap-3'>
          <div className='flex size-11 items-center justify-center rounded-xl bg-primary/10 text-primary'>
            {isVideoTool ? <Video className='size-5' /> : <Wand2 className='size-5' />}
          </div>
          <div>
            <div className='flex items-center gap-2'>
              <h2 className='text-lg font-bold text-foreground'>
                {currentTool?.name_th || currentTool?.name}
              </h2>
              {currentTool?.badge && (
                <Badge variant='secondary' className='text-[10px]'>
                  {currentTool.badge}
                </Badge>
              )}
            </div>
            <p className='text-xs text-muted-foreground'>
              {currentTool?.description_th || currentTool?.description}
            </p>
          </div>
        </div>

        {/* Tool Switcher & Testing Provider Selector */}
        <div className='flex items-center gap-2'>
          <NativeSelect
            value={currentTool?.slug}
            onChange={(e) => onToolChange(e.target.value)}
            className='h-8 text-xs'
          >
            {tools
              .filter((tool) => tool.status !== 'disabled')
              .map((tool) => (
                <option key={tool.id} value={tool.slug}>
                  {tool.name_th || tool.name} ({tool.credit_cost} Cr)
                </option>
              ))}
          </NativeSelect>

          {/* Provider Route Selector (Operator/Mock testing) */}
          <NativeSelect
            value={providerOverride}
            onChange={(e) => setProviderOverride(e.target.value as 'fal' | 'mock' | 'auto')}
            className='h-8 text-xs'
            title={t('เลือก Provider สำหรับการประมวลผล')}
          >
            <option value='auto'>{t('Auto Provider')}</option>
            <option value='fal'>{t('fal.ai (Real)')}</option>
            <option value='mock'>{t('Mock (Deterministic)')}</option>
          </NativeSelect>
        </div>
      </div>

      {/* Post-Purchase Return Funnel Banner (Queue 3) */}
      {isPostPurchaseReturn && (
        <div className='flex flex-col gap-3 rounded-2xl border border-emerald-500/30 bg-emerald-500/10 p-4 text-emerald-950 dark:text-emerald-100 sm:flex-row sm:items-center sm:justify-between'>
          <div className='flex items-center gap-3'>
            <div className='flex size-9 shrink-0 items-center justify-center rounded-xl bg-emerald-500/20 text-emerald-600 dark:text-emerald-400'>
              <Sparkles className='size-5' />
            </div>
            <div className='text-xs'>
              <p className='font-semibold text-emerald-700 dark:text-emerald-300'>
                {t('เติมเครดิตเรียบร้อยแล้ว! ข้อมูลและพารามิเตอร์ของคุณได้รับการกู้คืน')}
              </p>
              <p className='text-muted-foreground'>
                {t('อัตราบริการคำนวณใหม่:')}{' '}
                <span className='font-bold text-foreground'>
                  {freshQuoteCredits || currentTool?.credit_cost} {t('Credits')}
                </span>{' '}
                — {t('กรุณาตรวจสอบและกดยืนยันเพื่อเริ่มประมวลผล')}
              </p>
            </div>
          </div>
          <Button
            onClick={handleConfirmAfterPurchase}
            disabled={isSubmitting}
            size='sm'
            className='bg-emerald-600 hover:bg-emerald-500 text-white font-medium shrink-0 shadow-sm'
          >
            <CheckCircle2 className='mr-1.5 size-4' />
            {t('ยืนยันสร้างผลงาน')}
          </Button>
        </div>
      )}

      {/* Templates Pills */}
      {toolTemplates.length > 0 && (
        <div className='space-y-1.5'>
          <div className='text-[11px] font-medium text-muted-foreground'>
            {t('แม่แบบสำเร็จรูปด่วน (Quick Presets):')}
          </div>
          <div className='flex flex-wrap gap-1.5'>
            {toolTemplates.map((tpl) => (
              <Button
                key={tpl.id}
                size='sm'
                variant={activeTemplateId === tpl.id ? 'default' : 'outline'}
                onClick={() => applyTemplate(tpl)}
                className='h-7 text-xs font-normal'
              >
                <Sparkles className='mr-1 size-3' />
                {tpl.title_th || tpl.title}
              </Button>
            ))}
          </div>
        </div>
      )}

      {/* Main Studio Workspace: Form (Left) & Output (Right) */}
      <div className='grid grid-cols-1 gap-6 lg:grid-cols-12'>
        {/* Left Column: Input Form (7 cols) */}
        <div className='space-y-5 lg:col-span-7'>
          <Card className='border-border/60 shadow-xs'>
            <CardHeader className='pb-3'>
              <CardTitle className='text-sm font-semibold'>
                {t('พารามิเตอร์การสร้างสรรค์')}
              </CardTitle>
            </CardHeader>

            <CardContent className='space-y-4 text-xs'>
              {/* Prompt Input (if tool uses prompt) */}
              {currentTool?.slug !== 'background-remove' &&
                currentTool?.slug !== 'image-upscale' && (
                  <div className='space-y-1.5'>
                    <div className='flex items-center justify-between'>
                      <Label className='text-xs font-medium'>
                        {t('คำสั่งสร้างสรรค์ (Prompt)')}
                      </Label>
                      <span className='text-[10px] text-muted-foreground'>
                        {prompt.length} {t('ตัวอักษร')}
                      </span>
                    </div>
                    <Textarea
                      placeholder={
                        currentTool?.slug === 'product-photo'
                          ? t('เช่น: วางบนแท่นหินอ่อนสีขาว ท่ามกลางแสงแดดยามเช้า นุ่มนวล สมจริงระดับภาพถ่ายสตูดิโอ')
                          : t('อธิบายภาพหรือวิดีโอที่คุณต้องการสร้างอย่างละเอียด...')
                      }
                      value={prompt}
                      onChange={(e) => setPrompt(e.target.value)}
                      rows={4}
                      className='text-xs resize-none'
                    />
                  </div>
                )}

              {/* Negative Prompt Input for image-generate */}
              {currentTool?.slug === 'image-generate' && (
                <div className='space-y-1.5'>
                  <Label className='text-xs font-medium'>
                    {t('สิ่งที่ไม่ต้องการในภาพ (Negative Prompt)')}
                  </Label>
                  <Input
                    placeholder={t('เช่น: blurry, distorted, low quality, artifacts')}
                    value={negativePrompt}
                    onChange={(e) => setNegativePrompt(e.target.value)}
                    className='h-8 text-xs'
                  />
                </div>
              )}

              {/* Source Image Upload / URL (if tool needs input image) */}
              {[
                'image-upscale',
                'background-remove',
                'product-photo',
                'image-to-video',
                'image-extend',
                'object-erase',
              ].includes(currentTool?.slug || '') && (
                <div className='space-y-2'>
                  <Label className='text-xs font-medium'>
                    {t('รูปภาพต้นฉบับ (Source Image)')}
                  </Label>

                  <div className='flex gap-2'>
                    <Input
                      placeholder='https://example.com/image.jpg หรืออัปโหลดไฟล์'
                      value={imageUrl}
                      onChange={(e) => setImageUrl(e.target.value)}
                      className='h-8 text-xs'
                    />
                    <label className='inline-flex h-8 shrink-0 cursor-pointer items-center justify-center rounded-md border border-input bg-background px-3 text-xs font-medium hover:bg-accent'>
                      <Upload className='mr-1.5 size-3.5' />
                      {t('อัปโหลด')}
                      <input
                        type='file'
                        accept='image/*'
                        onChange={handleFileUpload}
                        className='hidden'
                      />
                    </label>
                  </div>

                  {imageUrl && (
                    <div className='relative mt-2 size-24 overflow-hidden rounded-md border bg-muted'>
                      <img
                        src={imageUrl}
                        alt='Source preview'
                        className='size-full object-cover'
                      />
                      <button
                        type='button'
                        onClick={() => setImageUrl('')}
                        className='absolute right-1 top-1 rounded-full bg-black/70 p-1 text-white hover:bg-black'
                      >
                        ×
                      </button>
                    </div>
                  )}
                </div>
              )}

              {/* Aspect Ratio Selector (for image generation / video) */}
              {['image-generate', 'image-to-video', 'product-photo'].includes(
                currentTool?.slug || ''
              ) && (
                <div className='space-y-1.5'>
                  <Label className='text-xs font-medium'>
                    {t('สัดส่วนภาพ (Aspect Ratio)')}
                  </Label>
                  <div className='grid grid-cols-4 gap-1.5'>
                    {[
                      { id: '1:1', label: '1:1 (จัตุรัส)' },
                      { id: '9:16', label: '9:16 (TikTok/Reel)' },
                      { id: '16:9', label: '16:9 (แนวนอน)' },
                      { id: '4:5', label: '4:5 (IG Feed)' },
                    ].map((ratio) => (
                      <Button
                        key={ratio.id}
                        type='button'
                        size='sm'
                        variant={aspectRatio === ratio.id ? 'default' : 'outline'}
                        onClick={() => setAspectRatio(ratio.id)}
                        className='h-8 text-[11px]'
                      >
                        {ratio.label}
                      </Button>
                    ))}
                  </div>
                </div>
              )}

              {/* Upscaler Scale Factor */}
              {currentTool?.slug === 'image-upscale' && (
                <div className='space-y-1.5'>
                  <Label className='text-xs font-medium'>
                    {t('อัตราขยายความละเอียด')}
                  </Label>
                  <div className='grid grid-cols-2 gap-2'>
                    <Button
                      type='button'
                      size='sm'
                      variant={scaleFactor === 2 ? 'default' : 'outline'}
                      onClick={() => setScaleFactor(2)}
                      className='h-8 text-xs'
                    >
                      {t('ขยาย 2 เท่า (2x HD)')}
                    </Button>
                    <Button
                      type='button'
                      size='sm'
                      variant={scaleFactor === 4 ? 'default' : 'outline'}
                      onClick={() => setScaleFactor(4)}
                      className='h-8 text-xs'
                    >
                      {t('ขยาย 4 เท่า (4x Ultra HD)')}
                    </Button>
                  </div>
                </div>
              )}

              {/* Video Duration */}
              {isVideoTool && (
                <div className='space-y-1.5'>
                  <Label className='text-xs font-medium'>
                    {t('ความยาววิดีโอ')}
                  </Label>
                  <div className='grid grid-cols-2 gap-2'>
                    <Button
                      type='button'
                      size='sm'
                      variant={durationSec === 5 ? 'default' : 'outline'}
                      onClick={() => setDurationSec(5)}
                      className='h-8 text-xs'
                    >
                      5 {t('วินาที (5s Reel)')}
                    </Button>
                    <Button
                      type='button'
                      size='sm'
                      variant={durationSec === 10 ? 'default' : 'outline'}
                      onClick={() => setDurationSec(10)}
                      className='h-8 text-xs'
                    >
                      10 {t('วินาที (10s Clip)')}
                    </Button>
                  </div>
                </div>
              )}

              {/* Object Cleanup Ownership & Rights Confirmation */}
              {(currentTool?.slug === 'object-erase' || currentTool?.id === 'object-erase') && (
                <div className='flex items-start gap-2.5 rounded-lg border border-border/70 bg-muted/30 p-3 text-xs'>
                  <input
                    type='checkbox'
                    id='confirmed-rights-checkbox'
                    checked={confirmedRights}
                    onChange={(e) => setConfirmedRights(e.target.checked)}
                    className='mt-0.5 size-4 rounded border-border text-primary focus:ring-primary'
                  />
                  <Label
                    htmlFor='confirmed-rights-checkbox'
                    className='cursor-pointer text-[11px] leading-relaxed text-muted-foreground'
                  >
                    {t(
                      'ฉันยืนยันว่าเป็นเจ้าของหรือได้รับสิทธิ์ในการแก้ไขรูปภาพนี้ และยอมรับข้อกำหนดการใช้งาน (Object Cleanup & Inpainting)'
                    )}
                  </Label>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Pricing & Execution Bar */}
          <div className='rounded-xl border border-border/60 bg-card p-4 shadow-xs'>
            <div className='flex items-center justify-between'>
              <div className='space-y-0.5'>
                <div className='flex items-center gap-2'>
                  <Coins className='size-4 text-primary' />
                  <span className='text-xs font-semibold text-foreground'>
                    {t('ค่าบริการ')}: {currentTool?.credit_cost} Tora Credits
                  </span>
                  <span className='text-[10px] text-muted-foreground'>
                    (~{(currentTool?.credit_cost || 0) * 0.07} THB)
                  </span>
                </div>
                <div className='text-[11px] text-muted-foreground'>
                  {t('เครดิตคงเหลือของคุณ')}:{' '}
                  <span
                    className={
                      hasEnoughCredits
                        ? 'font-medium text-foreground'
                        : 'font-medium text-rose-500'
                    }
                  >
                    {userCredits} Credits
                  </span>
                </div>
              </div>

              <div className='flex items-center gap-2'>
                {activeJob &&
                  (activeJob.status === 'PROCESSING' ||
                    activeJob.status === 'SUBMITTING') && (
                    <Button
                      variant='outline'
                      size='sm'
                      onClick={handleCancel}
                      className='h-9 text-xs'
                    >
                      {t('ยกเลิก')}
                    </Button>
                  )}

                <Button
                  onClick={handleGenerate}
                  disabled={isSubmitting || activeJob?.status === 'PROCESSING'}
                  className='h-9 font-medium'
                >
                  {isSubmitting ? (
                    <>
                      <Loader2 className='mr-1.5 size-3.5 animate-spin' />
                      {t('กำลังส่งคำสั่ง...')}
                    </>
                  ) : (
                    <>
                      <Sparkles className='mr-1.5 size-3.5' />
                      {t('สร้างผลงาน ({{cost}} Cr)', {
                        cost: currentTool?.credit_cost,
                      })}
                    </>
                  )}
                </Button>
              </div>
            </div>
          </div>
        </div>

        {/* Right Column: Live Output & Monitor (5 cols) */}
        <div className='space-y-5 lg:col-span-5'>
          <Card className='flex min-h-[400px] flex-col justify-between border-border/60 shadow-xs'>
            <CardHeader className='pb-3'>
              <div className='flex items-center justify-between'>
                <CardTitle className='text-sm font-semibold'>
                  {t('ผลงานที่ได้ (Generated Media)')}
                </CardTitle>
                {activeJob && (
                  <Badge
                    variant={renderBadgeVariant(activeJob.status)}
                    className='text-[10px]'
                  >
                    {activeJob.status}
                  </Badge>
                )}
              </div>
            </CardHeader>

            <CardContent className='flex flex-1 flex-col items-center justify-center p-4'>
              {/* State 1: Active Processing */}
              {activeJob &&
                (activeJob.status === 'RESERVED' ||
                  activeJob.status === 'SUBMITTING' ||
                  activeJob.status === 'PROCESSING') && (
                  <div className='w-full max-w-xs space-y-4 text-center'>
                    <div className='relative mx-auto flex size-16 items-center justify-center rounded-2xl bg-primary/10 text-primary'>
                      <Loader2 className='size-8 animate-spin' />
                    </div>

                    <div className='space-y-1'>
                      <div className='text-xs font-semibold text-foreground'>
                        {activeJob.status === 'RESERVED' && t('จองเครดิตเรียบร้อย')}
                        {activeJob.status === 'SUBMITTING' &&
                          t('กำลังเชื่อมต่อไปยัง AI Engine')}
                        {activeJob.status === 'PROCESSING' &&
                          t('AI กำลังสร้างสรรค์ผลงาน...')}
                      </div>
                      <div className='text-[11px] text-muted-foreground flex items-center justify-center gap-1.5'>
                        <Clock className='size-3' />
                        {t('เวลาที่ใช้ไป')}: {elapsedSec}s / ~
                        {currentTool?.estimated_sec}s
                      </div>
                    </div>

                    <Progress
                      value={Math.min(
                        95,
                        Math.round(
                          (elapsedSec / (currentTool?.estimated_sec || 10)) * 100
                        )
                      )}
                      className='h-1.5'
                    />

                    <div className='text-[10px] text-muted-foreground'>
                      {t('ระบบล็อคเครดิตแบบ Atomic ไม่คิดค่าบริการซ้ำซ้อน')}
                    </div>
                  </div>
                )}

              {/* State 2: Success Media Output */}
              {activeJob?.status === 'SUCCEEDED' && resultUrl && (
                <div className='space-y-3 w-full'>
                  <div className='relative overflow-hidden rounded-xl border bg-black/5'>
                    {isVideoTool ? (
                      <video
                        src={resultUrl}
                        controls
                        autoPlay
                        loop
                        className='w-full max-h-[380px] object-contain rounded-xl'
                      />
                    ) : (
                      <img
                        src={resultUrl}
                        alt='Generated Output'
                        className='w-full max-h-[380px] object-contain rounded-xl'
                      />
                    )}
                  </div>

                  <div className='flex items-center justify-between text-[11px] text-muted-foreground px-1'>
                    <span>
                      {t('ใช้เวลา')}: {(activeJob.duration_ms || 0) / 1000}s
                    </span>
                    <span>
                      {t('หักเครดิต')}: {activeJob.credit_charged} Cr
                    </span>
                  </div>

                  {/* Actions Bar */}
                  <div className='flex items-center gap-2 pt-1'>
                    <Button
                      size='sm'
                      className='flex-1 h-8 text-xs font-medium'
                      onClick={() => window.open(resultUrl, '_blank')}
                    >
                      <Download className='mr-1.5 size-3.5' />
                      {t('ดาวน์โหลด')}
                    </Button>
                    <Button
                      size='sm'
                      variant='outline'
                      className='h-8 text-xs'
                      onClick={() => {
                        navigator.clipboard.writeText(resultUrl)
                        toast.success(t('คัดลอกลิงก์ผลงานแล้ว'))
                      }}
                    >
                      <Copy className='size-3.5' />
                    </Button>
                    <Button
                      size='sm'
                      variant='outline'
                      className='h-8 text-xs'
                      title={t('Remix ผลงานนี้อีกครั้ง')}
                      onClick={() => handleGenerate()}
                    >
                      <RotateCcw className='mr-1.5 size-3.5' />
                      {t('Remix')}
                    </Button>
                  </div>
                </div>
              )}

              {/* State 3: Failure State with Refund Proof */}
              {activeJob?.status === 'FAILED' && (
                <div className='space-y-3 text-center max-w-xs'>
                  <div className='mx-auto flex size-12 items-center justify-center rounded-full bg-rose-500/10 text-rose-500'>
                    <AlertCircle className='size-6' />
                  </div>
                  <div className='space-y-1'>
                    <div className='text-xs font-semibold text-foreground'>
                      {t('การสร้างผลงานไม่สำเร็จ')}
                    </div>
                    <p className='text-[11px] text-muted-foreground'>
                      {activeJob.error_message || t('เกิดข้อผิดพลาดจาก Provider')}
                    </p>
                  </div>
                  <div className='rounded-lg bg-emerald-500/10 p-2.5 text-[10px] text-emerald-600 dark:text-emerald-400'>
                    <CheckCircle2 className='inline mr-1 size-3' />
                    {t('เครดิตของคุณได้รับการคืนเข้ากระเป๋าเรียบร้อยแล้ว (100% Refund)')}
                  </div>
                </div>
              )}

              {/* State 4: Initial Empty State */}
              {!activeJob && (
                <div className='space-y-3 text-center text-muted-foreground max-w-xs'>
                  <div className='mx-auto flex size-12 items-center justify-center rounded-2xl bg-muted'>
                    <ImageIcon className='size-6' />
                  </div>
                  <div>
                    <div className='text-xs font-medium text-foreground'>
                      {t('พร้อมสำหรับการสร้างสรรค์')}
                    </div>
                    <p className='text-[11px] mt-0.5 text-muted-foreground'>
                      {t('ระบุข้อมูลด้านซ้ายแล้วกดปุ่มสร้างผลงาน ผลลัพธ์จะปรากฏที่นี่')}
                    </p>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Insufficient Credit Modal */}
      <InsufficientCreditModal
        open={insufficientModalOpen}
        onOpenChange={setInsufficientModalOpen}
        data={insufficientData}
        onPreserveStateAndTopUp={handlePreserveState}
      />
    </div>
  )
}
