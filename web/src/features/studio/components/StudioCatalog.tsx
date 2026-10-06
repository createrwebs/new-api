import {
  Clock,
  Coins,
  Layers,
  Palette,
  Search,
  ShoppingBag,
  Sparkles,
  Video,
  Wand2,
} from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'

import type { StudioTemplate, StudioTool, StudioToolCategory } from '../types'

interface StudioCatalogProps {
  tools: StudioTool[]
  templates: StudioTemplate[]
  onSelectTool: (toolSlug: string, template?: StudioTemplate) => void
}

export function StudioCatalog({
  tools,
  templates,
  onSelectTool,
}: StudioCatalogProps) {
  const { t } = useTranslation()
  const [search, setSearch] = useState('')
  const [selectedCategory, setSelectedCategory] =
    useState<StudioToolCategory>('all')

  const categories: { id: StudioToolCategory; label: string; icon: React.ElementType }[] = [
    { id: 'all', label: t('ทั้งหมด'), icon: Sparkles },
    { id: 'image', label: t('สร้างรูปภาพ'), icon: Palette },
    { id: 'video', label: t('วิดีโอ AI'), icon: Video },
    { id: 'utility', label: t('เครื่องมือปรับแต่ง'), icon: Wand2 },
    { id: 'commercial', label: t('อีคอมเมิร์ซ'), icon: ShoppingBag },
  ]

  const filteredTools = useMemo(() => {
    return tools.filter((tool) => {
      const matchCat =
        selectedCategory === 'all' || tool.category === selectedCategory
      const query = search.toLowerCase().trim()
      const matchQuery =
        !query ||
        tool.name.toLowerCase().includes(query) ||
        tool.name_th?.toLowerCase().includes(query) ||
        tool.description?.toLowerCase().includes(query) ||
        tool.slug.toLowerCase().includes(query)
      return matchCat && matchQuery
    })
  }, [tools, selectedCategory, search])

  const filteredTemplates = useMemo(() => {
    if (selectedCategory === 'all') return templates
    return templates.filter((tpl) => tpl.category === selectedCategory)
  }, [templates, selectedCategory])

  return (
    <div className='space-y-8'>
      {/* Hero Header */}
      <div className='relative overflow-hidden rounded-2xl border border-primary/20 bg-linear-to-b from-primary/10 via-background to-background p-6 sm:p-8'>
        <div className='relative z-10 max-w-2xl space-y-3'>
          <div className='inline-flex items-center gap-1.5 rounded-full bg-primary/10 px-3 py-1 text-xs font-medium text-primary'>
            <Sparkles className='size-3.5' />
            <span>{t('Tora Studio V1 — Universal Creator Platform')}</span>
          </div>
          <h1 className='text-2xl font-bold tracking-tight text-foreground sm:text-3xl'>
            {t('สร้างสรรค์สื่อดิจิทัลด้วย AI ระดับโปร')}
          </h1>
          <p className='text-sm text-muted-foreground sm:text-base'>
            {t(
              'ใช้งานเครื่องมือลบพื้นหลัง สตูดิโอถ่ายสินค้า แต่งภาพ และสร้างวิดีโอระดับภาพยนตร์ ทั้งหมดนี้ใช้ Tora Credits กระเป๋าเดียว ไม่ต้องเสียค่าสมาชิกหลายที่'
            )}
          </p>
        </div>
      </div>

      {/* Filter and Search Bar */}
      <div className='flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between'>
        <div className='flex flex-wrap items-center gap-1.5'>
          {categories.map((cat) => {
            const Icon = cat.icon
            const active = selectedCategory === cat.id
            return (
              <Button
                key={cat.id}
                size='sm'
                variant={active ? 'default' : 'outline'}
                onClick={() => setSelectedCategory(cat.id)}
                className='h-8 text-xs font-medium'
              >
                <Icon className='mr-1.5 size-3.5' />
                {cat.label}
              </Button>
            )
          })}
        </div>

        <div className='relative w-full sm:w-64'>
          <Search className='absolute left-2.5 top-2.5 size-4 text-muted-foreground' />
          <Input
            placeholder={t('ค้นหาเครื่องมือ...')}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className='h-9 pl-9 text-xs'
          />
        </div>
      </div>

      {/* Tools Grid */}
      <div className='space-y-4'>
        <div className='flex items-center justify-between'>
          <h2 className='text-base font-semibold text-foreground'>
            {t('เครื่องมือทั้งหมด')} ({filteredTools.length})
          </h2>
        </div>

        <div className='grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4'>
          {filteredTools.map((tool) => {
            const isBlocked = tool.status === 'operator_blocked'
            const isDisabled = tool.status === 'disabled'
            const isActive = tool.status === 'active'

            return (
              <Card
                key={tool.id}
                className='group flex flex-col justify-between border-border/60 transition-all hover:border-primary/40 hover:shadow-md'
              >
                <CardHeader className='pb-3'>
                  <div className='flex items-start justify-between gap-2'>
                    <div className='flex size-10 items-center justify-center rounded-xl bg-primary/10 text-primary group-hover:scale-105 transition-transform'>
                      <Wand2 className='size-5' />
                    </div>
                    <div className='flex flex-wrap items-center gap-1.5'>
                      {tool.badge && (
                        <Badge variant='secondary' className='text-[10px]'>
                          {tool.badge}
                        </Badge>
                      )}
                      {isActive && (
                        <Badge className='bg-emerald-500/15 text-[10px] text-emerald-600 dark:text-emerald-400 border-none'>
                          {tool.credit_cost} Cr
                        </Badge>
                      )}
                      {isBlocked && (
                        <Badge variant='outline' className='text-[10px] text-amber-500 border-amber-500/30'>
                          {t('Provider Key Req')}
                        </Badge>
                      )}
                      {isDisabled && (
                        <Badge variant='outline' className='text-[10px] text-muted-foreground'>
                          {t('เร็วๆ นี้')}
                        </Badge>
                      )}
                    </div>
                  </div>

                  <CardTitle className='mt-3 text-sm font-semibold'>
                    {tool.name_th || tool.name}
                  </CardTitle>
                  <p className='text-xs text-muted-foreground line-clamp-2 mt-1'>
                    {tool.description_th || tool.description}
                  </p>
                </CardHeader>

                <CardContent className='pb-3 text-[11px] text-muted-foreground'>
                  <div className='flex items-center gap-4'>
                    <span className='flex items-center gap-1'>
                      <Clock className='size-3 text-muted-foreground/70' />
                      ~{tool.estimated_sec}s
                    </span>
                    <span className='flex items-center gap-1'>
                      <Coins className='size-3 text-muted-foreground/70' />
                      {tool.credit_cost} Credits
                    </span>
                  </div>
                </CardContent>

                <CardFooter className='pt-0'>
                  <Button
                    size='sm'
                    className='w-full text-xs font-medium'
                    variant={isActive ? 'default' : 'secondary'}
                    disabled={isDisabled}
                    onClick={() => onSelectTool(tool.slug)}
                  >
                    {isDisabled ? (
                      t('เร็วๆ นี้')
                    ) : (
                      <>
                        <Sparkles className='mr-1.5 size-3.5' />
                        {t('ใช้งานเครื่องมือ')}
                      </>
                    )}
                  </Button>
                </CardFooter>
              </Card>
            )
          })}
        </div>
      </div>

      {/* Curated Templates Section */}
      {filteredTemplates.length > 0 && (
        <div className='space-y-4 pt-4'>
          <div className='flex items-center justify-between'>
            <div>
              <h2 className='text-base font-semibold text-foreground'>
                {t('แม่แบบสตูดิโอยอดนิยม (Curated Presets)')}
              </h2>
              <p className='text-xs text-muted-foreground'>
                {t('คลิกเพื่อโหลดการตั้งค่าพร้อมสร้างทันที')}
              </p>
            </div>
          </div>

          <div className='grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6'>
            {filteredTemplates.map((template) => (
              <div
                key={template.id}
                onClick={() => onSelectTool(template.tool_id, template)}
                className='group cursor-pointer space-y-2 rounded-xl border border-border/60 bg-card p-2.5 transition-all hover:border-primary/50 hover:shadow-xs'
              >
                <div className='relative aspect-square overflow-hidden rounded-lg bg-muted'>
                  {template.preview_image_url ? (
                    <img
                      src={template.preview_image_url}
                      alt={template.title}
                      className='size-full object-cover transition-transform group-hover:scale-105'
                      loading='lazy'
                    />
                  ) : (
                    <div className='flex size-full items-center justify-center text-muted-foreground'>
                      <Layers className='size-6' />
                    </div>
                  )}
                  {template.badge && (
                    <span className='absolute right-1.5 top-1.5 rounded-sm bg-black/70 px-1.5 py-0.5 text-[9px] font-medium text-white backdrop-blur-xs'>
                      {template.badge}
                    </span>
                  )}
                </div>

                <div>
                  <div className='text-xs font-medium text-foreground line-clamp-1 group-hover:text-primary transition-colors'>
                    {template.title_th || template.title}
                  </div>
                  <div className='text-[10px] text-muted-foreground line-clamp-1'>
                    {template.description_th || template.description}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
