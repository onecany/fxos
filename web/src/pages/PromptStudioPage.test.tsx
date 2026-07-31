import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { PromptStudioPage } from './PromptStudioPage'

// Mock auth: logged in with a token so the page attempts to load strategies.
vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({
    token: 'test-token',
    isLoggedIn: true,
  }),
}))

// Mock language context: default to English for stable assertions.
vi.mock('../contexts/LanguageContext', () => ({
  useLanguage: () => ({ language: 'en' }),
}))

// Mock the strategy API so no network call happens in jsdom.
vi.mock('../lib/api/strategies', () => ({
  strategyApi: {
    getStrategies: vi.fn().mockResolvedValue([]),
    getStrategy: vi.fn(),
    getActiveStrategy: vi.fn(),
    getDefaultStrategyConfig: vi.fn(),
    createStrategy: vi.fn(),
    updateStrategy: vi.fn(),
    deleteStrategy: vi.fn(),
    activateStrategy: vi.fn(),
    duplicateStrategy: vi.fn(),
  },
}))

// Mock httpClient used by preview/estimate calls.
vi.mock('../lib/api/helpers', () => ({
  API_BASE: '/api',
  httpClient: {
    post: vi.fn().mockResolvedValue({
      success: true,
      data: { system_prompt: 'test prompt', prompt_variant: 'balanced' },
      message: '',
    }),
  },
}))

/**
 * PromptStudioPage smoke tests.
 *
 * The page is the standalone Prompt Studio editor for the AI system prompt.
 * These tests verify it renders without crashing in the empty state
 * (no strategies yet) and that the i18n keys resolve for the default language.
 */
describe('PromptStudioPage', () => {
  const renderPage = () =>
    render(
      <MemoryRouter>
        <PromptStudioPage />
      </MemoryRouter>
    )

  describe('Rendering', () => {
    it('should render the page without errors', () => {
      const { container } = renderPage()
      expect(container).toBeTruthy()
    })

    it('should show the Prompt Studio title', () => {
      renderPage()
      expect(screen.getAllByText(/prompt studio/i).length).toBeGreaterThan(0)
    })

    it('should show the empty-state hint when no strategies exist', async () => {
      renderPage()
      expect(await screen.findAllByText(/no strategy selected/i)).toBeTruthy()
    })
  })
})
