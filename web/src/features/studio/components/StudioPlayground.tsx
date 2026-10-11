import {
  AlertCircle,
  CheckCircle2,
  Clock,
  Coins,
  Copy,
  Download,
  Eye,
  ImageIcon,
  Layers,
  Loader2,
  RefreshCw,
  RotateCcw,
  ShoppingBag,
  Sparkles,
  SplitSquareVertical,
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
  trackStudioGenerateClick,
  trackStudioGenerationAfterPurchase,
  trackStudioPurchaseReturn,
  trackStudioRepeatGeneration,
  trackStudioToolVisit,
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

  // Product Studio E-Commerce states (Queue 4)
  const [marketplace, setMarketplace] = useState<'shopee' | 'lazada' | 'instagram' | 'story' | 'tiktok'>('shopee')
  const [packSize, setPackSize] = useState<1 | 4>(1)
  const [autoRemoveBg, setAutoRemoveBg] = useState(true)
  const [referenceImageUrl, setReferenceImageUrl] = useState('')
  const [activeVariantIndex, setActiveVariantIndex] = useState(0)
  const [showBeforeAfter, setShowBeforeAfter] = useState(false)
  const [generationCount, setGenerationCount] = useState(0)

  // Video Foundation states (Queue 5)
  const [videoResolution, setVideoResolution] = useState<'720p' | '1080p'>('720p')
  const [videoQuality, setVideoQuality] = useState<'standard' | 'high'>('standard')
  const [videoAudio, setVideoAudio] = useState(false)

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

  // Calculate user credits & dynamic pack quote (Queue 4 & 5)
  const isProductPhoto = currentTool?.slug === 'product-photo' || currentTool?.id === 'product-photo'
  const isVideoTool = currentTool?.category === 'video' || currentTool?.slug.includes('video') || currentTool?.id === 'image-to-video'
  const effectiveBaseCost = useMemo(() => {
    if (!currentTool) return 10
    if (isProductPhoto && packSize === 4) {
      return currentTool.credit_cost * 4
    }
    if (isVideoTool) {
      let cost = currentTool.credit_cost
      if (videoResolution === '1080p') cost = Math.ceil(cost * 1.25)
      if (videoQuality === 'high') cost = Math.ceil(cost * 1.20)
      if (videoAudio) cost += 15
      return cost
    }
    return currentTool.credit_cost
  }, [currentTool, isProductPhoto, packSize, isVideoTool, videoResolution, videoQuality, videoAudio])

  const userCredits = Math.floor((auth.user?.quota || 0) / 1000)
  const requiredCredits = freshQuoteCredits || effectiveBaseCost
  const hasEnoughCredits = userCredits >= requiredCredits

  // Track tool visit (Queue 4)
  useEffect(() => {
    if (currentTool?.id) {
      trackStudioToolVisit(currentTool.id)
      recordStudioAttribution('tool_visit', currentTool.id)
    }
  }, [currentTool?.id])

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

    // Product Studio parameters (Queue 4)
    if (isProductPhoto) {
      params.marketplace = marketplace
      params.pack_size = packSize
      params.num_outputs = packSize
      params.auto_remove_bg = autoRemoveBg
      if (referenceImageUrl.trim()) params.reference_image_url = referenceImageUrl.trim()
    }

    // Video Foundation parameters (Queue 5)
    if (isVideoTool) {
      params.duration = durationSec
      params.duration_sec = durationSec
      params.resolution = videoResolution
      params.quality = videoQuality
      params.audio = videoAudio
    }

    return params
  }, [
    prompt,
    negativePrompt,
    imageUrl,
    aspectRatio,
    scaleFactor,
    durationSec,
    isProductPhoto,
    marketplace,
    packSize,
    autoRemoveBg,
    referenceImageUrl,
    isVideoTool,
    videoResolution,
    videoQuality,
    videoAudio,
  ])

  // Requote on parameter change (for pack_size, etc.) or post-purchase return
  useEffect(() => {
    if (currentTool) {
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
          setFreshQuoteCredits(effectiveBaseCost)
        })
    }
  }, [
    isPostPurchaseReturn,
    currentTool,
    activeTemplateId,
    packSize,
    buildInputParams,
    effectiveBaseCost,
  ])

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

  // Handle media file upload through asset pipeline (Queue 5 Asset Pipeline)
  const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    if (!file.type.startsWith('image/')) {
      toast.error(t('กรุณาเลือกไฟล์รูปภาพเท่านั้น'))
      return
    }

    if (file.size > 15 * 1024 * 1024) {
      toast.error(t('ขนาดไฟล์ต้องไม่เกิน 15MB'))
      return
    }

    // Attempt direct upload via /api/v1/studio/upload to avoid base64 JSON
    try {
      const formData = new FormData()
      formData.append('file', file)
      const token = localStorage.getItem('token') || ''
      const res = await fetch('/api/v1/studio/upload', {
        method: 'POST',
        headers: token ? { Authorization: `Bearer ${token}` } : {},
        body: formData,
      })
      if (res.ok) {
        const json = await res.json()
        if (json.success && json.data?.url) {
          setImageUrl(json.data.url)
          toast.success(t('อัปโหลดรูปภาพเข้าสู่ระบบ Asset สำเร็จ'))
          return
        }
      }
    } catch {
      // Ignore and fallback if not video tool
    }

    if (isVideoTool) {
      toast.error(t('การสร้างวิดีโอต้องใช้อ็อบเจกต์ URL จากการอัปโหลด กรุณาเข้าสู่ระบบก่อนอัปโหลดภาพ'))
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
      const creditsToCharge = requiredCredits
      trackStudioGenerateClick(currentTool.id, creditsToCharge, packSize)
      recordStudioAttribution('generate_click', currentTool.id, creditsToCharge)

      if (generationCount > 0) {
        trackStudioRepeatGeneration(currentTool.id, creditsToCharge, generationCount + 1)
        recordStudioAttribution('repeat_generation', currentTool.id, creditsToCharge)
      }
      setGenerationCount((prev) => prev + 1)

      const payload = {
        tool_id: currentTool.id,
        template_id: activeTemplateId || undefined,
        provider: providerOverride === 'auto' ? undefined : providerOverride,
        input_params: buildInputParams(),
      }

      const job = await createStudioJob(payload)
      setActiveJob(job)
      setActiveVariantIndex(0)
      setShowBeforeAfter(false)
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
    const credits = freshQuoteCredits || effectiveBaseCost
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

  // Parse result media URL and multi-variants (Queue 4)
  const variants: string[] = useMemo(() => {
    if (!activeJob?.output_data) return []
    try {
      const data = JSON.parse(activeJob.output_data)
      if (Array.isArray(data.output_urls) && data.output_urls.length > 0) return data.output_urls
      if (Array.isArray(data.variants) && data.variants.length > 0) return data.variants
      if (data.output_url) return [data.output_url]
      if (data.image_url) return [data.image_url]
      if (data.video_url) return [data.video_url]
      return []
    } catch {
      return []
    }
  }, [activeJob?.output_data])

  const resultUrl = useMemo(() => {
    if (variants.length > 0) {
      return variants[activeVariantIndex] || variants[0]
    }
    if (!activeJob?.output_data) return null
    try {
      const data = JSON.parse(activeJob.output_data)
      return data.output_url || data.image_url || data.video_url || null
    } catch {
      return null
    }
  }, [variants, activeVariantIndex, activeJob?.output_data])

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

              {/* Product Studio Marketplace Presets & Aspects (Queue 4) */}
              {isProductPhoto ? (
                <div className='space-y-3 rounded-xl border border-primary/20 bg-primary/5 p-3.5'>
                  <div className='flex items-center justify-between'>
                    <div className='flex items-center gap-1.5'>
                      <ShoppingBag className='size-4 text-primary' />
                      <Label className='text-xs font-semibold text-foreground'>
                        {t('สัดส่วนตาม Marketplace (E-Commerce Presets)')}
                      </Label>
                    </div>
                    <Badge variant='outline' className='text-[10px] text-primary border-primary/30'>
                      {marketplace.toUpperCase()} ({aspectRatio})
                    </Badge>
                  </div>

                  <div className='grid grid-cols-2 sm:grid-cols-5 gap-1.5'>
                    {[
                      { id: 'shopee', label: 'Shopee (1:1)', ratio: '1:1' },
                      { id: 'lazada', label: 'Lazada (1:1)', ratio: '1:1' },
                      { id: 'instagram', label: 'Instagram (4:5)', ratio: '4:5' },
                      { id: 'story', label: 'Story (9:16)', ratio: '9:16' },
                      { id: 'tiktok', label: 'TikTok (9:16)', ratio: '9:16' },
                    ].map((m) => (
                      <Button
                        key={m.id}
                        type='button'
                        size='sm'
                        variant={marketplace === m.id ? 'default' : 'outline'}
                        onClick={() => {
                          setMarketplace(m.id as any)
                          setAspectRatio(m.ratio)
                        }}
                        className='h-8 text-[11px] px-2'
                      >
                        {m.label}
                      </Button>
                    ))}
                  </div>

                  {/* 8 Visual Templates Palette for Product Studio */}
                  <div className='space-y-1.5 pt-1'>
                    <Label className='text-[11px] font-medium text-muted-foreground'>
                      {t('สไตล์ฉากหลังยอดนิยม (Visual Templates):')}
                    </Label>
                    <div className='grid grid-cols-2 sm:grid-cols-4 gap-1.5'>
                      {[
                        { id: 'tpl-prod-white-studio', name: 'White Studio', prompt: 'Clean seamless white studio background, commercial advertising lighting, soft ambient reflection, high-end product showcase, 8k resolution' },
                        { id: 'tpl-prod-luxury-black', name: 'Luxury Black', prompt: 'Dark luxury minimalist stone background, dramatic moody rim light, premium cosmetic advertising, high contrast, elegant aesthetic' },
                        { id: 'tpl-prod-minimal-beige', name: 'Minimal Beige', prompt: 'Warm beige linen background, wooden pedestal, soft natural morning sunlight casting organic leaf shadows, aesthetic lifestyle product shot' },
                        { id: 'tpl-prod-kitchen', name: 'Kitchen', prompt: 'Modern bright marble kitchen countertop, blurry warm kitchen background, daylight, lifestyle culinary photography' },
                        { id: 'tpl-prod-food', name: 'Food & Dining', prompt: 'Rustic wooden dining table, warm cafe ambiance, appetizing soft daylight, gourmet culinary background bokeh, commercial food presentation' },
                        { id: 'tpl-prod-cosmetics', name: 'Cosmetics', prompt: 'Minimalist pastel acrylic pedestal, soft gradient backdrop, subtle water droplet reflections, clean beauty commercial lighting, delicate aesthetic' },
                        { id: 'tpl-prod-fashion', name: 'Fashion', prompt: 'Contemporary high-fashion concrete boutique backdrop, architectural soft shadows, sleek gallery aesthetic, Vogue style editorial product lighting' },
                        { id: 'tpl-prod-outdoor', name: 'Outdoor', prompt: 'Natural outdoor setting on a smooth wet pebble stone, lush green foliage background bokeh, golden hour sunlight, organic aesthetic' },
                      ].map((style) => (
                        <Button
                          key={style.id}
                          type='button'
                          size='sm'
                          variant={prompt.includes(style.name) || activeTemplateId === style.id ? 'default' : 'outline'}
                          onClick={() => {
                            setActiveTemplateId(style.id)
                            setPrompt(style.prompt)
                            toast.success(t('เลือกสไตล์ {{name}} สำเร็จ', { name: style.name }))
                          }}
                          className='h-7 text-[10px] justify-start px-2 truncate'
                          title={style.name}
                        >
                          <Sparkles className='mr-1 size-3 shrink-0' />
                          <span className='truncate'>{style.name}</span>
                        </Button>
                      ))}
                    </div>
                  </div>

                  {/* Result Pack Size: Single vs 4-Pack */}
                  <div className='pt-2 space-y-1.5'>
                    <div className='flex items-center justify-between'>
                      <Label className='text-[11px] font-semibold text-foreground'>
                        {t('จำนวนรูปที่ต้องการ (Result Pack):')}
                      </Label>
                      <span className='text-[10px] text-muted-foreground'>
                        {packSize === 1 ? '1 รูปเดี่ยว (50 Cr)' : 'แพ็ก 4 รูป (200 Cr)'}
                      </span>
                    </div>
                    <div className='grid grid-cols-2 gap-2'>
                      <Button
                        type='button'
                        size='sm'
                        variant={packSize === 1 ? 'default' : 'outline'}
                        onClick={() => setPackSize(1)}
                        className='h-8 text-xs font-normal'
                      >
                        {t('1 รูปเดี่ยว (Single 50 Cr)')}
                      </Button>
                      <Button
                        type='button'
                        size='sm'
                        variant={packSize === 4 ? 'default' : 'outline'}
                        onClick={() => setPackSize(4)}
                        className='h-8 text-xs font-medium border-primary/40'
                      >
                        <Layers className='mr-1.5 size-3.5 text-primary' />
                        {t('แพ็ก 4 ภาพ (4-Pack 200 Cr)')}
                      </Button>
                    </div>
                  </div>

                  {/* Multi-reference & Auto BG Removal */}
                  <div className='pt-2 space-y-2 border-t border-border/50'>
                    <div className='flex items-center gap-2'>
                      <input
                        type='checkbox'
                        id='auto-remove-bg-check'
                        checked={autoRemoveBg}
                        onChange={(e) => setAutoRemoveBg(e.target.checked)}
                        className='size-4 rounded border-border text-primary focus:ring-primary'
                      />
                      <Label htmlFor='auto-remove-bg-check' className='cursor-pointer text-[11px] font-normal'>
                        {t('ลบพื้นหลังเดิมอัตโนมัติก่อนจัดฉาก (Auto-remove background)')}
                      </Label>
                    </div>

                    <div className='space-y-1'>
                      <Label className='text-[11px] font-normal text-muted-foreground'>
                        {t('ภาพอ้างอิงฉากหลังหรือสไตล์ (Optional Reference Background URL):')}
                      </Label>
                      <Input
                        placeholder='https://example.com/reference-bg.jpg'
                        value={referenceImageUrl}
                        onChange={(e) => setReferenceImageUrl(e.target.value)}
                        className='h-7 text-[11px]'
                      />
                    </div>

                    <p className='text-[10px] text-muted-foreground leading-relaxed bg-muted/40 p-2 rounded-lg'>
                      {t('💡 คำแนะนำสำหรับผู้ขาย: สำหรับโลโก้แบรนด์ แนะนำให้แปะทับ (Overlay) บนภาพที่ได้ เพื่อรักษาความคมชัดของฟอนต์และเครื่องหมายการค้า')}
                    </p>
                  </div>
                </div>
              ) : (
                /* Aspect Ratio Selector (for image generation / video) */
                ['image-generate', 'image-to-video'].includes(
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
                )
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

              {/* Video Controls (Queue 5: Conservative Controls) */}
              {isVideoTool && (
                <div className='space-y-3 rounded-lg border border-border/80 bg-muted/20 p-3'>
                  {/* Duration (max 5s enforced) */}
                  <div className='space-y-1.5'>
                    <div className='flex items-center justify-between'>
                      <Label className='text-xs font-medium'>
                        {t('ความยาววิดีโอ (Duration)')}
                      </Label>
                      <span className='text-[10px] text-muted-foreground'>
                        {t('สูงสุด 5 วินาที')}
                      </span>
                    </div>
                    <div className='grid grid-cols-2 gap-2'>
                      <Button
                        type='button'
                        size='sm'
                        variant={durationSec === 3 ? 'default' : 'outline'}
                        onClick={() => setDurationSec(3)}
                        className='h-8 text-xs'
                      >
                        3 {t('วินาที (3s Quick)')}
                      </Button>
                      <Button
                        type='button'
                        size='sm'
                        variant={durationSec === 5 ? 'default' : 'outline'}
                        onClick={() => setDurationSec(5)}
                        className='h-8 text-xs'
                      >
                        5 {t('วินาที (5s Standard Reel)')}
                      </Button>
                    </div>
                  </div>

                  {/* Resolution (Controlled: 720p vs 1080p) */}
                  <div className='space-y-1.5'>
                    <Label className='text-xs font-medium'>
                      {t('ความละเอียด (Resolution)')}
                    </Label>
                    <div className='grid grid-cols-2 gap-2'>
                      <Button
                        type='button'
                        size='sm'
                        variant={videoResolution === '720p' ? 'default' : 'outline'}
                        onClick={() => setVideoResolution('720p')}
                        className='h-8 text-xs'
                      >
                        720p (HD มาตรฐาน)
                      </Button>
                      <Button
                        type='button'
                        size='sm'
                        variant={videoResolution === '1080p' ? 'default' : 'outline'}
                        onClick={() => setVideoResolution('1080p')}
                        className='h-8 text-xs'
                      >
                        1080p (Full HD คมชัด)
                      </Button>
                    </div>
                  </div>

                  {/* Quality & Audio */}
                  <div className='grid grid-cols-2 gap-2 pt-1'>
                    <div className='space-y-1'>
                      <Label className='text-[11px] font-medium'>
                        {t('ระดับคุณภาพ')}
                      </Label>
                      <div className='flex gap-1'>
                        <Button
                          type='button'
                          size='sm'
                          variant={videoQuality === 'standard' ? 'default' : 'outline'}
                          onClick={() => setVideoQuality('standard')}
                          className='h-7 text-[11px] flex-1'
                        >
                          Standard
                        </Button>
                        <Button
                          type='button'
                          size='sm'
                          variant={videoQuality === 'high' ? 'default' : 'outline'}
                          onClick={() => setVideoQuality('high')}
                          className='h-7 text-[11px] flex-1'
                        >
                          High
                        </Button>
                      </div>
                    </div>
                    <div className='space-y-1'>
                      <Label className='text-[11px] font-medium'>
                        {t('เสียงประกอบ (Audio)')}
                      </Label>
                      <Button
                        type='button'
                        size='sm'
                        variant={videoAudio ? 'default' : 'outline'}
                        onClick={() => setVideoAudio(!videoAudio)}
                        className='h-7 text-[11px] w-full'
                      >
                        {videoAudio ? t('เปิดเสียง (+15 Cr)') : t('ปิดเสียง (Mute)')}
                      </Button>
                    </div>
                  </div>

                  {/* Guard Notice Callout */}
                  <div className='rounded border border-blue-500/20 bg-blue-500/5 p-2 text-[10px] text-muted-foreground'>
                    <p className='font-medium text-blue-400 mb-0.5'>ข้อกำหนดความปลอดภัยของระบบวิดีโอ</p>
                    <p>ระบบควบคุมต้นทุนวิดีโอ: จำกัดสูงสุด 5 วินาที ความละเอียด 720p/1080p และรันได้ทีละ 1 งานต่อผู้ใช้เพื่อความเสถียร</p>
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

              {/* State 2: Success Media Output (Queue 4: Product Studio Before/After & Variants) */}
              {activeJob?.status === 'SUCCEEDED' && resultUrl && (
                <div className='space-y-3 w-full'>
                  {/* Before / After View Mode Toggle (when source image exists) */}
                  {imageUrl && (
                    <div className='flex items-center justify-between pb-1'>
                      <span className='text-[11px] font-medium text-muted-foreground'>
                        {t('โหมดแสดงผล')}:
                      </span>
                      <div className='flex gap-1'>
                        <Button
                          type='button'
                          size='sm'
                          variant={!showBeforeAfter ? 'default' : 'outline'}
                          onClick={() => setShowBeforeAfter(false)}
                          className='h-6 px-2 text-[10px]'
                        >
                          <Eye className='mr-1 size-3' />
                          {t('ผลงาน AI')}
                        </Button>
                        <Button
                          type='button'
                          size='sm'
                          variant={showBeforeAfter ? 'default' : 'outline'}
                          onClick={() => setShowBeforeAfter(true)}
                          className='h-6 px-2 text-[10px]'
                        >
                          <SplitSquareVertical className='mr-1 size-3' />
                          {t('ก่อน/หลัง (Before/After)')}
                        </Button>
                      </div>
                    </div>
                  )}

                  {/* Main Display Area */}
                  {showBeforeAfter && imageUrl ? (
                    <div className='grid grid-cols-2 gap-2 rounded-xl border bg-black/5 p-2'>
                      <div className='space-y-1 text-center'>
                        <span className='text-[10px] font-semibold text-muted-foreground uppercase'>
                          {t('ภาพต้นฉบับ (Before)')}
                        </span>
                        <div className='overflow-hidden rounded-lg border bg-background/50 h-[220px] flex items-center justify-center'>
                          <img
                            src={imageUrl}
                            alt='Before source'
                            className='max-h-full max-w-full object-contain'
                          />
                        </div>
                      </div>
                      <div className='space-y-1 text-center'>
                        <span className='text-[10px] font-semibold text-primary uppercase'>
                          {t('ภาพสตูดิโอ (After)')}
                        </span>
                        <div className='overflow-hidden rounded-lg border bg-background/50 h-[220px] flex items-center justify-center'>
                          <img
                            src={resultUrl}
                            alt='After generated'
                            className='max-h-full max-w-full object-contain'
                          />
                        </div>
                      </div>
                    </div>
                  ) : (
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
                  )}

                  {/* Multi-Variant Browsing Thumbnails (Queue 4: 4-Pack) */}
                  {variants.length > 1 && (
                    <div className='space-y-1.5 pt-1'>
                      <div className='flex items-center justify-between'>
                        <span className='text-[11px] font-semibold text-foreground'>
                          {t('เลือกดูผลงาน (4-Pack Variants):')}
                        </span>
                        <span className='text-[10px] text-muted-foreground'>
                          {activeVariantIndex + 1} / {variants.length}
                        </span>
                      </div>
                      <div className='flex gap-2 overflow-x-auto pb-1'>
                        {variants.map((vUrl, idx) => (
                          <button
                            key={idx}
                            type='button'
                            onClick={() => {
                              setActiveVariantIndex(idx)
                              setShowBeforeAfter(false)
                            }}
                            className={`relative size-14 shrink-0 rounded-lg overflow-hidden border-2 transition ${
                              activeVariantIndex === idx
                                ? 'border-primary ring-2 ring-primary/20 scale-105'
                                : 'border-border/60 opacity-70 hover:opacity-100'
                            }`}
                          >
                            <img
                              src={vUrl}
                              alt={`Variant ${idx + 1}`}
                              className='size-full object-cover'
                            />
                            <span className='absolute bottom-0 inset-x-0 bg-black/70 text-[9px] font-bold text-white text-center py-0.5'>
                              #{idx + 1}
                            </span>
                          </button>
                        ))}
                      </div>
                    </div>
                  )}

                  <div className='flex items-center justify-between text-[11px] text-muted-foreground px-1'>
                    <span>
                      {t('ใช้เวลา')}: {(activeJob.duration_ms || 0) / 1000}s
                    </span>
                    <span>
                      {t('หักเครดิต')}: {activeJob.credit_charged} Cr
                      {variants.length > 1 ? ` (${variants.length} ภาพ)` : ''}
                    </span>
                  </div>

                  {/* Actions Bar */}
                  <div className='flex flex-wrap items-center gap-2 pt-1'>
                    <Button
                      size='sm'
                      className='flex-1 h-8 text-xs font-medium'
                      onClick={() => window.open(resultUrl, '_blank')}
                    >
                      <Download className='mr-1.5 size-3.5' />
                      {variants.length > 1
                        ? t('ดาวน์โหลดภาพนี้ (#{{num}})', { num: activeVariantIndex + 1 })
                        : t('ดาวน์โหลด')}
                    </Button>

                    {variants.length > 1 && (
                      <Button
                        size='sm'
                        variant='outline'
                        className='h-8 text-xs font-medium'
                        title={t('ดาวน์โหลดผลงานทั้งหมด')}
                        onClick={() => {
                          variants.forEach((url) => window.open(url, '_blank'))
                          toast.success(t('เปิดหน้าต่างดาวน์โหลดทั้ง 4 ภาพเรียบร้อย'))
                        }}
                      >
                        <Layers className='mr-1.5 size-3.5 text-primary' />
                        {t('ดาวน์โหลดทั้งหมด')}
                      </Button>
                    )}

                    <Button
                      size='sm'
                      variant='outline'
                      className='h-8 text-xs'
                      onClick={() => {
                        navigator.clipboard.writeText(resultUrl)
                        toast.success(t('คัดลอกลิงก์ผลงานแล้ว'))
                      }}
                      title={t('คัดลอกลิงก์')}
                    >
                      <Copy className='size-3.5' />
                    </Button>

                    {isProductPhoto && (
                      <Button
                        size='sm'
                        variant='outline'
                        className='h-8 text-xs font-medium border-primary/30 text-primary hover:bg-primary/5'
                        title={t('เก็บรูปสินค้าไว้ แล้วเลือกฉากหลังใหม่')}
                        onClick={() => {
                          setPrompt('')
                          setActiveTemplateId(null)
                          toast.info(t('เลือกฉากหลังใหม่ทางด้านซ้ายเพื่อสร้างภาพสไตล์อื่น'))
                        }}
                      >
                        <RefreshCw className='mr-1.5 size-3.5' />
                        {t('ลองฉากหลังอื่น')}
                      </Button>
                    )}

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
