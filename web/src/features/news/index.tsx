import { useState, useEffect, useCallback } from 'react'
import {
  ExternalLink,
  RefreshCw,
  Trash2,
  Globe,
  FileText,
  Search,
} from 'lucide-react'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
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
} from './api'
import type { NewsPost } from './types'

export function NewsAdmin() {
  const { t } = useTranslation()
  const [posts, setPosts] = useState<NewsPost[]>([])
  const [loading, setLoading] = useState(false)
  const [scouting, setScouting] = useState(false)
  const [statusFilter, setStatusFilter] = useState<string>('all')
  const [keyword, setKeyword] = useState('')
  const [total, setTotal] = useState(0)

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
  }, [fetchPosts])

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
        <div className="space-y-4">
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
                  <TableHead className="w-48 text-right">{t('Actions')}</TableHead>
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
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
