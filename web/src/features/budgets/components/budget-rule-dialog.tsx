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
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { NativeSelect } from '@/components/ui/native-select'
import { Switch } from '@/components/ui/switch'
import { getServerErrorMessage } from '@/lib/server-error-message'

import type { BudgetRule, BudgetRulePayload } from '../api'
import { budgetRuleSchema, type BudgetRuleFormValues } from '../lib/schema'

type Props = {
  rule: BudgetRule | null
  onClose: () => void
  onSave: (payload: BudgetRulePayload) => Promise<void>
}

export function BudgetRuleDialog(props: Props) {
  const { t } = useTranslation()
  const form = useForm<BudgetRuleFormValues>({
    resolver: zodResolver(budgetRuleSchema(t('Enter a positive integer'))),
    defaultValues: {
      scope_type: props.rule?.scope_type ?? 'user',
      scope_id: props.rule?.scope_id ?? 0,
      period: props.rule?.period ?? 'daily',
      limit_quota: props.rule?.limit_quota ?? 0,
      enabled: props.rule?.enabled ?? true,
    },
  })
  const saving = form.formState.isSubmitting

  async function onSubmit(values: BudgetRuleFormValues) {
    try {
      await props.onSave(values)
      props.onClose()
    } catch (error) {
      form.setError('root', {
        message: getServerErrorMessage(error, t('Request failed')),
      })
    }
  }

  return (
    <Dialog
      open
      onOpenChange={(open) => !open && !saving && props.onClose()}
      title={props.rule ? t('Edit Rule') : t('Create budget rule')}
      description={
        props.rule
          ? t('Scope and period cannot be changed after creation.')
          : undefined
      }
      footer={
        <>
          <Button variant='outline' disabled={saving} onClick={props.onClose}>
            {t('Cancel')}
          </Button>
          <Button form='budget-rule-form' type='submit' disabled={saving}>
            {props.rule ? t('Save') : t('Create')}
          </Button>
        </>
      }
    >
      <Form {...form}>
        <form
          id='budget-rule-form'
          onSubmit={form.handleSubmit(onSubmit)}
          className='space-y-4'
        >
          <FormField
            control={form.control}
            name='scope_type'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Scope type')}</FormLabel>
                <FormControl>
                  <NativeSelect
                    className='w-full'
                    value={field.value}
                    onChange={field.onChange}
                    onBlur={field.onBlur}
                    disabled={!!props.rule || saving}
                    aria-label={t('Scope type')}
                  >
                    <option value='user'>{t('User')}</option>
                    <option value='token'>{t('Token')}</option>
                  </NativeSelect>
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name='scope_id'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Scope ID')}</FormLabel>
                <FormControl>
                  <Input
                    type='number'
                    min={1}
                    step={1}
                    value={field.value || ''}
                    onChange={(event) =>
                      field.onChange(
                        event.target.value === ''
                          ? 0
                          : Number(event.target.value)
                      )
                    }
                    onBlur={field.onBlur}
                    disabled={!!props.rule || saving}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name='period'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Period')}</FormLabel>
                <FormControl>
                  <NativeSelect
                    className='w-full'
                    value={field.value}
                    onChange={field.onChange}
                    onBlur={field.onBlur}
                    disabled={!!props.rule || saving}
                    aria-label={t('Period')}
                  >
                    <option value='daily'>{t('Daily')}</option>
                    <option value='monthly'>{t('Monthly')}</option>
                  </NativeSelect>
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name='limit_quota'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Limit (quota points)')}</FormLabel>
                <FormControl>
                  <Input
                    type='number'
                    min={1}
                    step={1}
                    value={field.value || ''}
                    onChange={(event) =>
                      field.onChange(
                        event.target.value === ''
                          ? 0
                          : Number(event.target.value)
                      )
                    }
                    onBlur={field.onBlur}
                    disabled={saving}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name='enabled'
            render={({ field }) => (
              <FormItem className='flex items-center justify-between gap-4 rounded-lg border p-3'>
                <FormLabel>{t('Enabled')}</FormLabel>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                    disabled={saving}
                    aria-label={t('Enabled')}
                  />
                </FormControl>
              </FormItem>
            )}
          />
          {form.formState.errors.root?.message && (
            <p role='alert' className='text-destructive text-sm'>
              {form.formState.errors.root.message}
            </p>
          )}
        </form>
      </Form>
    </Dialog>
  )
}
