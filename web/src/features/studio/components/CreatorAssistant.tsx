import {
  AlertCircle,
  ArrowRight,
  Bot,
  CheckCircle2,
  Coins,
  Copy,
  ExternalLink,
  Film,
  Layers,
  Play,
  RotateCw,
  Sparkles,
  Upload,
  Video,
  Wand2,
} from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { Progress } from '@/components/ui/progress'
import { Spinner } from '@/components/ui/spinner'
import { Textarea } from '@/components/ui/textarea'

interface WorkflowStep {
  step_index: number
  logical_tool: string
  tool_display_name: string
  template_id?: string
  template_name?: string
  parameters: Record<string, any>
  depends_on: number[]
  quote_id: string
  estimated_credits: number
  status: 'PENDING' | 'RESERVED' | 'RUNNING' | 'SUCCEEDED' | 'FAILED' | 'SKIPPED'
  job_id?: string
  output_result?: string
  error_message?: string
}

interface WorkflowPlan {
  id: string
  goal_prompt: string
  status: 'DRAFT' | 'CONFIRMED' | 'RUNNING' | 'COMPLETED' | 'PARTIAL_FAILED' | 'FAILED' | 'CANCELLED'
  steps_json: string
  total_estimated_credits: number
  total_settled_credits: number
  requires_consent: boolean
  consent_confirmed: boolean
}

export function CreatorAssistant() {
  const [prompt, setPrompt] = useState('')
  const [imageUrl, setImageUrl] = useState('')
  const [isUploading, setIsUploading] = useState(false)
  const [isPlanning, setIsPlanning] = useState(false)
  const [isExecuting, setIsExecuting] = useState(false)
  const [isRetrying, setIsRetrying] = useState<number | null>(null)
  const [plan, setPlan] = useState<WorkflowPlan | null>(null)
  const [steps, setSteps] = useState<WorkflowStep[]>([])
  const [consentConfirmed, setConsentConfirmed] = useState(false)
  const [insufficientError, setInsufficientError] = useState<{
    required: number
    available: number
    missing: number
  } | null>(null)

  const examplePrompts = [
    'ทำรูปสินค้านี้เป็นโฆษณา Instagram แล้วทำวิดีโอ 5 วิ',
    'ลบพื้นหลัง จัดแสงแบบสตูดิโอสีขาว 1:1 สำหรับ Shopee',
    'สร้างรูปสินค้า TikTok 9:16 แล้วขยายภาพ 2x',
  ]

  const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    setIsUploading(true)
    const formData = new FormData()
    formData.append('file', file)

    try {
      const res = await fetch('/api/v1/studio/upload', {
        method: 'POST',
        body: formData,
      })
      const data = await res.json()
      if (data.success && data.data?.url) {
        setImageUrl(data.data.url)
        toast.success('อัปโหลดรูปภาพสำเร็จ')
      } else {
        toast.error(data.message || 'อัปโหลดรูปภาพไม่สำเร็จ')
      }
    } catch {
      toast.error('การอัปโหลดขัดข้อง กรุณาลองใหม่อีกครั้ง')
    } finally {
      setIsUploading(false)
    }
  }

  const handleGeneratePlan = async () => {
    if (!prompt.trim()) {
      toast.error('กรุณาระบุสิ่งที่ต้องการสร้าง')
      return
    }

    setIsPlanning(true)
    setInsufficientError(null)
    setPlan(null)
    setSteps([])

    try {
      const res = await fetch('/api/v1/studio/assistant/plan', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          prompt: prompt.trim(),
          initial_image_url: imageUrl.trim() || undefined,
        }),
      })

      const data = await res.json()
      if (data.success && data.data?.plan) {
        setPlan(data.data.plan)
        setSteps(data.data.steps || [])
        toast.success('สร้าง ToolPlan พร้อมใบเสนอราคาจากระบบเรียบร้อย')
      } else {
        toast.error(data.message || 'ไม่สามารถสร้างแผนงานได้')
      }
    } catch {
      toast.error('ระบบขัดข้อง กรุณาลองใหม่อีกครั้ง')
    } finally {
      setIsPlanning(false)
    }
  }

  const handleConfirmExecute = async () => {
    if (!plan) return
    if (plan.requires_consent && !consentConfirmed) {
      toast.error('กรุณายืนยันสิทธิ์และความยินยอมในการใช้รูปภาพบุคคลก่อนดำเนินการ')
      return
    }

    setIsExecuting(true)
    setInsufficientError(null)

    try {
      const res = await fetch(`/api/v1/studio/assistant/plan/${plan.id}/confirm`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          consent_confirmed: consentConfirmed,
        }),
      })

      const data = await res.json()
      if (res.status === 402 || data.error_code === 'INSUFFICIENT_CREDITS') {
        setInsufficientError({
          required: data.data?.required_credits || plan.total_estimated_credits,
          available: data.data?.available_credits || 0,
          missing: data.data?.missing_credits || plan.total_estimated_credits,
        })
        toast.error('ยอดเครดิตของคุณไม่เพียงพอสำหรับแผนงานนี้')
        return
      }

      if (data.success && data.data?.plan) {
        setPlan(data.data.plan)
        setSteps(data.data.steps || [])
        toast.success('ดำเนินการเวิร์กโฟลว์สำเร็จครบทุกขั้นตอน!')
      } else {
        if (data.data?.plan) {
          setPlan(data.data.plan)
          setSteps(data.data.steps || [])
        }
        toast.error(data.message || 'การดำเนินการบางขั้นตอนล้มเหลว')
      }
    } catch (err: any) {
      toast.error(err.message || 'เกิดข้อผิดพลาดในการดำเนินการ')
    } finally {
      setIsExecuting(false)
    }
  }

  const handleRetryStep = async (stepIndex: number) => {
    if (!plan) return
    setIsRetrying(stepIndex)

    try {
      const res = await fetch(`/api/v1/studio/assistant/plan/${plan.id}/retry`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ step_index: stepIndex }),
      })

      const data = await res.json()
      if (data.success && data.data?.plan) {
        setPlan(data.data.plan)
        setSteps(data.data.steps || [])
        toast.success(`ทำซ้ำขั้นตอนที่ ${stepIndex} สำเร็จ`)
      } else {
        toast.error(data.message || `ทำซ้ำขั้นตอนที่ ${stepIndex} ไม่สำเร็จ`)
      }
    } catch {
      toast.error('การเชื่อมต่อขัดข้อง')
    } finally {
      setIsRetrying(null)
    }
  }

  const handleRequote = async () => {
    if (!plan) return
    setIsPlanning(true)
    try {
      const res = await fetch(`/api/v1/studio/assistant/plan/${plan.id}/requote`, {
        method: 'POST',
      })
      const data = await res.json()
      if (data.success && data.data?.plan) {
        setPlan(data.data.plan)
        setSteps(data.data.steps || [])
        setInsufficientError(null)
        toast.success('อัปเดตราคาล่าสุดเรียบร้อย')
      }
    } catch {
      toast.error('ไม่สามารถอัปเดตราคาได้')
    } finally {
      setIsPlanning(false)
    }
  }

  return (
    <div className='mx-auto max-w-4xl space-y-6'>
      {/* Header Banner */}
      <div className='rounded-2xl border border-primary/20 bg-gradient-to-r from-primary/10 via-primary/5 to-transparent p-6'>
        <div className='flex items-start gap-4'>
          <div className='flex size-12 shrink-0 items-center justify-center rounded-xl bg-primary text-primary-foreground shadow-md'>
            <Bot className='size-6' />
          </div>
          <div className='space-y-1.5'>
            <div className='flex items-center gap-2'>
              <h2 className='text-xl font-bold tracking-tight text-foreground'>
                Creator Assistant (ผู้ช่วยครีเอเตอร์)
              </h2>
              <Badge variant='secondary' className='border-primary/20 bg-primary/10 text-primary'>
                Queue 6
              </Badge>
            </div>
            <p className='text-sm text-muted-foreground leading-relaxed'>
              เพียงพิมพ์ผลลัพธ์ที่คุณต้องการเป็นภาษาไทย ผู้ช่วยจะวางแผนการทำงาน (ToolPlan)
              และขอใบเสนอราคาจากระบบให้อัตโนมัติ โดยตัดเครดิตเฉพาะขั้นตอนที่สำเร็จจริง
            </p>
          </div>
        </div>
      </div>

      {/* Input Section */}
      <Card className='border-border/60 shadow-sm'>
        <CardHeader className='pb-3'>
          <CardTitle className='text-base font-semibold flex items-center gap-2'>
            <Wand2 className='size-4 text-primary' />
            1. บอกสิ่งที่คุณต้องการสร้าง (Describe Outcome)
          </CardTitle>
        </CardHeader>
        <CardContent className='space-y-4'>
          <div>
            <Textarea
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              placeholder='เช่น: ทำรูปสินค้านี้เป็นโฆษณา Instagram แล้วทำวิดีโอ 5 วิ'
              className='min-h-[90px] resize-none text-sm'
            />
          </div>

          {/* Quick example tags */}
          <div className='flex flex-wrap items-center gap-2 pt-1'>
            <span className='text-xs text-muted-foreground'>ตัวอย่างคำสั่ง:</span>
            {examplePrompts.map((p, i) => (
              <button
                key={i}
                type='button'
                onClick={() => setPrompt(p)}
                className='rounded-md border border-border/60 bg-muted/40 px-2 py-1 text-xs text-muted-foreground hover:bg-muted hover:text-foreground transition-colors'
              >
                {p}
              </button>
            ))}
          </div>

          {/* Image input upload */}
          <div className='rounded-xl border border-dashed border-border p-4 bg-muted/20'>
            <div className='flex flex-col sm:flex-row items-center gap-4'>
              {imageUrl ? (
                <div className='relative size-16 shrink-0 rounded-lg overflow-hidden border border-border'>
                  <img src={imageUrl} alt='Upload preview' className='size-full object-cover' />
                </div>
              ) : (
                <div className='flex size-14 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground'>
                  <Upload className='size-5' />
                </div>
              )}
              <div className='flex-1 space-y-1 text-center sm:text-left'>
                <p className='text-xs font-medium text-foreground'>
                  {imageUrl ? 'รูปภาพต้นฉบับพร้อมใช้งาน' : 'อัปโหลดรูปภาพสินค้า/วัตถุต้นฉบับ'}
                </p>
                <p className='text-[11px] text-muted-foreground'>
                  รองรับ PNG, JPG, WebP (สูงสุด 50MB) ไม่ส่งเป็น Base64
                </p>
              </div>
              <div className='flex gap-2 shrink-0'>
                <label className='cursor-pointer'>
                  <Button variant='outline' size='sm' disabled={isUploading} asChild>
                    <span>
                      {isUploading ? <Spinner className='mr-1.5 size-3.5' /> : <Upload className='mr-1.5 size-3.5' />}
                      {imageUrl ? 'เปลี่ยนรูปภาพ' : 'เลือกไฟล์รูปภาพ'}
                    </span>
                  </Button>
                  <input
                    type='file'
                    accept='image/png,image/jpeg,image/webp'
                    className='hidden'
                    onChange={handleFileUpload}
                  />
                </label>
                {imageUrl && (
                  <Button variant='ghost' size='sm' onClick={() => setImageUrl('')}>
                    ลบ
                  </Button>
                )}
              </div>
            </div>
          </div>

          <Button
            onClick={handleGeneratePlan}
            disabled={isPlanning || isUploading || !prompt.trim()}
            className='w-full font-medium'
          >
            {isPlanning ? (
              <>
                <Spinner className='mr-2 size-4' />
                กำลังวิเคราะห์ความต้องการและขอใบเสนอราคา...
              </>
            ) : (
              <>
                <Sparkles className='mr-2 size-4' />
                สร้างแผนงานและขอใบเสนอราคา (Generate ToolPlan)
              </>
            )}
          </Button>
        </CardContent>
      </Card>

      {/* Insufficient Credit Banner */}
      {insufficientError && (
        <Alert variant='destructive' className='border-destructive/30 bg-destructive/10'>
          <AlertCircle className='size-4' />
          <AlertTitle className='font-semibold'>เครดิตในกระเป๋าของคุณไม่เพียงพอ</AlertTitle>
          <AlertDescription className='space-y-3 pt-2 text-xs'>
            <div className='grid grid-cols-3 gap-2 rounded-lg bg-background/50 p-2 text-center border border-border/40'>
              <div>
                <p className='text-muted-foreground'>จำเป็นต้องใช้</p>
                <p className='font-bold text-foreground text-sm'>{insufficientError.required} Credits</p>
              </div>
              <div>
                <p className='text-muted-foreground'>คงเหลืออยู่</p>
                <p className='font-bold text-foreground text-sm'>{insufficientError.available} Credits</p>
              </div>
              <div>
                <p className='text-destructive font-medium'>ขาดอีก</p>
                <p className='font-bold text-destructive text-sm'>{insufficientError.missing} Credits</p>
              </div>
            </div>
            <div className='flex items-center gap-3 pt-1'>
              <Button size='sm' variant='default' asChild>
                <a href='/wallet'>
                  <Coins className='mr-1.5 size-3.5' />
                  เติมเครดิต (Buy Credits)
                </a>
              </Button>
              <Button size='sm' variant='outline' onClick={handleRequote}>
                <RotateCw className='mr-1.5 size-3.5' />
                ตรวจสอบยอดใหม่ (Requote)
              </Button>
            </div>
          </AlertDescription>
        </Alert>
      )}

      {/* ToolPlan Review & Execution */}
      {plan && steps.length > 0 && (
        <Card className='border-border/60 shadow-sm'>
          <CardHeader className='pb-3 flex flex-row items-center justify-between border-b border-border/40'>
            <div>
              <CardTitle className='text-base font-semibold flex items-center gap-2'>
                <Layers className='size-4 text-primary' />
                2. ตรวจสอบแผนงาน ToolPlan & ราคาจากระบบ
              </CardTitle>
              <p className='text-xs text-muted-foreground mt-0.5'>
                Plan ID: {plan.id} | สถานะ: {plan.status}
              </p>
            </div>
            <Badge variant='outline' className='font-mono text-xs'>
              รวม {plan.total_estimated_credits} Credits
            </Badge>
          </CardHeader>
          <CardContent className='space-y-4 pt-4'>
            {/* Step list */}
            <div className='space-y-3'>
              {steps.map((step, idx) => (
                <div
                  key={step.step_index}
                  className={`rounded-xl border p-4 transition-all ${
                    step.status === 'SUCCEEDED'
                      ? 'border-emerald-500/30 bg-emerald-500/5'
                      : step.status === 'FAILED'
                        ? 'border-destructive/40 bg-destructive/5'
                        : step.status === 'RUNNING'
                          ? 'border-primary/40 bg-primary/5'
                          : 'border-border/60 bg-muted/20'
                  }`}
                >
                  <div className='flex items-start justify-between gap-3'>
                    <div className='flex items-start gap-3'>
                      <div
                        className={`flex size-7 shrink-0 items-center justify-center rounded-full text-xs font-bold ${
                          step.status === 'SUCCEEDED'
                            ? 'bg-emerald-500 text-white'
                            : step.status === 'FAILED'
                              ? 'bg-destructive text-white'
                              : step.status === 'RUNNING'
                                ? 'bg-primary text-white animate-pulse'
                                : 'bg-muted text-muted-foreground'
                        }`}
                      >
                        {step.status === 'SUCCEEDED' ? (
                          <CheckCircle2 className='size-4' />
                        ) : (
                          step.step_index
                        )}
                      </div>
                      <div className='space-y-1'>
                        <div className='flex items-center gap-2 flex-wrap'>
                          <p className='text-sm font-semibold text-foreground'>
                            {step.tool_display_name}
                          </p>
                          <Badge variant='outline' className='text-[10px] font-mono'>
                            {step.logical_tool}
                          </Badge>
                        </div>
                        {step.template_name && (
                          <p className='text-xs text-muted-foreground'>
                            พรีเซ็ต: {step.template_name}
                          </p>
                        )}
                        {step.depends_on.length > 0 && (
                          <p className='text-[11px] text-muted-foreground flex items-center gap-1'>
                            <span>รับผลลัพธ์จาก:</span>
                            {step.depends_on.map((dep) => (
                              <Badge key={dep} variant='secondary' className='text-[10px] py-0 px-1.5'>
                                ขั้นตอนที่ {dep}
                              </Badge>
                            ))}
                          </p>
                        )}
                        {step.error_message && (
                          <p className='text-xs text-destructive font-medium pt-1'>
                            ข้อผิดพลาด: {step.error_message}
                          </p>
                        )}
                      </div>
                    </div>

                    <div className='flex flex-col items-end gap-1.5 shrink-0'>
                      <span className='font-mono text-xs font-semibold text-foreground'>
                        {step.estimated_credits} Credits
                      </span>
                      {step.status === 'FAILED' && (
                        <Button
                          size='sm'
                          variant='destructive'
                          className='h-7 text-xs px-2.5'
                          disabled={isRetrying === step.step_index}
                          onClick={() => handleRetryStep(step.step_index)}
                        >
                          {isRetrying === step.step_index ? (
                            <Spinner className='mr-1.5 size-3' />
                          ) : (
                            <RotateCw className='mr-1.5 size-3' />
                          )}
                          ลองใหม่
                        </Button>
                      )}
                    </div>
                  </div>

                  {/* Output Preview if succeeded */}
                  {step.output_result && (
                    <div className='mt-3 pt-3 border-t border-border/40'>
                      {(() => {
                        try {
                          const out = JSON.parse(step.output_result)
                          const isVideo =
                            step.logical_tool === 'image-to-video' ||
                            out.video_url ||
                            out.output_url?.endsWith('.mp4')

                          return (
                            <div className='flex items-center gap-3'>
                              {isVideo ? (
                                <video
                                  src={out.video_url || out.output_url}
                                  poster={out.poster_url || out.thumbnail_url}
                                  controls
                                  className='h-20 rounded-lg border border-border bg-black'
                                />
                              ) : (
                                <img
                                  src={out.output_url}
                                  alt='Step output'
                                  className='h-20 w-20 rounded-lg border border-border object-cover'
                                />
                              )}
                              <div className='text-xs space-y-1'>
                                <p className='font-medium text-emerald-600 dark:text-emerald-400'>
                                  สำเร็จ • ผลลัพธ์พร้อมใช้งาน
                                </p>
                                <a
                                  href={out.output_url}
                                  target='_blank'
                                  rel='noreferrer'
                                  className='inline-flex items-center gap-1 text-[11px] text-primary hover:underline'
                                >
                                  เปิดดูไฟล์เต็ม <ExternalLink className='size-3' />
                                </a>
                              </div>
                            </div>
                          )
                        } catch {
                          return null
                        }
                      })()}
                    </div>
                  )}
                </div>
              ))}
            </div>

            {/* Total quote bar */}
            <div className='flex items-center justify-between rounded-xl bg-muted/40 p-4 border border-border/60'>
              <div>
                <p className='text-xs text-muted-foreground'>ค่าใช้จ่ายรวมทั้งหมดตามใบเสนอราคา:</p>
                <p className='text-lg font-bold text-foreground'>
                  {plan.total_estimated_credits} Credits{' '}
                  <span className='text-xs font-normal text-muted-foreground'>
                    (ชำระทีละขั้นตอนตามที่สำเร็จจริง)
                  </span>
                </p>
              </div>
              <div className='text-right'>
                <p className='text-xs text-muted-foreground'>เครดิตที่ตัดจริงแล้ว:</p>
                <p className='text-sm font-semibold text-emerald-600 dark:text-emerald-400'>
                  {plan.total_settled_credits} Credits
                </p>
              </div>
            </div>

            {/* Consent checkbox for sensitive workflows */}
            {plan.requires_consent && (
              <div className='rounded-xl border border-amber-500/30 bg-amber-500/10 p-4 space-y-2'>
                <div className='flex items-start gap-2.5'>
                  <Checkbox
                    id='consent'
                    checked={consentConfirmed}
                    onCheckedChange={(c) => setConsentConfirmed(!!c)}
                  />
                  <label htmlFor='consent' className='text-xs text-foreground cursor-pointer leading-relaxed'>
                    <span className='font-semibold text-amber-600 dark:text-amber-400'>
                      การยืนยันสิทธิ์และความยินยอม (Safety Consent Required):
                    </span>{' '}
                    ฉันยืนยันว่าฉันมีสิทธิ์ตามกฎหมายและได้รับความยินยอมจากบุคคลในภาพในการใช้เทคโนโลยี AI ดัดแปลงอัตลักษณ์
                  </label>
                </div>
              </div>
            )}

            {/* Action Buttons */}
            {plan.status === 'DRAFT' && (
              <Button
                size='lg'
                onClick={handleConfirmExecute}
                disabled={isExecuting || (plan.requires_consent && !consentConfirmed)}
                className='w-full font-bold'
              >
                {isExecuting ? (
                  <>
                    <Spinner className='mr-2 size-4' />
                    กำลังดำเนินการเวิร์กโฟลว์ตามลำดับขั้นตอน...
                  </>
                ) : (
                  <>
                    <Play className='mr-2 size-4 fill-current' />
                    ยืนยันแผนงานและเริ่มสร้าง (Confirm & Execute)
                  </>
                )}
              </Button>
            )}

            {plan.status === 'COMPLETED' && (
              <div className='rounded-xl border border-emerald-500/30 bg-emerald-500/10 p-4 text-center space-y-1.5'>
                <p className='text-sm font-bold text-emerald-600 dark:text-emerald-400 flex items-center justify-center gap-1.5'>
                  <CheckCircle2 className='size-4' /> เวิร์กโฟลว์เสร็จสิ้นสมบูรณ์ครบทุกขั้นตอน
                </p>
                <p className='text-xs text-muted-foreground'>
                  ยอดเครดิตที่ตัดจริงรวม: {plan.total_settled_credits} Credits ในกระเป๋า Tora ของคุณ
                </p>
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  )
}
