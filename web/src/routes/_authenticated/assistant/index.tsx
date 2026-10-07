import { createFileRoute } from '@tanstack/react-router'

import { Studio } from '@/features/studio'

export const Route = createFileRoute('/_authenticated/assistant/')({
  component: RouteComponent,
})

function RouteComponent() {
  return <Studio initialTab='assistant' />
}
