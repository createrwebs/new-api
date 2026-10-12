import { expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { recordStudioAttribution } from '../api'

it.each(['tool_visit', 'generate_click', 'repeat_generation'] as const)(
  'records the %s product studio milestone using the backend event name',
  async (eventType) => {
    const post = vi
      .spyOn(api, 'post')
      .mockResolvedValue({ data: { success: true } })

    await recordStudioAttribution(eventType, 'product-photo', 40)

    expect(post).toHaveBeenCalledWith('/api/studio/attribution', {
      event_type: eventType,
      tool_id: 'product-photo',
      credits: 40,
      session_id: undefined,
    })
  }
)

it('keeps generation usable when attribution fails', async () => {
  vi.spyOn(api, 'post').mockRejectedValue(new Error('Network unavailable'))

  await expect(
    recordStudioAttribution('generate_click', 'product-photo')
  ).resolves.toBeUndefined()
})
