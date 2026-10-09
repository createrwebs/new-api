import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { AuthRequiredModal } from '../components/AuthRequiredModal'

const mockNavigate = vi.fn()

vi.mock('@tanstack/react-router', () => ({
  useNavigate: () => mockNavigate,
}))

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, opts?: Record<string, unknown>) => {
      if (opts?.tool) return `เพื่อใช้งาน ${opts.tool} และสร้างผลงานด้วย AI`
      return key
    },
  }),
}))

describe('AuthRequiredModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders modal with title and action buttons when open', () => {
    render(
      <AuthRequiredModal
        open={true}
        onOpenChange={vi.fn()}
        toolName='AI Image Generate'
      />
    )

    expect(screen.getByText('เข้าสู่ระบบหรือสมัครสมาชิก')).toBeInTheDocument()
    expect(screen.getByText('เพื่อใช้งาน AI Image Generate และสร้างผลงานด้วย AI')).toBeInTheDocument()
    expect(screen.getByText('สมัครสมาชิกใหม่')).toBeInTheDocument()
    expect(screen.getByText('เข้าสู่ระบบ')).toBeInTheDocument()
    expect(screen.getByText('ไว้คราวหน้า')).toBeInTheDocument()
  })

  it('navigates to /sign-up with redirect when sign-up button is clicked', async () => {
    const user = userEvent.setup()
    const handleOpenChange = vi.fn()

    render(
      <AuthRequiredModal
        open={true}
        onOpenChange={handleOpenChange}
        toolName='AI Image Generate'
      />
    )

    const signUpBtn = screen.getByRole('button', { name: /สมัครสมาชิกใหม่/i })
    await user.click(signUpBtn)

    expect(handleOpenChange).toHaveBeenCalledWith(false)
    expect(mockNavigate).toHaveBeenCalledWith(
      expect.objectContaining({
        to: '/sign-up',
        search: expect.objectContaining({
          redirect: expect.any(String),
        }),
      })
    )
  })

  it('navigates to /sign-in with redirect when sign-in button is clicked', async () => {
    const user = userEvent.setup()
    const handleOpenChange = vi.fn()

    render(
      <AuthRequiredModal
        open={true}
        onOpenChange={handleOpenChange}
        toolName='AI Image Generate'
      />
    )

    const signInBtn = screen.getByRole('button', { name: /เข้าสู่ระบบ/i })
    await user.click(signInBtn)

    expect(handleOpenChange).toHaveBeenCalledWith(false)
    expect(mockNavigate).toHaveBeenCalledWith(
      expect.objectContaining({
        to: '/sign-in',
        search: expect.objectContaining({
          redirect: expect.any(String),
        }),
      })
    )
  })
})
