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

export interface NewsUrlInspection {
  id: number
  post_id: number
  inspection_url: string
  index_verdict: string
  coverage_state: string
  robots_state: string
  indexing_state: string
  google_canonical: string
  user_canonical: string
  referring_urls: string
  last_crawled_time: string
  inspection_time: string
}

export interface DevToAnalytics {
  views: number
  comments: number
  reactions: number
  url: string
  published_at: string
}

export interface NewsSeoExperiment {
  id: number
  post_id: number
  experiment_type: string
  field_name: string
  before_value: string
  after_value: string
  hypothesis: string
  status: string
  result_verdict: string
  created_at: number
}

export interface NewsGrowthOpportunity {
  post_id: number
  slug: string
  type: string
  score: number
  reason: string
  action_summary: string
  recommended_change?: string
}

export interface RecommendedLink {
  target_url: string
  title: string
  anchor_text: string
  context: string
}

export interface GrowthTimelineEvent {
  timestamp: string
  event_type: string
  description: string
  details?: Record<string, unknown>
}

export interface NewsDistributionRecord {
  platform: string
  status: string
  remote_id: string
  remote_url: string
  error_message?: string
  published_at?: number
}

export interface PostGrowthRecord {
  post_id: number
  title: string
  slug: string
  status: string
  published_at: number
  canonical_url: string
  view_count: number
  average_position: number
  search_clicks: number
  search_impressions: number
  url_inspection?: NewsUrlInspection
  distributions: NewsDistributionRecord[]
  devto_analytics?: DevToAnalytics
  experiments: NewsSeoExperiment[]
  opportunities: NewsGrowthOpportunity[]
  internal_links: RecommendedLink[]
  timeline: GrowthTimelineEvent[]
}

export interface GlobalGrowthOverview {
  total_published_posts: number
  total_organic_views: number
  gsc_status: string
  gsc_data_available: boolean
  gsc_row_count: number
  gsc_site_url: string
  mass_autopublish: boolean
  devto_status: string
  devto_update_policy: string
  active_experiments: number
  daily_reviews_count: number
  recent_opportunities: NewsGrowthOpportunity[]
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
