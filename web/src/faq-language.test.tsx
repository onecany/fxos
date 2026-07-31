import { describe, it, expect } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { LanguageProvider, useLanguage } from './contexts/LanguageContext'
import { FAQLayout } from './components/faq/FAQLayout'
import { getFaqCategories } from './components/faq/faqData'

// Language switcher harness — mirrors what HeaderBar does
function SwitchToZh() {
  const { setLanguage } = useLanguage()
  return (
    <button onClick={() => setLanguage('zh')} data-testid="switch-zh">
      switch
    </button>
  )
}

describe('FAQ page language switching', () => {
  it('renders content in the selected language and switches on demand', () => {
    render(
      <LanguageProvider>
        <SwitchToZh />
        <FAQLayout />
      </LanguageProvider>
    )

    // Default: English content
    expect(screen.getAllByText('Getting Started').length).toBeGreaterThan(0)
    expect(screen.getAllByText('What is FXOS?').length).toBeGreaterThan(0)
    // Header + search placeholder are localized too
    expect(screen.getByPlaceholderText('Search FAQ...')).toBeTruthy()

    // Switch to Chinese
    fireEvent.click(screen.getByTestId('switch-zh'))
    expect(screen.getAllByText('快速上手').length).toBeGreaterThan(0)
    expect(screen.getAllByText('FXOS 是什么?').length).toBeGreaterThan(0)
    expect(screen.getByText('常见问题')).toBeTruthy()
    expect(screen.getByPlaceholderText('搜索常见问题...')).toBeTruthy()
    // English no longer present
    expect(screen.queryByText('What is FXOS?')).toBeNull()
  })

  it('data trees are complete and share category ids across languages', () => {
    const en = getFaqCategories('en')
    const zh = getFaqCategories('zh')
    const id = getFaqCategories('id')

    expect(en.length).toBe(7)
    expect(zh.length).toBe(7)
    expect(id.length).toBe(7)

    // Same category ids, same item ids, icons attached
    expect(en.map((c) => c.id)).toEqual(zh.map((c) => c.id))
    expect(en.map((c) => c.id)).toEqual(id.map((c) => c.id))
    en.forEach((cat, i) => {
      expect(cat.icon).toBeTruthy()
      expect(cat.items.map((it) => it.id)).toEqual(
        zh[i].items.map((it) => it.id)
      )
      expect(cat.items.map((it) => it.id)).toEqual(
        id[i].items.map((it) => it.id)
      )
    })
  })
})
