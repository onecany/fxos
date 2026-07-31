import { useState, useEffect } from 'react'
import { X } from 'lucide-react'
import type { DecisionRecord, DecisionAction } from '../../types'
import { t, type Language } from '../../i18n/translations'

interface DecisionDetailModalProps {
  decision: DecisionRecord
  language: Language
  onClose: () => void
  onSymbolClick?: (symbol: string) => void
}

type Tab = 'summary' | 'cot' | 'system' | 'user' | 'raw'

const ACTION_CONFIG: Record<string, { color: string; bg: string; icon: string; label: string }> = {
  open_long: { color: 'var(--fxos-success)', bg: 'rgba(46, 139, 87, 0.15)', icon: '📈', label: 'LONG' },
  open_short: { color: 'var(--fxos-danger)', bg: 'rgba(214, 67, 58, 0.15)', icon: '📉', label: 'SHORT' },
  close_long: { color: 'var(--fxos-gold)', bg: 'rgba(45, 212, 191, 0.14)', icon: '💰', label: 'CLOSE' },
  close_short: { color: 'var(--fxos-gold)', bg: 'rgba(45, 212, 191, 0.14)', icon: '💰', label: 'CLOSE' },
  hold: { color: 'var(--text-secondary)', bg: 'rgba(138, 132, 120, 0.15)', icon: '⏸️', label: 'HOLD' },
  wait: { color: 'var(--text-secondary)', bg: 'rgba(138, 132, 120, 0.15)', icon: '⏳', label: 'WAIT' },
  modify: { color: 'var(--fxos-gold)', bg: 'rgba(212, 160, 23, 0.15)', icon: '🔧', label: 'MODIFY' },
}

// Parse CoT trace into structured steps
interface CoTStep {
  label: string
  icon: string
  content: string
}

function parseCoTTrace(cotTrace: string): CoTStep[] {
  if (!cotTrace) return []
  const stepPattern = /\*\*([^*]+)\*\*[:\s]*(.*?)(?=\n\*\*[^*]+\*\*|\n*$)/gs
  const steps: CoTStep[] = []
  let match
  const iconMap: Record<string, string> = {
    'REGIME': '🎯', 'RISK': '⚠️', 'POSITION': '📊',
    'CANDIDATE': '🔍', 'CONFLUENCE': '🤝', 'R/R': '💰', 'DECISION': '✅',
  }
  while ((match = stepPattern.exec(cotTrace)) !== null) {
    const label = match[1].trim()
    const content = match[2].trim()
    let icon = '📌'
    for (const [key, val] of Object.entries(iconMap)) {
      if (label.toUpperCase().includes(key)) { icon = val; break }
    }
    steps.push({ label, icon, content })
  }
  if (steps.length === 0) {
    cotTrace.split(/\n\n+/).filter(p => p.trim()).forEach(para => {
      steps.push({ label: '', icon: '📌', content: para.trim() })
    })
  }
  return steps
}

function formatTime(timestamp: string): string {
  try { return new Date(timestamp).toLocaleString() } catch { return timestamp }
}

function getConfidenceColor(confidence: number | undefined): string {
  if (!confidence) return 'var(--text-secondary)'
  if (confidence >= 80) return 'var(--fxos-success)'
  if (confidence >= 60) return 'var(--fxos-gold)'
  return 'var(--fxos-danger)'
}

function formatPrice(price: number | undefined): string {
  if (!price || price === 0) return '-'
  if (price >= 1000) return price.toFixed(2)
  if (price >= 1) return price.toFixed(4)
  return price.toFixed(6)
}

export default function DecisionDetailModal({ decision, language, onClose, onSymbolClick }: DecisionDetailModalProps) {
  const [activeTab, setActiveTab] = useState<Tab>('summary')
  const cotSteps = parseCoTTrace(decision.cot_trace)

  useEffect(() => {
    const handleEsc = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    window.addEventListener('keydown', handleEsc)
    return () => window.removeEventListener('keydown', handleEsc)
  }, [onClose])

  const tabs: { key: Tab; labelKey: string; icon: string }[] = [
    { key: 'summary', labelKey: 'summary', icon: '📋' },
    { key: 'cot', labelKey: 'cotAnalysis', icon: '🧠' },
    { key: 'system', labelKey: 'systemPrompt', icon: '⚙️' },
    { key: 'user', labelKey: 'userPrompt', icon: '📨' },
  ]
  if (decision.raw_response) {
    tabs.push({ key: 'raw', labelKey: 'rawResponse', icon: '📄' })
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4"
      onClick={(e) => { if (e.target === e.currentTarget) onClose() }}
    >
      <div
        className="w-full max-w-4xl max-h-[90vh] rounded-2xl overflow-hidden flex flex-col"
        style={{ background: 'var(--panel-bg)', border: '1px solid var(--panel-border)' }}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b" style={{ borderColor: 'var(--panel-border)' }}>
          <div className="flex items-center gap-3">
            <span className="text-lg font-bold" style={{ color: 'var(--text-primary)' }}>
              {t('cycle', language)} #{decision.cycle_number}
            </span>
            <span className="text-sm" style={{ color: 'var(--text-secondary)' }}>{formatTime(decision.timestamp)}</span>
            <span
              className="text-xs px-2 py-0.5 rounded-full font-semibold"
              style={{
                background: decision.success ? 'rgba(46,139,87,0.15)' : 'rgba(214,67,58,0.15)',
                color: decision.success ? 'var(--fxos-success)' : 'var(--fxos-danger)',
              }}
            >
              {decision.success ? `✅ ${t('success', language)}` : `❌ ${t('failed', language)}`}
            </span>
          </div>
          <button onClick={onClose} className="p-2 rounded-lg hover:bg-fxos-gold-dim transition-colors">
            <X size={20} style={{ color: 'var(--text-secondary)' }} />
          </button>
        </div>

        {/* Tabs */}
        <div className="flex gap-1 px-6 pt-3 border-b overflow-x-auto flex-nowrap" style={{ borderColor: 'var(--panel-border)' }}>
          {tabs.map((tab) => (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key)}
              className="px-3 py-2 text-sm font-medium rounded-t-lg transition-colors"
              style={{
                color: activeTab === tab.key ? 'var(--fxos-gold)' : 'var(--text-secondary)',
                background: activeTab === tab.key ? 'var(--fxos-bg-lighter)' : 'transparent',
                borderBottom: activeTab === tab.key ? '2px solid var(--fxos-gold)' : '2px solid transparent',
              }}
            >
              {tab.icon} {t(tab.labelKey, language)}
            </button>
          ))}
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto px-6 py-4">
          {activeTab === 'summary' && (
            <div className="space-y-4">
              {decision.account_state && (
                <div className="rounded-lg p-4" style={{ background: 'var(--fxos-bg-lighter)', border: '1px solid var(--panel-border)' }}>
                  <h3 className="text-sm font-semibold mb-2" style={{ color: 'var(--text-primary)' }}>{t('accountState', language)}</h3>
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-3 text-sm">
                    <div><span style={{ color: 'var(--text-secondary)' }}>{t('equity', language)}:</span> <span style={{ color: 'var(--text-primary)' }}>${decision.account_state.total_balance?.toFixed(2)}</span></div>
                    <div><span style={{ color: 'var(--text-secondary)' }}>{t('availableBalance', language)}:</span> <span style={{ color: 'var(--text-primary)' }}>${decision.account_state.available_balance?.toFixed(2)}</span></div>
                    <div><span style={{ color: 'var(--text-secondary)' }}>{t('unrealizedPnL', language)}:</span> <span style={{ color: 'var(--text-primary)' }}>${decision.account_state.total_unrealized_profit?.toFixed(2)}</span></div>
                    <div><span style={{ color: 'var(--text-secondary)' }}>{t('positions', language)}:</span> <span style={{ color: 'var(--text-primary)' }}>{decision.account_state.position_count}</span></div>
                  </div>
                </div>
              )}
              <div>
                <h3 className="text-sm font-semibold mb-3" style={{ color: 'var(--text-primary)' }}>{t('decisions', language)} ({decision.decisions?.length || 0})</h3>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                  {decision.decisions?.map((action, idx) => (
                    <ActionDetailCard key={idx} action={action} language={language} onSymbolClick={onSymbolClick} />
                  ))}
                </div>
              </div>
              {decision.execution_log && decision.execution_log.length > 0 && (
                <div className="rounded-lg p-4" style={{ background: 'var(--fxos-bg-lighter)', border: '1px solid var(--panel-border)' }}>
                  <h3 className="text-sm font-semibold mb-2" style={{ color: 'var(--text-primary)' }}>{t('executionLog', language)}</h3>
                  <div className="text-xs font-mono space-y-1">
                    {decision.execution_log.map((log, i) => (
                      <div key={i} style={{ color: 'var(--text-primary)' }}>{log}</div>
                    ))}
                  </div>
                </div>
              )}
              {decision.error_message && (
                <div className="rounded-lg p-4" style={{ background: 'rgba(214,67,58,0.1)', border: '1px solid rgba(214,67,58,0.4)' }}>
                  <span style={{ color: 'var(--fxos-danger)' }}>❌ {decision.error_message}</span>
                </div>
              )}
            </div>
          )}

          {activeTab === 'cot' && (
            <div className="space-y-3">
              {cotSteps.length === 0 ? (
                <div className="text-sm" style={{ color: 'var(--text-secondary)' }}>{t('noCotTrace', language)}</div>
              ) : (
                cotSteps.map((step, idx) => (
                  <div key={idx} className="rounded-lg p-4" style={{ background: 'var(--fxos-bg-lighter)', border: '1px solid var(--panel-border)' }}>
                    <div className="flex items-center gap-2 mb-2">
                      <span className="text-base">{step.icon}</span>
                      {step.label && <span className="text-sm font-bold" style={{ color: 'var(--fxos-gold)' }}>{step.label}</span>}
                    </div>
                    <div className="text-sm whitespace-pre-wrap" style={{ color: 'var(--text-primary)' }}>{step.content}</div>
                  </div>
                ))
              )}
            </div>
          )}

          {activeTab === 'system' && (
            <div className="rounded-lg p-4 text-sm font-mono whitespace-pre-wrap max-h-[60vh] overflow-y-auto" style={{ background: 'var(--fxos-bg-lighter)', border: '1px solid var(--panel-border)', color: 'var(--text-primary)' }}>
              {decision.system_prompt || t('noSystemPrompt', language)}
            </div>
          )}

          {activeTab === 'user' && (
            <div className="rounded-lg p-4 text-sm font-mono whitespace-pre-wrap max-h-[60vh] overflow-y-auto" style={{ background: 'var(--fxos-bg-lighter)', border: '1px solid var(--panel-border)', color: 'var(--text-primary)' }}>
              {decision.input_prompt || t('noUserPrompt', language)}
            </div>
          )}

          {activeTab === 'raw' && decision.raw_response && (
            <div className="rounded-lg p-4 text-sm font-mono whitespace-pre-wrap max-h-[60vh] overflow-y-auto" style={{ background: 'var(--fxos-bg-lighter)', border: '1px solid var(--panel-border)', color: 'var(--text-primary)' }}>
              {decision.raw_response}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

// Action detail card
function ActionDetailCard({ action, language, onSymbolClick }: { action: DecisionAction; language: Language; onSymbolClick?: (symbol: string) => void }) {
  const config = ACTION_CONFIG[action.action] || ACTION_CONFIG.wait
  return (
    <div className="rounded-lg p-3 transition-all" style={{ background: 'var(--panel-bg)', border: `1px solid ${config.color}33` }}>
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          <span className="text-xs font-bold px-2 py-0.5 rounded" style={{ background: config.bg, color: config.color }}>
            {config.icon} {config.label}
          </span>
          {onSymbolClick ? (
            <button onClick={() => onSymbolClick(action.symbol)} className="text-sm font-semibold hover:underline" style={{ color: 'var(--text-primary)' }}>
              {action.symbol}
            </button>
          ) : (
            <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>{action.symbol}</span>
          )}
        </div>
        {action.confidence !== undefined && (
          <span className="text-xs font-bold px-2 py-0.5 rounded" style={{ background: `${getConfidenceColor(action.confidence)}20`, color: getConfidenceColor(action.confidence) }}>
            {action.confidence}%
          </span>
        )}
      </div>
      <div className="grid grid-cols-2 gap-2 text-xs">
        {action.leverage > 0 && <div><span style={{ color: 'var(--text-secondary)' }}>{t('leverage', language)}:</span> <span style={{ color: 'var(--text-primary)' }}>{action.leverage}x</span></div>}
        {action.price > 0 && <div><span style={{ color: 'var(--text-secondary)' }}>{t('price', language)}:</span> <span style={{ color: 'var(--text-primary)' }}>${formatPrice(action.price)}</span></div>}
        {action.stop_loss && action.stop_loss > 0 && <div><span style={{ color: 'var(--text-secondary)' }}>{t('stopLoss', language)}:</span> <span style={{ color: 'var(--fxos-danger)' }}>${formatPrice(action.stop_loss)}</span></div>}
        {action.take_profit && action.take_profit > 0 && <div><span style={{ color: 'var(--text-secondary)' }}>{t('takeProfit', language)}:</span> <span style={{ color: 'var(--fxos-success)' }}>${formatPrice(action.take_profit)}</span></div>}
      </div>
      {action.reasoning && <div className="mt-2 text-xs" style={{ color: 'var(--text-secondary)' }}>💡 {action.reasoning}</div>}
      {action.error && <div className="mt-2 text-xs" style={{ color: 'var(--fxos-danger)' }}>❌ {action.error}</div>}
    </div>
  )
}
