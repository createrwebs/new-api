import { z } from 'zod'

// JSON numbers above MAX_SAFE_INTEGER lose precision before reaching the API.
export function budgetRuleSchema(message: string) {
  const positiveInteger = z
    .number({ error: message })
    .int({ error: message })
    .positive({ error: message })
    .max(Number.MAX_SAFE_INTEGER, { error: message })

  return z.object({
    scope_type: z.enum(['user', 'token']),
    scope_id: positiveInteger,
    period: z.enum(['daily', 'monthly']),
    limit_quota: positiveInteger,
    enabled: z.boolean(),
  })
}

export type BudgetRuleFormValues = z.infer<ReturnType<typeof budgetRuleSchema>>
