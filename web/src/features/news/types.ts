export interface NewsPost {
  id: number
  slug: string
  content_type: string
  title: string
  summary: string
  content_markdown: string
  content_html: string
  author_name: string
  canonical_url: string
  status: 'draft' | 'review_required' | 'scheduled' | 'published' | 'archived'
  content_risk: 'low' | 'medium' | 'high'
  fact_check_status: 'pending' | 'verified' | 'disputed'
  fact_check_notes: string
  seo_title: string
  seo_description: string
  seo_keywords: string
  og_image_url: string
  is_seed: boolean
  published_at: number
  view_count: number
  created_at: number
  updated_at: number
}

export interface NewsSource {
  id: number
  name: string
  slug: string
  feed_url: string
  feed_type: string
  trust_tier: string
  is_enabled: boolean
  polling_interval_minutes: number
  last_fetched_at: number
  fetch_error_count: number
  last_error: string
  tags: string
}

export interface DailyGrowthReview {
  id: number
  review_date: string
  stories_discovered: number
  posts_published: number
  pageviews_24h: number
  conversions_24h: number
  top_referrers: string
  growth_recommendations: string
  created_at: number
}

export interface GetNewsPostsParams {
  p?: number
  page_size?: number
  status?: string
  keyword?: string
}

export interface GetNewsPostsResponse {
  success: boolean
  data: NewsPost[]
  total: number
  message?: string
}

export interface ApiResponse<T = unknown> {
  success: boolean
  data?: T
  message?: string
}
