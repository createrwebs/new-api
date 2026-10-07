import { useQuery } from '@tanstack/react-query'
import { Bot, History, LayoutGrid, Sparkles, Wand2 } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Spinner } from '@/components/ui/spinner'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'

import { getStudioTemplates, getStudioTools } from './api'
import { CreatorAssistant } from './components/CreatorAssistant'
import { StudioCatalog } from './components/StudioCatalog'
import { StudioHistory } from './components/StudioHistory'
import { StudioPlayground } from './components/StudioPlayground'
import type { StudioJob, StudioTemplate } from './types'

interface StudioProps {
  initialTool?: string
  initialTab?: 'assistant' | 'catalog' | 'playground' | 'history'
}

export function Studio({
  initialTool = 'image-generate',
  initialTab = 'assistant',
}: StudioProps) {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState<'assistant' | 'catalog' | 'playground' | 'history'>(
    initialTab
  )
  const [selectedToolSlug, setSelectedToolSlug] = useState<string>(initialTool)
  const [selectedTemplate, setSelectedTemplate] =
    useState<StudioTemplate | null>(null)

  // Fetch tools & templates
  const { data: tools = [], isLoading: isLoadingTools } = useQuery({
    queryKey: ['studio', 'tools'],
    queryFn: getStudioTools,
    staleTime: 60 * 1000,
  })

  const { data: templates = [], isLoading: isLoadingTemplates } = useQuery({
    queryKey: ['studio', 'templates'],
    queryFn: () => getStudioTemplates(),
    staleTime: 60 * 1000,
  })

  // Select tool from catalog
  const handleSelectTool = (slug: string, template?: StudioTemplate) => {
    setSelectedToolSlug(slug)
    if (template) {
      setSelectedTemplate(template)
    } else {
      setSelectedTemplate(null)
    }
    setActiveTab('playground')
  }

  // Remix job from history
  const handleRemix = (job: StudioJob) => {
    setSelectedToolSlug(job.tool_id)
    if (job.template_id) {
      const match = templates.find((tpl) => tpl.id === job.template_id)
      if (match) setSelectedTemplate(match)
    }
    setActiveTab('playground')
  }

  if (isLoadingTools || isLoadingTemplates) {
    return (
      <SectionPageLayout>
        <div className='flex h-96 items-center justify-center'>
          <Spinner className='size-8 text-primary' />
        </div>
      </SectionPageLayout>
    )
  }

  return (
    <SectionPageLayout>
      <div className='space-y-6'>
        {/* Navigation Tabs Bar */}
        <div className='flex items-center justify-between border-b border-border/60 pb-3'>
          <div className='flex items-center gap-2.5'>
            <div className='flex size-9 items-center justify-center rounded-xl bg-primary/10 text-primary'>
              <Sparkles className='size-5' />
            </div>
            <div>
              <h1 className='text-lg font-bold tracking-tight text-foreground'>
                {t('Tora Studio')}
              </h1>
              <p className='text-xs text-muted-foreground'>
                {t('เครื่องมือสร้างภาพ วิดีโอ และสื่อโฆษณาด้วย AI')}
              </p>
            </div>
          </div>

          <Tabs
            value={activeTab}
            onValueChange={(val) =>
              setActiveTab(val as 'assistant' | 'catalog' | 'playground' | 'history')
            }
          >
            <TabsList className='h-9 bg-muted/60 p-1'>
              <TabsTrigger value='assistant' className='h-7 text-xs font-medium'>
                <Bot className='mr-1.5 size-3.5 text-primary' />
                {t('ผู้ช่วย AI (Assistant)')}
              </TabsTrigger>
              <TabsTrigger value='catalog' className='h-7 text-xs font-medium'>
                <LayoutGrid className='mr-1.5 size-3.5' />
                {t('คลังเครื่องมือ (Catalog)')}
              </TabsTrigger>
              <TabsTrigger value='playground' className='h-7 text-xs font-medium'>
                <Wand2 className='mr-1.5 size-3.5' />
                {t('สตูดิโอ (Playground)')}
              </TabsTrigger>
              <TabsTrigger value='history' className='h-7 text-xs font-medium'>
                <History className='mr-1.5 size-3.5' />
                {t('ผลงานของฉัน (History)')}
              </TabsTrigger>
            </TabsList>
          </Tabs>
        </div>

        {/* Tab 0: Creator Assistant */}
        {activeTab === 'assistant' && <CreatorAssistant />}

        {/* Tab 1: Catalog */}
        {activeTab === 'catalog' && (
          <StudioCatalog
            tools={tools}
            templates={templates}
            onSelectTool={handleSelectTool}
          />
        )}

        {/* Tab 2: Playground */}
        {activeTab === 'playground' && (
          <StudioPlayground
            tools={tools}
            templates={templates}
            selectedToolSlug={selectedToolSlug}
            initialTemplate={selectedTemplate}
            onToolChange={(slug) => {
              setSelectedToolSlug(slug)
              setSelectedTemplate(null)
            }}
          />
        )}

        {/* Tab 3: History */}
        {activeTab === 'history' && (
          <StudioHistory tools={tools} onRemix={handleRemix} />
        )}
      </div>
    </SectionPageLayout>
  )
}
