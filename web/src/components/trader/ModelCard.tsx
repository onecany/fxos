import { Check } from 'lucide-react'
import type { AIModel } from '../../types'
import { getModelIcon } from '../common/ModelIcons'
import { getShortName } from './model-constants'

interface ModelCardProps {
  model: AIModel
  selected: boolean
  onClick: () => void
  configured?: boolean
}

export function ModelCard({ model, selected, onClick, configured }: ModelCardProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="flex flex-col items-center gap-2 p-4 rounded-xl transition-all hover:scale-105"
      style={{
        background: selected ? 'rgba(45, 212, 191, 0.12)' : 'var(--panel-bg)',
        border: selected ? '2px solid var(--fxos-gold)' : '2px solid rgba(26,24,19,0.14)',
      }}
    >
      <div className="relative">
        <div className="w-12 h-12 rounded-xl flex items-center justify-center bg-fxos-bg-deeper border border-[var(--panel-border)]">
          {getModelIcon(model.provider || model.id, { width: 32, height: 32 }) || (
            <span className="text-lg font-bold" style={{ color: 'var(--fxos-gold)' }}>{model.name[0]}</span>
          )}
        </div>
        {selected && (
          <div
            className="absolute -top-1 -right-1 w-5 h-5 rounded-full flex items-center justify-center"
            style={{ background: 'var(--fxos-success)' }}
          >
            <Check className="w-3 h-3 text-white" />
          </div>
        )}
        {configured && !selected && (
          <div
            className="absolute -top-1 -right-1 w-4 h-4 rounded-full flex items-center justify-center"
            style={{ background: 'var(--fxos-gold)' }}
          >
            <Check className="w-2.5 h-2.5 text-white" />
          </div>
        )}
      </div>
      <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>
        {getShortName(model.name)}
      </span>
      <span
        className="text-[10px] px-2 py-0.5 rounded-full uppercase tracking-wide"
        style={{ background: 'rgba(45, 212, 191, 0.16)', color: 'var(--fxos-gold)' }}
      >
        {model.provider}
      </span>
    </button>
  )
}
