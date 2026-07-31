import type { UserMode } from '../../lib/onboarding'

interface OnboardingModeSelectorProps {
  language: string
  mode: UserMode
  onChange: (mode: UserMode) => void
}

export function OnboardingModeSelector({
  language,
  mode,
  onChange,
}: OnboardingModeSelectorProps) {
  const isZh = language === 'zh'

  const options: Array<{
    id: UserMode
    title: string
    badge?: string
    description: string
  }> = [
    {
      id: 'beginner',
      title: isZh ? 'Beginner Mode' : 'Beginner Mode',
      badge: isZh ? 'Recommended' : 'Recommended',
      description: isZh
        ? 'Generate a Base wallet automatically and start with Claw402 + GLM by default.'
        : 'Generate a Base wallet automatically and start with Claw402 + GLM by default.',
    },
    {
      id: 'advanced',
      title: isZh ? 'Advanced Mode' : 'Advanced Mode',
      description: isZh
        ? 'Keep the full manual flow and configure models, wallets, and exchanges yourself.'
        : 'Keep the full manual flow and configure models, wallets, and exchanges yourself.',
    },
  ]

  return (
    <div className="space-y-2">
      <div className="text-xs font-medium text-fxos-text-muted">
        {isZh ? 'Experience' : 'Experience'}
      </div>
      <div className="grid grid-cols-1 gap-2">
        {options.map((option) => {
          const selected = option.id === mode
          return (
            <button
              key={option.id}
              type="button"
              onClick={() => onChange(option.id)}
              className={`w-full rounded-xl border px-4 py-3 text-left transition-all ${
                selected
                  ? 'border-fxos-gold/60 bg-fxos-gold/10'
                  : 'border-[var(--panel-border)] bg-fxos-bg-lighter hover:border-fxos-gold/40'
              }`}
            >
              <div className="flex items-center gap-2 text-sm font-semibold text-fxos-text">
                <span>{option.title}</span>
                {option.badge ? (
                  <span className="rounded-full bg-fxos-gold px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide text-fxos-bg">
                    {option.badge}
                  </span>
                ) : null}
              </div>
              <p className="mt-1 text-xs leading-5 text-fxos-text-muted">
                {option.description}
              </p>
            </button>
          )
        })}
      </div>
    </div>
  )
}
