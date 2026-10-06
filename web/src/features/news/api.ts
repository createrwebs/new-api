import { api } from '@/lib/api'

import type {
  ApiResponse,
  DailyGrowthReview,
  GetNewsPostsParams,
  GetNewsPostsResponse,
  NewsPost,
  NewsSource,
} from './types'

export async function getAdminNewsPosts(
  params: GetNewsPostsParams = {}
): Promise<GetNewsPostsResponse> {
  const { p = 1, page_size = 20, status = '', keyword = '' } = params
  const queryParams = new URLSearchParams()
  queryParams.set('p', String(p))
  queryParams.set('page_size', String(page_size))
  if (status) queryParams.set('status', status)
  if (keyword) queryParams.set('keyword', keyword)

  const res = await api.get(`/api/admin/news/posts?${queryParams.toString()}`)
  return res.data
}

export async function updateAdminNewsPost(
  id: number,
  data: Partial<NewsPost>
): Promise<ApiResponse<NewsPost>> {
  const res = await api.put(`/api/admin/news/posts/${id}`, data)
  return res.data
}

export async function deleteAdminNewsPost(
  id: number
): Promise<ApiResponse<null>> {
  const res = await api.delete(`/api/admin/news/posts/${id}`)
  return res.data
}

export async function getAdminNewsSources(): Promise<ApiResponse<NewsSource[]>> {
  const res = await api.get('/api/admin/news/sources')
  return res.data
}

export async function triggerAdminNewsScout(): Promise<ApiResponse<{ sources_processed: number; posts_drafted: number }>> {
  const res = await api.post('/api/admin/news/scout/trigger')
  return res.data
}

export async function getAdminDailyGrowthReviews(): Promise<ApiResponse<DailyGrowthReview[]>> {
  const res = await api.get('/api/admin/news/reviews/daily')
  return res.data
}
