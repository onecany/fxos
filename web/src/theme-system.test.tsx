import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { ThemeProvider, useTheme } from './contexts/ThemeContext'
import { LanguageProvider } from './contexts/LanguageContext'
import { AuthProvider } from './contexts/AuthContext'
import HeaderBar from './components/common/HeaderBar'

// Theme toggle behavior — no app chrome needed
function ToggleHarness() {
  const { theme, toggleTheme } = useTheme()
  return (
    <button onClick={toggleTheme} data-testid="theme-toggle">
      {theme}
    </button>
  )
}

describe('theme system', () => {
  beforeEach(() => {
    window.localStorage.clear()
    document.documentElement.removeAttribute('data-theme')
  })

  it('defaults to dark and flips data-theme + localStorage on toggle', () => {
    render(
      <ThemeProvider>
        <ToggleHarness />
      </ThemeProvider>
    )

    const btn = screen.getByTestId('theme-toggle')
    expect(btn.textContent).toBe('dark')
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')

    fireEvent.click(btn)
    expect(btn.textContent).toBe('light')
    expect(document.documentElement.getAttribute('data-theme')).toBe('light')
    expect(window.localStorage.getItem('fxos-theme')).toBe('light')

    fireEvent.click(btn)
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
  })

  it('restores the persisted theme on mount', () => {
    window.localStorage.setItem('fxos-theme', 'light')
    render(
      <ThemeProvider>
        <ToggleHarness />
      </ThemeProvider>
    )
    expect(screen.getByTestId('theme-toggle').textContent).toBe('light')
    expect(document.documentElement.getAttribute('data-theme')).toBe('light')
  })
})

describe('HeaderBar chrome', () => {
  it('renders the theme toggle and no Leaderboard tab (desktop + mobile arrays)', () => {
    render(
      <MemoryRouter>
        <LanguageProvider>
          <AuthProvider>
            <ThemeProvider>
              <HeaderBar isLoggedIn={false} language="en" />
            </ThemeProvider>
          </AuthProvider>
        </LanguageProvider>
      </MemoryRouter>
    )

    // Theme toggle present
    const toggle = screen.getByRole('button', {
      name: /switch to light theme/i,
    })
    expect(toggle).toBeTruthy()

    // Leaderboard must be gone from the nav
    expect(screen.queryByText('Leaderboard')).toBeNull()
    expect(screen.queryByText('排行榜')).toBeNull()

    // Data page must be gone too (removed)
    expect(screen.queryByText('Data')).toBeNull()

    // Core nav still intact
    expect(screen.getByText('FAQ')).toBeTruthy()
  })
})
