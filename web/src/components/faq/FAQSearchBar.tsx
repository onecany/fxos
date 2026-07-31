import { Search, X } from 'lucide-react'

interface FAQSearchBarProps {
  searchTerm: string
  onSearchChange: (value: string) => void
  placeholder?: string
}

export function FAQSearchBar({
  searchTerm,
  onSearchChange,
  placeholder = 'Search FAQ...',
}: FAQSearchBarProps) {
  return (
    <div className="relative group">
      <Search
        className="absolute left-4 top-1/2 transform -translate-y-1/2 w-5 h-5 text-fxos-text-muted group-focus-within:text-fxos-gold transition-colors"
      />
      <input
        type="text"
        value={searchTerm}
        onChange={(e) => onSearchChange(e.target.value)}
        placeholder={placeholder}
        className="w-full pl-12 pr-12 py-3 rounded-lg text-base transition-all focus:outline-none bg-fxos-bg-lighter border border-[var(--panel-border)] text-fxos-text placeholder-fxos-text-muted/50 focus:border-fxos-gold/50 focus:ring-1 focus:ring-fxos-gold/20 hover:border-fxos-gold/30 font-mono"
      />
      {searchTerm && (
        <button
          onClick={() => onSearchChange('')}
          className="absolute right-4 top-1/2 transform -translate-y-1/2 text-fxos-text-muted hover:text-fxos-text transition-colors"
        >
          <X className="w-5 h-5" />
        </button>
      )}
    </div>
  )
}
