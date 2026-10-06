import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'

import { Studio } from '@/features/studio'

const studioSearchSchema = z.object({
  tool: z.string().optional(),
  tab: z.enum(['catalog', 'playground', 'history']).optional(),
})

export const Route = createFileRoute('/_authenticated/studio/')({
  validateSearch: studioSearchSchema,
  component: RouteComponent,
})

function RouteComponent() {
  const { tool, tab } = Route.useSearch()
  return <Studio initialTool={tool} initialTab={tab} />
}
