/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import {
  ArrowLeft,
  Check,
  Layers,
  Loader2,
  Plus,
  RefreshCw,
  Search,
  Server,
  Sparkles,
  TestTube,
  Trash2,
  Wifi,
  WifiOff,
  Zap,
} from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { getGroups as getUserGroups } from '@/features/users/api'
import { requireServerSuccess } from '@/lib/server-error-message'
import { cn } from '@/lib/utils'

import {
  createChannel,
  deleteChannel,
  getChannels,
  testChannel,
  updateChannelStatus,
} from '../api'
import {
  CHANNEL_STATUS,
  CHANNEL_TYPE_9ROUTER,
} from '../constants'
import type { AddChannelRequest, Channel } from '../types'

interface NineRouterModelItem {
  id: string
  object: string
  owned_by?: string
  capabilities?: {
    reasoning?: boolean
    vision?: boolean
    tools?: boolean
  }
}

const STORAGE_GATEWAY_URL_KEY = '9router_gateway_url'
const STORAGE_API_KEY_KEY = '9router_api_key'
const DEFAULT_GATEWAY_URL = 'http://localhost:20128'

const RECOMMENDED_FREE_MODELS = [
  'combo',
  'oc/glm-5.3-flash-free',
  'oc/deepseek-v4.1-flash-free',
  'oc/mimo-v2.6-flash-free',
  'oc/nemotron-3.5-super-vl-free',
  'oc/nemotron-3.5-lightning-free',
  'oc/union-alpha',
  'oc/jev-1.13-free',
  'cl/recursive/flint-v1.0:free',
  'cl/nvidia/nemotron-3.5-super-vl:free',
  'cl/dots-studio/dots-3-note-preview:free',
  'cl/liquid/lfm-2.5-2.6b:free',
  'cl/poolside/laguna-s-2.1-free',
]

export function NineRouterPoolView() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const [gatewayUrl, setGatewayUrl] = useState(() => {
    return localStorage.getItem(STORAGE_GATEWAY_URL_KEY) || DEFAULT_GATEWAY_URL
  })
  const [apiKey, setApiKey] = useState(() => {
    return localStorage.getItem(STORAGE_API_KEY_KEY) || 'sk-1d745ca5d2a51841-9mwj6u-d636ae47'
  })

  const [isTesting, setIsTesting] = useState(false)
  const [isConnected, setIsConnected] = useState<boolean | null>(null)
  const [latencyMs, setLatencyMs] = useState<number | null>(null)
  const [remoteModels, setRemoteModels] = useState<NineRouterModelItem[]>([])
  const [selectedModels, setSelectedModels] = useState<Set<string>>(
    new Set(RECOMMENDED_FREE_MODELS)
  )
  const [modelSearch, setModelSearch] = useState('')
  const [filterMode, setFilterMode] = useState<'free' | 'all'>('free')

  // Channel Creation Form State
  const [channelName, setChannelName] = useState('9Router Free Tier')
  const [targetGroup, setTargetGroup] = useState('default')
  const [priority, setPriority] = useState(10)
  const [weight, setWeight] = useState(1)
  const [modelMapping, setModelMapping] = useState(
    '{\n  "gpt-4o-mini": "combo",\n  "gemini-2.5-flash": "combo",\n  "claude-3-5-haiku": "combo"\n}'
  )

  // Fetch Existing Channels in New-API to see 9Router channels in pool
  const existingChannelsQuery = useQuery({
    queryKey: ['channels', '9router-pool'],
    queryFn: async () => {
      const res = await getChannels({
        p: 0,
        page_size: 100,
        type: CHANNEL_TYPE_9ROUTER,
      })
      return res.data?.items || []
    },
    staleTime: 10 * 1000,
  })

  // Fetch available user groups
  const groupsQuery = useQuery({
    queryKey: ['user-groups'],
    queryFn: async () => {
      const res = await getUserGroups()
      return requireServerSuccess(res).data || ['default']
    },
    staleTime: 5 * 60 * 1000,
  })

  // Probe 9Router Connection & Fetch Models
  const testConnection = async (urlToTest = gatewayUrl, keyToTest = apiKey) => {
    setIsTesting(true)
    const cleanUrl = urlToTest.trim().replace(/\/+$/, '')
    const startTime = performance.now()
    try {
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
      }
      if (keyToTest.trim()) {
        headers.Authorization = `Bearer ${keyToTest.trim()}`
      }

      const res = await fetch(`${cleanUrl}/v1/models`, {
        method: 'GET',
        headers,
      })

      const elapsed = Math.round(performance.now() - startTime)
      if (!res.ok) {
        throw new Error(`HTTP ${res.status}: ${res.statusText}`)
      }

      const json = await res.json()
      const modelsList: NineRouterModelItem[] = Array.isArray(json.data)
        ? json.data
        : []

      setIsConnected(true)
      setLatencyMs(elapsed)
      setRemoteModels(modelsList)

      // Update selected models to include any matched recommended ones that exist on server
      const availableIds = new Set(modelsList.map((m) => m.id))
      const matched = new Set<string>()
      for (const rec of RECOMMENDED_FREE_MODELS) {
        if (availableIds.has(rec)) {
          matched.add(rec)
        }
      }
      // Also add any models ending in -free or :free or starting with oc/
      for (const m of modelsList) {
        if (
          m.id === 'combo' ||
          m.id.startsWith('oc/') ||
          m.id.endsWith('-free') ||
          m.id.endsWith(':free')
        ) {
          matched.add(m.id)
        }
      }
      if (matched.size > 0) {
        setSelectedModels(matched)
      }

      // Save settings in localStorage
      localStorage.setItem(STORAGE_GATEWAY_URL_KEY, cleanUrl)
      if (keyToTest.trim()) {
        localStorage.setItem(STORAGE_API_KEY_KEY, keyToTest.trim())
      }

      toast.success(
        t('Connected to 9Router! Found {{count}} models in {{ms}}ms', {
          count: modelsList.length,
          ms: elapsed,
        })
      )
    } catch (err: unknown) {
      setIsConnected(false)
      setLatencyMs(null)
      const message = err instanceof Error ? err.message : String(err)
      toast.error(t('Cannot connect to 9Router: {{error}}', { error: message }))
    } finally {
      setIsTesting(false)
    }
  }

  // Probe on mount
  useEffect(() => {
    testConnection(gatewayUrl, apiKey)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // Filtered models for display
  const displayedModels = useMemo(() => {
    return remoteModels.filter((model) => {
      const id = model.id.toLowerCase()
      const query = modelSearch.toLowerCase().trim()
      if (query && !id.includes(query)) {
        return false
      }
      if (filterMode === 'free') {
        const isFree =
          id === 'combo' ||
          id.startsWith('oc/') ||
          id.endsWith('-free') ||
          id.endsWith(':free') ||
          id.includes('free')
        return isFree
      }
      return true
    })
  }, [remoteModels, modelSearch, filterMode])

  // Select/Deselect handlers
  const handleToggleModel = (id: string) => {
    setSelectedModels((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }

  const handleSelectAllFree = () => {
    const next = new Set(selectedModels)
    for (const m of remoteModels) {
      const id = m.id
      if (
        id === 'combo' ||
        id.startsWith('oc/') ||
        id.endsWith('-free') ||
        id.endsWith(':free') ||
        id.includes('free')
      ) {
        next.add(id)
      }
    }
    setSelectedModels(next)
  }

  const handleClearSelection = () => {
    setSelectedModels(new Set())
  }

  // Create Channel Mutation (Add to Pool)
  const createChannelMutation = useMutation({
    mutationFn: async () => {
      if (selectedModels.size === 0) {
        throw new Error(t('Please select at least one model for the pool channel'))
      }
      const modelsStr = [...selectedModels].join(',')
      const payload: AddChannelRequest = {
        mode: 'single',
        channel: {
          name: channelName.trim() || '9Router Free Tier',
          type: CHANNEL_TYPE_9ROUTER,
          key: apiKey.trim() || 'EMPTY',
          base_url: gatewayUrl.trim().replace(/\/+$/, ''),
          models: modelsStr,
          group: targetGroup.trim() || 'default',
          priority: Number(priority) || 0,
          weight: Number(weight) || 1,
          model_mapping: modelMapping.trim() || undefined,
        },
      }
      const res = await createChannel(payload)
      if (!res.success) {
        throw new Error(res.message || t('Failed to create channel'))
      }
      return res
    },
    onSuccess: () => {
      toast.success(t('9Router Free Tier channel added to the pool successfully!'))
      queryClient.invalidateQueries({ queryKey: ['channels'] })
    },
    onError: (err: unknown) => {
      const message = err instanceof Error ? err.message : String(err)
      toast.error(message)
    },
  })

  // Test Channel latency in pool
  const testChannelMutation = useMutation({
    mutationFn: async (id: number) => {
      const res = await testChannel(id, { model: 'combo' })
      return res
    },
    onSuccess: (res) => {
      if (res.success) {
        const timeMs =
          typeof res.time === 'number'
            ? Math.round(res.time * 1000)
            : res.data?.response_time || 0
        toast.success(
          t('Channel test passed in {{ms}}ms', {
            ms: timeMs,
          })
        )
      } else {
        toast.error(res.message || t('Channel test failed'))
      }
      queryClient.invalidateQueries({ queryKey: ['channels'] })
    },
  })

  // Delete channel in pool
  const deleteChannelMutation = useMutation({
    mutationFn: async (id: number) => {
      const res = await deleteChannel(id)
      if (!res.success) {
        throw new Error(res.message || t('Failed to delete channel'))
      }
      return res
    },
    onSuccess: () => {
      toast.success(t('Channel removed from pool'))
      queryClient.invalidateQueries({ queryKey: ['channels'] })
    },
    onError: (err: unknown) => {
      const message = err instanceof Error ? err.message : String(err)
      toast.error(message)
    },
  })

  // Status toggle
  const toggleStatusMutation = useMutation({
    mutationFn: async ({ id, status }: { id: number; status: number }) => {
      const nextStatus =
        status === CHANNEL_STATUS.ENABLED
          ? CHANNEL_STATUS.MANUAL_DISABLED
          : CHANNEL_STATUS.ENABLED
      const res = await updateChannelStatus(id, nextStatus)
      if (!res.success) {
        throw new Error(res.message || t('Failed to update channel status'))
      }
      return res
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['channels'] })
    },
  })

  const channelsInPool = existingChannelsQuery.data || []

  const renderConnectionStatus = () => {
    if (isConnected === true) {
      return (
        <Badge variant='outline' className='border-emerald-500/30 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 py-1.5 px-3'>
          <Wifi className='mr-1.5 h-3.5 w-3.5' />
          {t('Connected ({{ms}}ms)', { ms: latencyMs || 0 })}
        </Badge>
      )
    }
    if (isConnected === false) {
      return (
        <Badge variant='destructive' className='py-1.5 px-3'>
          <WifiOff className='mr-1.5 h-3.5 w-3.5' />
          {t('Gateway Offline')}
        </Badge>
      )
    }
    return (
      <Badge variant='secondary' className='py-1.5 px-3'>
        <Loader2 className='mr-1.5 h-3.5 w-3.5 animate-spin' />
        {t('Checking...')}
      </Badge>
    )
  }

  const renderModelListContent = () => {
    if (remoteModels.length === 0) {
      return (
        <div className='py-12 text-center text-muted-foreground space-y-3'>
          <Server className='mx-auto h-8 w-8 opacity-40' />
          <p className='text-sm'>{t('No models loaded yet.')}</p>
          <Button
            size='sm'
            variant='outline'
            onClick={() => testConnection()}
            disabled={isTesting}
          >
            {t('Connect to 9Router')}
          </Button>
        </div>
      )
    }

    if (displayedModels.length === 0) {
      return (
        <div className='py-12 text-center text-muted-foreground'>
          <p className='text-sm'>{t('No models match the search filter.')}</p>
        </div>
      )
    }

    return displayedModels.map((model) => {
      const isSelected = selectedModels.has(model.id)
      const isCombo = model.id === 'combo' || model.owned_by === 'combo'
      const isOpenCode = model.id.startsWith('oc/')
      const isFreeTier =
        model.id.endsWith('-free') ||
        model.id.endsWith(':free') ||
        model.id.includes('free')

      return (
        <div
          key={model.id}
          onClick={() => handleToggleModel(model.id)}
          className={cn(
            'flex items-center justify-between p-3 cursor-pointer transition-colors hover:bg-muted/50',
            isSelected && 'bg-primary/5'
          )}
        >
          <div className='flex items-center gap-3 min-w-0'>
            <Checkbox
              checked={isSelected}
              onCheckedChange={() => handleToggleModel(model.id)}
              onClick={(e) => e.stopPropagation()}
            />
            <div className='min-w-0'>
              <div className='flex items-center gap-2 flex-wrap'>
                <span className='font-mono text-sm font-medium truncate'>
                  {model.id}
                </span>
                {isCombo && (
                  <Badge variant='outline' className='bg-amber-500/10 text-amber-600 border-amber-500/30 text-[10px] py-0 px-1.5'>
                    {t('Auto-Fallback Combo')}
                  </Badge>
                )}
                {isOpenCode && (
                  <Badge variant='outline' className='bg-orange-500/10 text-orange-600 border-orange-500/30 text-[10px] py-0 px-1.5'>
                    {t('OpenCode Free')}
                  </Badge>
                )}
                {isFreeTier && !isOpenCode && (
                  <Badge variant='outline' className='bg-emerald-500/10 text-emerald-600 border-emerald-500/30 text-[10px] py-0 px-1.5'>
                    {t('Free Tier')}
                  </Badge>
                )}
              </div>
              {model.owned_by && (
                <p className='text-muted-foreground text-xs'>
                  {t('Provider: {{provider}}', { provider: model.owned_by })}
                </p>
              )}
            </div>
          </div>

          <div className='flex items-center gap-1.5 shrink-0 pl-2'>
            {model.capabilities?.reasoning && (
              <Badge variant='secondary' className='text-[10px] px-1.5 py-0'>
                {t('Thinking')}
              </Badge>
            )}
            {model.capabilities?.vision && (
              <Badge variant='secondary' className='text-[10px] px-1.5 py-0'>
                {t('Vision')}
              </Badge>
            )}
            {model.capabilities?.tools && (
              <Badge variant='secondary' className='text-[10px] px-1.5 py-0'>
                {t('Tools')}
              </Badge>
            )}
          </div>
        </div>
      )
    })
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        <div className='flex items-center gap-3'>
          <Button
            variant='ghost'
            size='icon'
            className='h-8 w-8'
            render={<Link to='/channels' />}
            aria-label={t('Back to Channels')}
          >
            <ArrowLeft className='h-4 w-4' />
          </Button>
          <div className='flex items-center gap-2'>
            <span className='font-semibold tracking-tight text-xl'>
              {t('9Router Free Tier Pool')}
            </span>
            <Badge variant='outline' className='bg-primary/10 text-primary border-primary/20'>
              <Sparkles className='mr-1 h-3 w-3' />
              {t('Free Tier Gateway')}
            </Badge>
          </div>
        </div>
      </SectionPageLayout.Title>

      <SectionPageLayout.Actions>
        <Button
          variant='outline'
          size='sm'
          onClick={() => testConnection()}
          disabled={isTesting}
        >
          {isTesting ? (
            <Loader2 className='mr-2 h-4 w-4 animate-spin' />
          ) : (
            <RefreshCw className='mr-2 h-4 w-4' />
          )}
          {t('Test Gateway Connection')}
        </Button>
      </SectionPageLayout.Actions>

      <SectionPageLayout.Content>
        <div className='space-y-6 pb-12'>
          {/* Top Info Banner */}
          <Card className='border-primary/20 bg-gradient-to-r from-primary/5 via-primary/2 to-transparent'>
            <CardContent className='flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 p-5'>
              <div className='space-y-1'>
                <h3 className='font-semibold text-base flex items-center gap-2'>
                  <Zap className='h-4 w-4 text-primary' />
                  {t('Zero-Cost High-Availability Routing with 9Router')}
                </h3>
                <p className='text-muted-foreground text-sm max-w-2xl'>
                  {t(
                    'Connect 9Router to provide automatic fallback to free models (OpenCode Free, Kiro, and Smart 3-Tier Combos) with zero token cost and high reliability. Add this channel to the New-API channel pool to serve inference requests automatically.'
                  )}
                </p>
              </div>
              <div className='flex items-center gap-2 self-end sm:self-center'>
                {renderConnectionStatus()}
              </div>
            </CardContent>
          </Card>

          {/* Grid: Gateway Settings & Add To Pool */}
          <div className='grid grid-cols-1 lg:grid-cols-12 gap-6'>
            {/* Column Left: Gateway Connection Settings */}
            <div className='lg:col-span-5 space-y-6'>
              <Card>
                <CardHeader>
                  <CardTitle className='text-base flex items-center gap-2'>
                    <Server className='h-4 w-4' />
                    {t('9Router Gateway Settings')}
                  </CardTitle>
                  <CardDescription>
                    {t('Configure your local or remote 9Router gateway connection')}
                  </CardDescription>
                </CardHeader>
                <CardContent className='space-y-4'>
                  <div className='space-y-2'>
                    <Label htmlFor='gateway-url'>{t('Gateway Base URL')}</Label>
                    <Input
                      id='gateway-url'
                      value={gatewayUrl}
                      onChange={(e) => setGatewayUrl(e.target.value)}
                      placeholder='http://localhost:20128'
                    />
                    <p className='text-muted-foreground text-xs'>
                      {t('Default port for 9Router daemon is 20128')}
                    </p>
                  </div>

                  <div className='space-y-2'>
                    <Label htmlFor='api-key'>{t('API Key (Optional)')}</Label>
                    <Input
                      id='api-key'
                      type='password'
                      value={apiKey}
                      onChange={(e) => setApiKey(e.target.value)}
                      placeholder='sk-... or empty for unauthenticated mode'
                    />
                    <p className='text-muted-foreground text-xs'>
                      {t('Key from 9Router dashboard or leave empty if requireApiKey is off')}
                    </p>
                  </div>
                </CardContent>
                <CardFooter className='border-t pt-4 flex justify-between'>
                  <Button
                    variant='outline'
                    size='sm'
                    onClick={() => testConnection()}
                    disabled={isTesting}
                  >
                    {isTesting ? (
                      <Loader2 className='mr-2 h-3.5 w-3.5 animate-spin' />
                    ) : (
                      <RefreshCw className='mr-2 h-3.5 w-3.5' />
                    )}
                    {t('Check Connection')}
                  </Button>
                  <Button
                    size='sm'
                    variant='secondary'
                    onClick={() => {
                      localStorage.setItem(STORAGE_GATEWAY_URL_KEY, gatewayUrl)
                      localStorage.setItem(STORAGE_API_KEY_KEY, apiKey)
                      toast.success(t('Saved gateway configuration'))
                    }}
                  >
                    <Check className='mr-1.5 h-3.5 w-3.5' />
                    {t('Save Config')}
                  </Button>
                </CardFooter>
              </Card>

              {/* Add to Pool Card */}
              <Card className='border-primary/30'>
                <CardHeader>
                  <CardTitle className='text-base flex items-center gap-2'>
                    <Layers className='h-4 w-4 text-primary' />
                    {t('Add to Channel Pool')}
                  </CardTitle>
                  <CardDescription>
                    {t('Register this 9Router connection as a channel in New-API')}
                  </CardDescription>
                </CardHeader>
                <CardContent className='space-y-4'>
                  <div className='space-y-2'>
                    <Label htmlFor='channel-name'>{t('Channel Name')}</Label>
                    <Input
                      id='channel-name'
                      value={channelName}
                      onChange={(e) => setChannelName(e.target.value)}
                      placeholder='9Router Free Tier'
                    />
                  </div>

                  <div className='grid grid-cols-2 gap-4'>
                    <div className='space-y-2'>
                      <Label htmlFor='target-group'>{t('User Group')}</Label>
                      <Input
                        id='target-group'
                        value={targetGroup}
                        onChange={(e) => setTargetGroup(e.target.value)}
                        placeholder='default'
                      />
                      <p className='text-muted-foreground text-xs'>
                        {t('Available: {{groups}}', {
                          groups: (groupsQuery.data || ['default']).join(', '),
                        })}
                      </p>
                    </div>

                    <div className='space-y-2'>
                      <Label htmlFor='priority'>{t('Priority')}</Label>
                      <Input
                        id='priority'
                        type='number'
                        value={priority}
                        onChange={(e) => setPriority(Number(e.target.value))}
                      />
                      <p className='text-muted-foreground text-xs'>
                        {t('Higher priority channels are tried first')}
                      </p>
                    </div>
                  </div>

                  <div className='space-y-2'>
                    <Label htmlFor='weight'>{t('Weight')}</Label>
                    <Input
                      id='weight'
                      type='number'
                      value={weight}
                      onChange={(e) => setWeight(Number(e.target.value))}
                    />
                  </div>

                  <div className='space-y-2'>
                    <Label htmlFor='model-mapping'>{t('Model Redirection Mapping (JSON)')}</Label>
                    <textarea
                      id='model-mapping'
                      className='w-full font-mono text-xs p-2 rounded-md border bg-muted/30 focus:outline-none focus:ring-1 focus:ring-primary'
                      rows={4}
                      value={modelMapping}
                      onChange={(e) => setModelMapping(e.target.value)}
                      placeholder='{"client-model": "upstream-model"}'
                    />
                    <p className='text-muted-foreground text-xs'>
                      {t('Map standard model calls to free fallback combos (e.g. gpt-4o-mini -> combo)')}
                    </p>
                  </div>

                  <div className='rounded-lg bg-muted/40 p-3 text-xs space-y-1.5'>
                    <div className='flex justify-between font-medium'>
                      <span>{t('Selected Models for Pool:')}</span>
                      <span className='text-primary font-bold'>{selectedModels.size}</span>
                    </div>
                    <p className='text-muted-foreground truncate'>
                      {[...selectedModels].slice(0, 5).join(', ')}
                      {selectedModels.size > 5 && ` +${selectedModels.size - 5} more`}
                    </p>
                  </div>
                </CardContent>
                <CardFooter className='border-t pt-4'>
                  <Button
                    className='w-full'
                    onClick={() => createChannelMutation.mutate()}
                    disabled={createChannelMutation.isPending || selectedModels.size === 0}
                  >
                    {createChannelMutation.isPending ? (
                      <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                    ) : (
                      <Plus className='mr-2 h-4 w-4' />
                    )}
                    {t('Add to Channel Pool')}
                  </Button>
                </CardFooter>
              </Card>
            </div>

            {/* Column Right: Free Tier Models & Combos Selector */}
            <div className='lg:col-span-7 space-y-6'>
              <Card className='h-full flex flex-col'>
                <CardHeader>
                  <div className='flex flex-col sm:flex-row sm:items-center justify-between gap-3'>
                    <div>
                      <CardTitle className='text-base flex items-center gap-2'>
                        <Sparkles className='h-4 w-4 text-amber-500' />
                        {t('Available Models in 9Router')}
                      </CardTitle>
                      <CardDescription>
                        {t('Select models to include in your free tier channel pool')}
                      </CardDescription>
                    </div>
                    <div className='flex items-center gap-2'>
                      <Button
                        size='sm'
                        variant={filterMode === 'free' ? 'default' : 'outline'}
                        onClick={() => setFilterMode('free')}
                        className='h-8 text-xs'
                      >
                        {t('Free Tier Only')}
                      </Button>
                      <Button
                        size='sm'
                        variant={filterMode === 'all' ? 'default' : 'outline'}
                        onClick={() => setFilterMode('all')}
                        className='h-8 text-xs'
                      >
                        {t('All Models')} ({remoteModels.length})
                      </Button>
                    </div>
                  </div>

                  {/* Search and Quick Selection Actions */}
                  <div className='pt-3 flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3'>
                    <div className='relative flex-1'>
                      <Search className='text-muted-foreground absolute left-2.5 top-2.5 h-4 w-4' />
                      <Input
                        value={modelSearch}
                        onChange={(e) => setModelSearch(e.target.value)}
                        placeholder={t('Search models (e.g. combo, glm, flint)...')}
                        className='pl-8 h-9 text-xs'
                      />
                    </div>
                    <div className='flex items-center gap-2 shrink-0'>
                      <Button
                        size='sm'
                        variant='ghost'
                        onClick={handleSelectAllFree}
                        className='h-8 text-xs text-primary'
                      >
                        {t('Select Recommended Free')}
                      </Button>
                      <Button
                        size='sm'
                        variant='ghost'
                        onClick={handleClearSelection}
                        className='h-8 text-xs text-muted-foreground'
                      >
                        {t('Clear')}
                      </Button>
                    </div>
                  </div>
                </CardHeader>

                <CardContent className='flex-1 overflow-y-auto max-h-[500px] divide-y border-y'>
                  {renderModelListContent()}
                </CardContent>

                <CardFooter className='p-3 bg-muted/20 text-xs text-muted-foreground flex justify-between'>
                  <span>
                    {t('Showing {{shown}} of {{total}} models', {
                      shown: displayedModels.length,
                      total: remoteModels.length,
                    })}
                  </span>
                  <span>
                    {t('{{count}} selected', { count: selectedModels.size })}
                  </span>
                </CardFooter>
              </Card>
            </div>
          </div>

          {/* Existing 9Router Channels in Pool */}
          <div className='space-y-3 pt-4'>
            <div className='flex items-center justify-between'>
              <div>
                <h3 className='font-semibold text-base'>
                  {t('Active 9Router Channels in New-API Pool')}
                </h3>
                <p className='text-muted-foreground text-xs'>
                  {t('Channels in the system configured to route to 9Router')}
                </p>
              </div>
              <Button
                variant='outline'
                size='sm'
                onClick={() => existingChannelsQuery.refetch()}
              >
                <RefreshCw className='mr-1.5 h-3.5 w-3.5' />
                {t('Refresh Pool')}
              </Button>
            </div>

            {channelsInPool.length === 0 ? (
              <Card>
                <CardContent className='py-8 text-center text-muted-foreground'>
                  <Layers className='mx-auto h-8 w-8 opacity-40 mb-2' />
                  <p className='text-sm'>
                    {t('No 9Router channels in the pool yet. Use the form above to add one!')}
                  </p>
                </CardContent>
              </Card>
            ) : (
              <div className='grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4'>
                {channelsInPool.map((channel: Channel) => (
                  <Card key={channel.id} className='relative overflow-hidden'>
                    <CardHeader className='pb-3'>
                      <div className='flex items-start justify-between gap-2'>
                        <div>
                          <CardTitle className='text-base font-semibold truncate'>
                            {channel.name}
                          </CardTitle>
                          <CardDescription className='text-xs'>
                            ID: #{channel.id} · {channel.base_url || DEFAULT_GATEWAY_URL}
                          </CardDescription>
                        </div>
                        <Switch
                          checked={channel.status === CHANNEL_STATUS.ENABLED}
                          onCheckedChange={() =>
                            toggleStatusMutation.mutate({
                              id: channel.id,
                              status: channel.status,
                            })
                          }
                          aria-label={t('Toggle channel')}
                        />
                      </div>
                    </CardHeader>
                    <CardContent className='space-y-3 text-xs pb-3'>
                      <div className='flex items-center gap-2 flex-wrap'>
                        <Badge variant='outline'>
                          {t('Group: {{group}}', { group: channel.group || 'default' })}
                        </Badge>
                        <Badge variant='outline'>
                          {t('Priority: {{p}}', { p: channel.priority || 0 })}
                        </Badge>
                        {channel.response_time > 0 && (
                          <Badge variant='secondary' className='text-emerald-600'>
                            {channel.response_time}ms
                          </Badge>
                        )}
                      </div>

                      <div className='space-y-1'>
                        <span className='text-muted-foreground font-medium'>
                          {t('Pool Models:')}
                        </span>
                        <p className='font-mono text-muted-foreground line-clamp-2 text-[11px] bg-muted/40 p-1.5 rounded'>
                          {channel.models || t('All Models')}
                        </p>
                      </div>
                    </CardContent>
                    <CardFooter className='border-t pt-2 pb-2 px-4 flex justify-between bg-muted/10'>
                      <Button
                        size='sm'
                        variant='ghost'
                        className='h-8 text-xs'
                        onClick={() => testChannelMutation.mutate(channel.id)}
                        disabled={testChannelMutation.isPending}
                      >
                        <TestTube className='mr-1.5 h-3.5 w-3.5' />
                        {t('Test')}
                      </Button>
                      <Button
                        size='sm'
                        variant='ghost'
                        className='h-8 text-xs text-destructive hover:text-destructive'
                        onClick={() => {
                          if (confirm(t('Are you sure you want to remove this channel from the pool?'))) {
                            deleteChannelMutation.mutate(channel.id)
                          }
                        }}
                      >
                        <Trash2 className='mr-1.5 h-3.5 w-3.5' />
                        {t('Delete')}
                      </Button>
                    </CardFooter>
                  </Card>
                ))}
              </div>
            )}
          </div>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
