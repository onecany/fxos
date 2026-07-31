import { Globe } from 'lucide-react'
import { useLanguage } from '../../contexts/LanguageContext'
import type { Language } from '../../i18n/translations'

const languages: { code: Language; label: string }[] = [
  { code: 'zh', label: 'Chinese' },
  { code: 'en', label: 'EN' },
  { code: 'id', label: 'ID' },
]

export function LanguageSwitcher() {
  const { language, setLanguage } = useLanguage()

  return (
    <div className="absolute top-4 right-4 z-50 flex items-center gap-1 rounded-lg p-1 border border-[var(--panel-border)] bg-fxos-bg-lighter backdrop-blur-sm">
      <Globe size={14} className="text-fxos-text-muted ml-1.5 mr-0.5" />
      {languages.map(({ code, label }) => (
        <button
          key={code}
          type="button"
          onClick={() => setLanguage(code)}
          className={`px-2.5 py-1 rounded text-xs font-semibold transition-all ${
            language === code
              ? 'bg-fxos-gold/15 text-fxos-gold'
              : 'text-fxos-text-muted hover:text-fxos-text bg-transparent'
          }`}
        >
          {label}
        </button>
      ))}
    </div>
  )
}
