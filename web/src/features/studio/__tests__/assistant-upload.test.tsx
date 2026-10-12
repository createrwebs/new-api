import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it, vi } from 'vitest'

import { CreatorAssistant } from '../components/CreatorAssistant'

afterEach(() => vi.unstubAllGlobals())

it('opens the file picker when the upload button is activated with the keyboard', async () => {
  const user = userEvent.setup()
  const { container } = render(<CreatorAssistant />)
  const input = container.querySelector<HTMLInputElement>('input[type="file"]')
  if (!input) throw new Error('Missing file input')
  const openPicker = vi.spyOn(input, 'click')

  screen.getByRole('button', { name: 'เลือกไฟล์รูปภาพ' }).focus()
  await user.keyboard('{Enter}')

  expect(openPicker).toHaveBeenCalledOnce()
})

it('disables the file picker while an upload is pending and restores it after failure', async () => {
  const user = userEvent.setup()
  let resolveUpload!: (response: Response) => void
  vi.stubGlobal(
    'fetch',
    vi.fn(
      () =>
        new Promise<Response>((resolve) => {
          resolveUpload = resolve
        })
    )
  )
  const { container } = render(<CreatorAssistant />)
  const input = container.querySelector<HTMLInputElement>('input[type="file"]')
  if (!input) throw new Error('Missing file input')
  const uploadButton = screen.getByRole('button', { name: 'เลือกไฟล์รูปภาพ' })

  await user.upload(
    input,
    new File(['image'], 'product.png', { type: 'image/png' })
  )

  expect(uploadButton).toBeDisabled()
  expect(input).toBeDisabled()

  await act(async () =>
    resolveUpload(new Response(JSON.stringify({ success: false })))
  )

  expect(uploadButton).toBeEnabled()
  expect(input).toBeEnabled()
})
