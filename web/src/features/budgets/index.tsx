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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { ColumnDef } from '@tanstack/react-table'
import { Plus } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import {
  DataTablePage,
  DISABLED_ROW_DESKTOP,
  DISABLED_ROW_MOBILE,
  useDataTable,
} from '@/components/data-table'
import { ErrorState } from '@/components/error-state'
import { SectionPageLayout } from '@/components/layout'
import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'
import { getServerErrorMessage } from '@/lib/server-error-message'

import {
  createBudgetRule,
  disableBudgetRule,
  getBudgetRules,
  updateBudgetRule,
  type BudgetRule,
  type BudgetRulePayload,
} from './api'
import { BudgetRuleDialog } from './components/budget-rule-dialog'

export function Budgets() {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const queryClient = useQueryClient()
  const [editing, setEditing] = useState<BudgetRule | null | 'create'>(null)
  const [disabling, setDisabling] = useState<BudgetRule | null>(null)
  const rulesQuery = useQuery({
    queryKey: ['admin-budget-rules'],
    queryFn: getBudgetRules,
    retry: false,
    meta: { errorToast: false },
  })
  const saveMutation = useMutation({
    mutationFn: (payload: BudgetRulePayload) =>
      editing && editing !== 'create'
        ? updateBudgetRule(editing.id, payload)
        : createBudgetRule(payload),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['admin-budget-rules'] })
      toast.success(
        t(editing === 'create' ? 'Create succeeded' : 'Update succeeded')
      )
    },
    meta: { errorToast: false },
  })
  const disableMutation = useMutation({
    mutationFn: disableBudgetRule,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['admin-budget-rules'] })
      toast.success(t('Has been disabled'))
      setDisabling(null)
    },
    onError: (error) => handleServerError(error, t('Request failed')),
    meta: { errorToast: false },
  })

  const columns = useMemo<ColumnDef<BudgetRule>[]>(
    () => [
      {
        id: 'scope',
        header: t('Scope type'),
        meta: { mobileTitle: true },
        cell: ({ row }) => (
          <span className='font-medium'>
            {t(row.original.scope_type === 'user' ? 'User' : 'Token')} #
            {formatNumber(row.original.scope_id, locale)}
          </span>
        ),
      },
      {
        accessorKey: 'scope_id',
        header: t('Scope ID'),
        meta: { mobileHidden: true },
        cell: ({ row }) => formatNumber(row.original.scope_id, locale),
      },
      {
        accessorKey: 'period',
        header: t('Period'),
        cell: ({ row }) =>
          t(row.original.period === 'daily' ? 'Daily' : 'Monthly'),
      },
      {
        accessorKey: 'limit_quota',
        header: t('Limit (quota points)'),
        cell: ({ row }) => formatNumber(row.original.limit_quota, locale),
      },
      {
        accessorKey: 'used_quota',
        header: t('Used'),
        cell: ({ row }) => formatNumber(row.original.used_quota, locale),
      },
      {
        accessorKey: 'remaining_quota',
        header: t('Remaining'),
        cell: ({ row }) => formatNumber(row.original.remaining_quota, locale),
      },
      {
        id: 'percentage',
        header: t('Usage %'),
        cell: ({ row }) => {
          const percent =
            row.original.limit_quota > 0
              ? (row.original.used_quota / row.original.limit_quota) * 100
              : 0
          return (
            <div className='min-w-24 space-y-1'>
              <span
                className={percent > 100 ? 'text-destructive font-medium' : ''}
              >
                {formatNumber(percent, locale)}%
              </span>
              <Progress
                value={Math.min(100, Math.max(0, percent))}
                aria-label={t('Usage %')}
              />
            </div>
          )
        },
      },
      {
        accessorKey: 'period_start',
        header: t('Period start'),
        meta: { mobileHidden: true },
        cell: ({ row }) =>
          new Intl.DateTimeFormat(locale, {
            dateStyle: 'short',
            timeStyle: 'short',
          }).format(new Date(row.original.period_start)),
      },
      {
        accessorKey: 'enabled',
        header: t('Status'),
        meta: { mobileBadge: true },
        cell: ({ row }) => (
          <StatusBadge
            label={t(row.original.enabled ? 'Enabled' : 'Disabled')}
            variant={row.original.enabled ? 'success' : 'neutral'}
            copyable={false}
          />
        ),
      },
      {
        id: 'actions',
        header: t('Actions'),
        meta: { pinned: 'right' },
        cell: ({ row }) => (
          <div className='flex gap-1'>
            <Button
              size='sm'
              variant='ghost'
              onClick={() => setEditing(row.original)}
            >
              {t('Edit')}
            </Button>
            {row.original.enabled && (
              <Button
                size='sm'
                variant='ghost'
                className='text-destructive'
                onClick={() => setDisabling(row.original)}
              >
                {t('Disable')}
              </Button>
            )}
          </div>
        ),
      },
    ],
    [t, locale]
  )
  const rules = rulesQuery.data ?? []
  const { table } = useDataTable({
    data: rules,
    columns,
    withFacetedRowModel: false,
  })

  return (
    <>
      <SectionPageLayout fixedContent>
        <SectionPageLayout.Title>{t('Budgets')}</SectionPageLayout.Title>
        <SectionPageLayout.Actions>
          <Button size='sm' onClick={() => setEditing('create')}>
            <Plus className='size-4' />
            {t('Create budget rule')}
          </Button>
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          {rulesQuery.isError && !rulesQuery.data ? (
            <ErrorState
              description={getServerErrorMessage(
                rulesQuery.error,
                t('Request failed')
              )}
              onRetry={() => void rulesQuery.refetch()}
            />
          ) : (
            <DataTablePage
              table={table}
              columns={columns}
              isLoading={rulesQuery.isLoading}
              isFetching={rulesQuery.isFetching}
              emptyTitle={t('No budget rules yet')}
              emptyDescription={t(
                'Create a rule to set a usage limit for a user or token.'
              )}
              skeletonKeyPrefix='budget-rules-skeleton'
              getRowClassName={(row, ctx) => {
                if (row.original.enabled) return undefined
                return ctx.isMobile ? DISABLED_ROW_MOBILE : DISABLED_ROW_DESKTOP
              }}
            />
          )}
        </SectionPageLayout.Content>
      </SectionPageLayout>
      {editing && (
        <BudgetRuleDialog
          key={editing === 'create' ? 'create' : editing.id}
          rule={editing === 'create' ? null : editing}
          onClose={() => setEditing(null)}
          onSave={(payload) => saveMutation.mutateAsync(payload)}
        />
      )}
      {disabling && (
        <ConfirmDialog
          open
          onOpenChange={(open) =>
            !open && !disableMutation.isPending && setDisabling(null)
          }
          title={t('Confirm disable')}
          desc={t(
            'Disabling this rule preserves its usage history. You can re-enable it by editing the rule.'
          )}
          confirmText={t('Disable')}
          destructive
          isLoading={disableMutation.isPending}
          handleConfirm={() => {
            if (!disableMutation.isPending) disableMutation.mutate(disabling.id)
          }}
        />
      )}
    </>
  )
}
