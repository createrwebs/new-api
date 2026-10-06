import { useState, useEffect, useCallback } from 'react'
import {
  ExternalLink,
  RefreshCw,
  Trash2,
  Globe,
  FileText,
  Search,
  TrendingUp,
  ShieldCheck,
  SearchCheck,
  Sparkles,
  Link as LinkIcon,
  Activity,
} from 'lucide-react'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import {
  getAdminNewsPosts,
  updateAdminNewsPost,
  deleteAdminNewsPost,
  triggerAdminNewsScout,
  getAdminGrowthOverview,
  getAdminPostGrowthRecord,
  inspectAdminPostURL,
  triggerAdminGrowthSync,
} from './api'
import type {
  NewsPost,
  GlobalGrowthOverview,
  PostGrowthRecord,
} from './types'

export function NewsAdmin() {
  const { t } = useTranslation()
  const [posts, setPosts] = useState<NewsPost[]>([])
  const [loading, setLoading] = useState(false)
  const [scouting, setScouting] = useState(false)
  const [syncingGrowth, setSyncingGrowth] = useState(false)
  const [inspectingId, setInspectingId] = useState<number | null>(null)
  const [statusFilter, setStatusFilter] = useState<string>('all')
  const [keyword, setKeyword] = useState('')
  const [total, setTotal] = useState(0)

  // Growth Overview & Record Modal
  const [growthOverview, setGrowthOverview] = useState<GlobalGrowthOverview | null>(null)
  const [selectedRecord, setSelectedRecord] = useState<PostGrowthRecord | null>(null)
  const [growthModalOpen, setGrowthModalOpen] = useState(false)
  const [loadingRecord, setLoadingRecord] = useState(false)

  const fetchOverview = useCallback(async () => {
    try {
      const res = await getAdminGrowthOverview()
      if (res.success && res.data) {
        setGrowthOverview(res.data)
      }
    } catch {
      // Non-blocking for overview
    }
  }, [])

  const fetchPosts = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getAdminNewsPosts({
        status: statusFilter === 'all' ? '' : statusFilter,
        keyword: keyword.trim(),
        page_size: 50,
      })
      if (res.success && Array.isArray(res.data)) {
        setPosts(res.data)
        setTotal(res.total || res.data.length)
      }
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : 'Failed to load posts')
    } finally {
      setLoading(false)
    }
  }, [statusFilter, keyword])

  useEffect(() => {
    fetchPosts()
    fetchOverview()
  }, [fetchPosts, fetchOverview])

  const handleTogglePublish = async (post: NewsPost) => {
    const nextStatus = post.status === 'published' ? 'draft' : 'published'
    try {
      const res = await updateAdminNewsPost(post.id, { status: nextStatus })
      if (res.success) {
        toast.success(
          nextStatus === 'published'
            ? 'Article published successfully'
            : 'Article moved back to draft'
        )
        fetchPosts()
      } else {
        toast.error(res.message || 'Update failed')
      }
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : 'Update failed')
    }
  }

  const handleDelete = async (post: NewsPost) => {
    if (!window.confirm(`Delete post "${post.title}"?`)) {
      return
    }
    try {
      const res = await deleteAdminNewsPost(post.id)
      if (res.success) {
        toast.success('Post deleted successfully')
        fetchPosts()
      } else {
        toast.error(res.message || 'Delete failed')
      }
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : 'Delete failed')
    }
  }

  const handleTriggerScout = async () => {
    setScouting(true)
    try {
      const res = await triggerAdminNewsScout()
      if (res.success) {
        toast.success(
          `Scout sync completed: ${res.data?.sources_processed ?? 0} sources checked, ${res.data?.posts_drafted ?? 0} drafts created`
        )
        fetchPosts()
      } else {
        toast.error(res.message || 'Scout trigger failed')
      }
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : 'Scout trigger failed')
    } finally {
      setScouting(false)
    }
  }

  const handleTriggerGrowthSync = async () => {
    setSyncingGrowth(true)
    try {
      const res = await triggerAdminGrowthSync()
      if (res.success) {
        toast.success('Growth loop iteration triggered')
        fetchOverview()
      } else {
        toast.error(res.message || 'Growth sync failed')
      }
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : 'Growth sync failed')
    } finally {
      setSyncingGrowth(false)
    }
  }

  const handleOpenGrowthRecord = async (post: NewsPost) => {
    setGrowthModalOpen(true)
    setLoadingRecord(true)
    try {
      const res = await getAdminPostGrowthRecord(post.id)
      if (res.success && res.data) {
        setSelectedRecord(res.data)
      } else {
        toast.error(res.message || 'Failed to load growth record')
      }
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : 'Failed to load growth record')
    } finally {
      setLoadingRecord(false)
    }
  }

  const handleInspectPost = async (postId: number) => {
    setInspectingId(postId)
    try {
      const res = await inspectAdminPostURL(postId)
      if (res.success && res.data) {
        toast.success(`URL Inspection complete: verdict is ${res.data.index_verdict || 'DONE'}`)
        const updated = await getAdminPostGrowthRecord(postId)
        if (updated.success && updated.data) {
          setSelectedRecord(updated.data)
        }
      } else {
        toast.error(res.message || 'Inspection failed')
      }
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : 'Inspection failed')
    } finally {
      setInspectingId(null)
    }
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('News & Growth Engine')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => window.open('/news', '_blank')}
          >
            <Globe className="h-4 w-4 mr-1.5" />
            {t('Public News')}
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={handleTriggerGrowthSync}
            disabled={syncingGrowth}
          >
            <Activity className={`h-4 w-4 mr-1.5 ${syncingGrowth ? 'animate-spin' : ''}`} />
            {syncingGrowth ? t('Syncing Loop...') : t('Sync Growth Loop')}
          </Button>
          <Button
            variant="default"
            size="sm"
            onClick={handleTriggerScout}
            disabled={scouting}
          >
            <RefreshCw
              className={`h-4 w-4 mr-1.5 ${scouting ? 'animate-spin' : ''}`}
            />
            {scouting ? t('Syncing...') : t('Sync Scout Feeds')}
          </Button>
        </div>
      </SectionPageLayout.Actions>

      <SectionPageLayout.Content>
        <div className="space-y-6">
          {/* Growth Intelligence Banner */}
          {growthOverview && (
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <Card className="bg-card/60 backdrop-blur-xs border-border/80">
                <CardHeader className="pb-2">
                  <CardTitle className="text-xs font-semibold uppercase text-muted-foreground flex items-center justify-between">
                    <span className="flex items-center gap-1.5">
                      <SearchCheck className="h-4 w-4 text-primary" />
                      Google Search Console
                    </span>
                    <Badge
                      variant={
                        growthOverview.gsc_status === 'DATA_AVAILABLE'
                          ? 'default'
                          : growthOverview.gsc_status === 'CONNECTED'
                            ? 'secondary'
                            : 'destructive'
                      }
                      className="text-[10px] font-mono"
                    >
                      {growthOverview.gsc_status}
                    </Badge>
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-1">
                  <p className="text-xs font-mono text-foreground truncate">
                    {growthOverview.gsc_site_url || 'sc-domain:toraapi.com'}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {growthOverview.gsc_data_available
                      ? `${growthOverview.gsc_row_count} Search Analytics rows`
                      : 'Verified property; analytics warming up (2-3d)'}
                  </p>
                </CardContent>
              </Card>

              <Card className="bg-card/60 backdrop-blur-xs border-border/80">
                <CardHeader className="pb-2">
                  <CardTitle className="text-xs font-semibold uppercase text-muted-foreground flex items-center justify-between">
                    <span className="flex items-center gap-1.5">
                      <TrendingUp className="h-4 w-4 text-emerald-500" />
                      DEV.to Syndication
                    </span>
                    <Badge
                      variant={
                        growthOverview.devto_status === 'ACTIVE'
                          ? 'default'
                          : 'secondary'
                      }
                      className="text-[10px] font-mono bg-emerald-600/90 text-white"
                    >
                      {growthOverview.devto_status}
                    </Badge>
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-1">
                  <p className="text-xs text-foreground font-mono">
                    Policy: {growthOverview.devto_update_policy}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    Allowlist guarded (Post ID 8) · Idempotent updates
                  </p>
                </CardContent>
              </Card>

              <Card className="bg-card/60 backdrop-blur-xs border-border/80">
                <CardHeader className="pb-2">
                  <CardTitle className="text-xs font-semibold uppercase text-muted-foreground flex items-center justify-between">
                    <span className="flex items-center gap-1.5">
                      <ShieldCheck className="h-4 w-4 text-blue-500" />
                      Growth Safety Guards
                    </span>
                    <Badge variant="outline" className="text-[10px] font-mono border-blue-500/40 text-blue-400">
                      SAFE
                    </Badge>
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-1">
                  <p className="text-xs text-foreground">
                    Mass Autopublish: <span className="font-semibold text-amber-400">DISABLED</span>
                  </p>
                  <p className="text-xs text-muted-foreground">
                    Experiments: {growthOverview.active_experiments} · Reviews: {growthOverview.daily_reviews_count}
                  </p>
                </CardContent>
              </Card>
            </div>
          )}

          {/* Filters */}
          <div className="flex flex-col sm:flex-row items-center gap-3">
            <div className="relative w-full sm:w-72">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder={t('Search articles...')}
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
                className="pl-9"
              />
            </div>
            <Select
              value={statusFilter}
              onValueChange={(val) => setStatusFilter(val || 'all')}
            >
              <SelectTrigger className="w-full sm:w-44">
                <SelectValue placeholder={t('Status filter')} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">{t('All Statuses')}</SelectItem>
                <SelectItem value="published">{t('Published')}</SelectItem>
                <SelectItem value="draft">{t('Draft')}</SelectItem>
                <SelectItem value="review_required">
                  {t('Review Required')}
                </SelectItem>
              </SelectContent>
            </Select>
            <span className="text-xs text-muted-foreground ml-auto">
              {t('Total articles')}: {total}
            </span>
          </div>

          {/* Posts Table */}
          <div className="rounded-md border bg-card">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-16">ID</TableHead>
                  <TableHead>{t('Title & Slug')}</TableHead>
                  <TableHead className="w-32">{t('Status')}</TableHead>
                  <TableHead className="w-28">{t('Type')}</TableHead>
                  <TableHead className="w-36">{t('Published')}</TableHead>
                  <TableHead className="w-20 text-right">{t('Views')}</TableHead>
                  <TableHead className="w-56 text-right">{t('Actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow>
                    <TableCell colSpan={7} className="h-24 text-center">
                      <div className="flex items-center justify-center gap-2 text-muted-foreground">
                        <RefreshCw className="h-4 w-4 animate-spin" />
                        {t('Loading articles...')}
                      </div>
                    </TableCell>
                  </TableRow>
                ) : posts.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={7} className="h-24 text-center text-muted-foreground">
                      <div className="flex flex-col items-center justify-center gap-1">
                        <FileText className="h-8 w-8 text-muted-foreground/50 mb-1" />
                        <p>{t('No articles found.')}</p>
                      </div>
                    </TableCell>
                  </TableRow>
                ) : (
                  posts.map((post) => (
                    <TableRow key={post.id}>
                      <TableCell className="font-mono text-xs text-muted-foreground">
                        {post.id}
                      </TableCell>
                      <TableCell>
                        <div className="space-y-1">
                          <p className="font-medium text-sm leading-snug line-clamp-1">
                            {post.title}
                          </p>
                          <p className="text-xs text-muted-foreground font-mono">
                            /news/{post.slug}
                          </p>
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge
                          variant={
                            post.status === 'published'
                              ? 'default'
                              : post.status === 'review_required'
                                ? 'destructive'
                                : 'secondary'
                          }
                          className="capitalize text-xs font-semibold"
                        >
                          {post.status}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <span className="text-xs text-muted-foreground font-mono uppercase">
                          {post.content_type || 'news'}
                        </span>
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {post.published_at > 0
                          ? new Date(post.published_at * 1000).toLocaleDateString()
                          : '—'}
                      </TableCell>
                      <TableCell className="text-right text-xs font-mono">
                        {post.view_count || 0}
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex items-center justify-end gap-1.5">
                          <Button
                            variant="outline"
                            size="sm"
                            className="h-8 text-xs font-medium gap-1"
                            onClick={() => handleOpenGrowthRecord(post)}
                          >
                            <TrendingUp className="h-3.5 w-3.5 text-primary" />
                            Growth
                          </Button>
                          {post.status === 'published' && (
                            <Button
                              variant="ghost"
                              size="icon"
                              className="h-8 w-8"
                              title={t('View Public Page')}
                              onClick={() =>
                                window.open(`/news/${post.slug}`, '_blank')
                              }
                            >
                              <ExternalLink className="h-3.5 w-3.5" />
                            </Button>
                          )}
                          <Button
                            variant="outline"
                            size="sm"
                            className="h-8 text-xs"
                            onClick={() => handleTogglePublish(post)}
                          >
                            {post.status === 'published'
                              ? t('Unpublish')
                              : t('Publish')}
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-8 w-8 text-destructive hover:text-destructive"
                            title={t('Delete')}
                            onClick={() => handleDelete(post)}
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>
        </div>

        {/* Growth Record Detail Dialog */}
        <Dialog open={growthModalOpen} onOpenChange={setGrowthModalOpen}>
          <DialogContent className="max-w-2xl max-h-[85vh] overflow-y-auto">
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2 text-base">
                <Sparkles className="h-5 w-5 text-primary" />
                Post Growth Intelligence — #{selectedRecord?.post_id}
              </DialogTitle>
              <DialogDescription className="truncate">
                {selectedRecord?.title}
              </DialogDescription>
            </DialogHeader>

            {loadingRecord ? (
              <div className="py-8 flex items-center justify-center gap-2 text-muted-foreground text-sm">
                <RefreshCw className="h-4 w-4 animate-spin" />
                Loading growth intelligence...
              </div>
            ) : selectedRecord ? (
              <div className="space-y-5 text-sm">
                {/* Search Performance Stats */}
                <div className="grid grid-cols-3 gap-3 p-3 bg-muted/40 rounded-lg text-center">
                  <div>
                    <span className="text-xs text-muted-foreground block">Organic Views</span>
                    <span className="text-lg font-bold font-mono">{selectedRecord.view_count}</span>
                  </div>
                  <div>
                    <span className="text-xs text-muted-foreground block">Search Clicks</span>
                    <span className="text-lg font-bold font-mono">{selectedRecord.search_clicks}</span>
                  </div>
                  <div>
                    <span className="text-xs text-muted-foreground block">Avg Position</span>
                    <span className="text-lg font-bold font-mono">
                      {selectedRecord.average_position > 0 ? selectedRecord.average_position.toFixed(1) : '—'}
                    </span>
                  </div>
                </div>

                {/* Google Search Console URL Inspection */}
                <div className="border rounded-lg p-3.5 space-y-2">
                  <div className="flex items-center justify-between">
                    <span className="font-semibold text-xs uppercase text-muted-foreground flex items-center gap-1.5">
                      <SearchCheck className="h-4 w-4 text-primary" />
                      Google URL Inspection
                    </span>
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-7 text-xs"
                      onClick={() => handleInspectPost(selectedRecord.post_id)}
                      disabled={inspectingId === selectedRecord.post_id}
                    >
                      <RefreshCw className={`h-3 w-3 mr-1 ${inspectingId === selectedRecord.post_id ? 'animate-spin' : ''}`} />
                      {inspectingId === selectedRecord.post_id ? 'Inspecting...' : 'Inspect URL'}
                    </Button>
                  </div>
                  {selectedRecord.url_inspection ? (
                    <div className="space-y-1 text-xs font-mono bg-muted/30 p-2.5 rounded">
                      <div className="flex justify-between">
                        <span className="text-muted-foreground">Verdict:</span>
                        <span className="font-semibold">{selectedRecord.url_inspection.index_verdict || 'UNKNOWN'}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-muted-foreground">Coverage State:</span>
                        <span>{selectedRecord.url_inspection.coverage_state || '—'}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-muted-foreground">Robots State:</span>
                        <span>{selectedRecord.url_inspection.robots_state || '—'}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-muted-foreground">User Canonical:</span>
                        <span className="truncate max-w-[280px]">{selectedRecord.url_inspection.user_canonical || '—'}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-muted-foreground">Google Canonical:</span>
                        <span className="truncate max-w-[280px]">{selectedRecord.url_inspection.google_canonical || '—'}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-muted-foreground">Inspected At:</span>
                        <span>{selectedRecord.url_inspection.inspection_time || '—'}</span>
                      </div>
                    </div>
                  ) : (
                    <p className="text-xs text-muted-foreground">
                      No URL inspection record cached yet. Click Inspect URL to query Google Search Console.
                    </p>
                  )}
                </div>

                {/* DEV.to Syndication Analytics */}
                <div className="border rounded-lg p-3.5 space-y-2">
                  <span className="font-semibold text-xs uppercase text-muted-foreground flex items-center gap-1.5">
                    <TrendingUp className="h-4 w-4 text-emerald-500" />
                    DEV.to Distribution
                  </span>
                  {selectedRecord.devto_analytics ? (
                    <div className="space-y-1.5 text-xs">
                      <div className="grid grid-cols-3 gap-2 p-2 bg-muted/30 rounded text-center font-mono">
                        <div>
                          <span className="text-muted-foreground text-[10px] block">Views</span>
                          <span className="font-bold">{selectedRecord.devto_analytics.views}</span>
                        </div>
                        <div>
                          <span className="text-muted-foreground text-[10px] block">Reactions</span>
                          <span className="font-bold">{selectedRecord.devto_analytics.reactions}</span>
                        </div>
                        <div>
                          <span className="text-muted-foreground text-[10px] block">Comments</span>
                          <span className="font-bold">{selectedRecord.devto_analytics.comments}</span>
                        </div>
                      </div>
                      {selectedRecord.devto_analytics.url && (
                        <a
                          href={selectedRecord.devto_analytics.url}
                          target="_blank"
                          rel="noreferrer"
                          className="inline-flex items-center gap-1 text-primary hover:underline font-mono text-[11px]"
                        >
                          <ExternalLink className="h-3 w-3" />
                          {selectedRecord.devto_analytics.url}
                        </a>
                      )}
                    </div>
                  ) : (
                    <p className="text-xs text-muted-foreground">
                      No DEV.to live analytics found for this article.
                    </p>
                  )}
                </div>

                {/* Internal Links Recommendations */}
                {selectedRecord.internal_links && selectedRecord.internal_links.length > 0 && (
                  <div className="border rounded-lg p-3.5 space-y-2">
                    <span className="font-semibold text-xs uppercase text-muted-foreground flex items-center gap-1.5">
                      <LinkIcon className="h-4 w-4 text-blue-500" />
                      Recommended Internal Links
                    </span>
                    <ul className="space-y-1.5 text-xs">
                      {selectedRecord.internal_links.map((link, idx) => (
                        <li key={idx} className="p-2 bg-muted/30 rounded flex justify-between items-center">
                          <div>
                            <span className="font-medium text-foreground block">{link.anchor_text}</span>
                            <span className="text-muted-foreground text-[11px] font-mono">{link.target_url}</span>
                          </div>
                          <Badge variant="outline" className="text-[10px]">{link.context}</Badge>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>
            ) : null}
          </DialogContent>
        </Dialog>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
