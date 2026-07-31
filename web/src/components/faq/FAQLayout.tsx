import { useMemo, useState } from 'react'
import { HelpCircle } from 'lucide-react'
import { DeepVoidBackground } from '../common/DeepVoidBackground'
import { FAQSearchBar } from './FAQSearchBar'
import { FAQSidebar } from './FAQSidebar'
import { FAQContent } from './FAQContent'
import { getFaqCategories, faqItemSearchText } from './faqData'
import type { FAQCategory } from './faqData'
import { useLanguage } from '../../contexts/LanguageContext'
import { t } from '../../i18n/translations'

export function FAQLayout() {
  const { language } = useLanguage()
  const [searchTerm, setSearchTerm] = useState('')
  const [activeItemId, setActiveItemId] = useState<string | null>(null)

  const categories = useMemo(() => getFaqCategories(language), [language])

  const filteredCategories = useMemo(() => {
    if (!searchTerm.trim()) return categories

    const term = searchTerm.toLowerCase()
    const filtered: FAQCategory[] = []
    categories.forEach((category) => {
      const matchingItems = category.items.filter((item) =>
        faqItemSearchText(item).includes(term)
      )
      if (matchingItems.length > 0) {
        filtered.push({ ...category, items: matchingItems })
      }
    })
    return filtered
  }, [searchTerm, categories])

  const totalItems = useMemo(
    () => categories.reduce((sum, category) => sum + category.items.length, 0),
    [categories]
  )

  const handleItemClick = (_categoryId: string, itemId: string) => {
    const element = document.getElementById(itemId)
    if (!element) return
    const offset = 100
    const top =
      element.getBoundingClientRect().top + window.pageYOffset - offset
    window.scrollTo({ top, behavior: 'smooth' })
  }

  return (
    <DeepVoidBackground className="py-8 pt-24" disableAnimation>
      <div className="mx-auto w-full max-w-6xl px-4 md:px-8">
        {/* page header — same strip language as the other terminal pages */}
        <div className="mb-8 border-b border-fxos-gold/20 pb-6">
          <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-xl border border-fxos-gold/30 bg-fxos-bg-lighter text-fxos-gold md:h-14 md:w-14">
                <HelpCircle className="h-6 w-6 md:h-7 md:w-7" />
              </div>
              <div>
                <h1 className="font-mono text-2xl font-bold tracking-tight text-fxos-text md:text-3xl">
                  {t('faqPage.title', language)}
                </h1>
                <p className="mt-1 font-mono text-xs uppercase tracking-[0.14em] text-fxos-text-muted">
                  {t('faqPage.subtitle', language, { count: totalItems })}
                </p>
              </div>
            </div>
            <div className="w-full md:w-80">
              <FAQSearchBar
                searchTerm={searchTerm}
                onSearchChange={setSearchTerm}
                placeholder={t('faqPage.searchPlaceholder', language)}
              />
            </div>
          </div>
        </div>

        {/* content */}
        <div className="flex gap-8">
          <aside className="hidden w-64 flex-shrink-0 lg:block">
            <FAQSidebar
              categories={filteredCategories}
              activeItemId={activeItemId}
              onItemClick={handleItemClick}
            />
          </aside>

          <main className="min-w-0 flex-1">
            {filteredCategories.length > 0 ? (
              <FAQContent
                categories={filteredCategories}
                onActiveItemChange={setActiveItemId}
              />
            ) : (
              <div className="rounded-xl border border-fxos-gold/20 bg-fxos-bg-lighter py-16 text-center">
                <p className="font-mono text-sm text-fxos-text-muted">
                  {t('faqPage.noResults', language, { term: searchTerm })}
                </p>
                <button
                  onClick={() => setSearchTerm('')}
                  className="mt-4 rounded-lg border border-fxos-gold/30 bg-fxos-gold/10 px-5 py-2 font-mono text-xs font-bold uppercase tracking-[0.12em] text-fxos-gold hover:bg-fxos-gold/20"
                >
                  {t('faqPage.clearSearch', language)}
                </button>
              </div>
            )}
          </main>
        </div>

        {/* still stuck */}
        <div className="mt-12 rounded-xl border border-fxos-gold/20 bg-fxos-bg-lighter p-6 text-center md:p-8">
          <h3 className="font-mono text-sm font-bold uppercase tracking-[0.16em] text-fxos-text">
            {t('faqPage.stillQuestions', language)}
          </h3>
          <p className="mt-2 text-sm text-fxos-text-muted">
            {t('faqPage.askCommunity', language)}
          </p>
          <div className="mt-5 flex items-center justify-center gap-3">
            <a
              href="https://github.com/onecany/fxos"
              target="_blank"
              rel="noopener noreferrer"
              className="rounded-lg border border-[var(--panel-border)] bg-fxos-bg-deeper px-5 py-2.5 font-mono text-xs font-bold uppercase tracking-[0.12em] text-fxos-text hover:border-fxos-gold/40"
            >
              GitHub
            </a>
            <a
              href="https://t.me/fxos_dev_community"
              target="_blank"
              rel="noopener noreferrer"
              className="rounded-lg bg-fxos-gold px-5 py-2.5 font-mono text-xs font-bold uppercase tracking-[0.12em] text-white hover:bg-fxos-accent"
            >
              {t('faqPage.telegramCommunity', language)}
            </a>
          </div>
        </div>
      </div>
    </DeepVoidBackground>
  )
}
