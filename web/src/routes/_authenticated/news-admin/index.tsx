import { createFileRoute, redirect } from '@tanstack/react-router'
import z from 'zod'

import { NewsAdmin } from '@/features/news'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

const newsSearchSchema = z.object({
  page: z.number().optional().catch(1),
  pageSize: z.number().optional().catch(undefined),
  status: z.string().optional().catch(''),
  keyword: z.string().optional().catch(''),
})

export const Route = createFileRoute('/_authenticated/news-admin/')({
  beforeLoad: () => {
    const { auth } = useAuthStore.getState()

    if (!auth.user || auth.user.role < ROLE.ADMIN) {
      throw redirect({
        to: '/403',
      })
    }
  },
  validateSearch: newsSearchSchema,
  component: NewsAdmin,
})
