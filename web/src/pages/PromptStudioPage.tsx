import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Check, ClipboardCopy, Loader2, RefreshCw, Save } from 'lucide-react'
import { useAuth } from '../contexts/AuthContext'
import { useLanguage } from '../contexts/LanguageContext'
import { DeepVoidBackground } from '../components/common/DeepVoidBackground'
import { t } from '../i18n/translations'
import { API_BASE, httpClient } from '../lib/api/helpers'
import { strategyApi } from '../lib/api/strategies'
import type { Strategy, StrategyConfig } from '../types'
import { ROUTES } from '../router/paths'

interface PromptSections {
  role_definition: string
  trading_frequency: string
  entry_standards: string
  decision_process: string
}

interface TokenEstimate {
  total: number
  breakdown?: {
    system_prompt?: number
    market_data?: number
    ranking_data?: number
    quant_data?: number
    fixed_overhead?: number
  }
  model_limits?: Array<{
    name: string
    context_limit: number
    usage_pct: number
    level: string
  }>
  suggestions?: string[]
}

interface PreviewResponse {
  system_prompt: string
  prompt_variant: string
  lint_warnings?: string[]
}

interface TestRunResponse {
  system_prompt?: string
  user_prompt?: string
  ai_response?: string
  candidate_count?: number
  prompt_variant?: string
  note?: string
  ai_error?: string
}

interface SafeModel {
  id: string
  name: string
  provider: string
  enabled: boolean
}

const EMPTY_SECTIONS: PromptSections = {
  role_definition: '',
  trading_frequency: '',
  entry_standards: '',
  decision_process: '',
}

const VARIANT_OPTIONS = ['careful', 'balanced', 'active'] as const
type Variant = (typeof VARIANT_OPTIONS)[number]

function readSections(
  config: StrategyConfig | null | undefined
): PromptSections {
  const ai = config?.ai_config
  const sections = ai?.prompt_sections
  return {
    role_definition: sections?.role_definition || '',
    trading_frequency: sections?.trading_frequency || '',
    entry_standards: sections?.entry_standards || '',
    decision_process: sections?.decision_process || '',
  }
}

function readCustomPrompt(config: StrategyConfig | null | undefined): string {
  return config?.ai_config?.custom_prompt || ''
}

export function PromptStudioPage() {
  const { token } = useAuth()
  const { language } = useLanguage()
  const navigate = useNavigate()

  const [strategies, setStrategies] = useState<Strategy[]>([])
  const [selectedId, setSelectedId] = useState('')
  const [sections, setSections] = useState<PromptSections>(EMPTY_SECTIONS)
  const [customPrompt, setCustomPrompt] = useState('')
  const [loadingStrategies, setLoadingStrategies] = useState(true)
  const [loadError, setLoadError] = useState('')

  const [equity, setEquity] = useState(1000)
  const [variant, setVariant] = useState<Variant>('balanced')
  const [preview, setPreview] = useState('')
  const [lintWarnings, setLintWarnings] = useState<string[]>([])
  const [previewLoading, setPreviewLoading] = useState(false)
  const [previewError, setPreviewError] = useState('')
  const [estimate, setEstimate] = useState<TokenEstimate | null>(null)
  const [estimateError, setEstimateError] = useState('')

  // AI real-run test
  const [models, setModels] = useState<SafeModel[]>([])
  const [selectedModelId, setSelectedModelId] = useState('')
  const [testResponse, setTestResponse] = useState<TestRunResponse | null>(null)
  const [testLoading, setTestLoading] = useState(false)
  const [testError, setTestError] = useState('')

  const [saving, setSaving] = useState(false)
  const [savedFlag, setSavedFlag] = useState(false)
  const [copiedFlag, setCopiedFlag] = useState(false)
  const [resetting, setResetting] = useState(false)

  const previewTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const estimateTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const savedTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const copiedTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  const loadStrategies = useCallback(async () => {
    if (!token) return
    setLoadingStrategies(true)
    setLoadError('')
    try {
      const list = await strategyApi.getStrategies()
      setStrategies(list)
      if (list.length > 0 && !selectedId) {
        // Prefer a strategy with a pinned (static) coin universe — the
        // user most likely wants to preview the coins they selected
        // manually. Fall back to the first strategy otherwise.
        const staticStrategy =
          list.find((s) => {
            const cs = (s.config as StrategyConfig)?.ai_config?.coin_source
            return (
              cs?.source_type === 'static' && (cs.static_coins?.length || 0) > 0
            )
          }) || list[0]
        setSelectedId(staticStrategy.id)
      }
    } catch {
      setLoadError(t('promptStudio.loadFailed', language))
    } finally {
      setLoadingStrategies(false)
    }
  }, [token, language, selectedId])

  useEffect(() => {
    loadStrategies()
  }, [loadStrategies])

  const selectedStrategy = useMemo(
    () => strategies.find((s) => s.id === selectedId) || null,
    [strategies, selectedId]
  )

  // Load sections when strategy changes
  useEffect(() => {
    if (!selectedStrategy) {
      setSections(EMPTY_SECTIONS)
      setCustomPrompt('')
      return
    }
    setSections(readSections(selectedStrategy.config))
    setCustomPrompt(readCustomPrompt(selectedStrategy.config))
  }, [selectedStrategy])

  // Build the full config that the backend preview expects: merge current
  // section edits on top of the strategy's saved config.
  const previewConfig = useMemo((): StrategyConfig | null => {
    if (!selectedStrategy) return null
    const base = selectedStrategy.config
    return {
      ...base,
      ai_config: {
        ...(base.ai_config as StrategyConfig['ai_config']),
        prompt_sections: {
          ...(base.ai_config?.prompt_sections || {}),
          ...sections,
        },
        custom_prompt: customPrompt,
      } as StrategyConfig['ai_config'],
    }
  }, [selectedStrategy, sections, customPrompt])

  const runPreview = useCallback(async () => {
    if (!previewConfig || !token) return
    setPreviewLoading(true)
    setPreviewError('')
    try {
      const result = await httpClient.post<PreviewResponse>(
        `${API_BASE}/strategies/preview-prompt`,
        {
          config: previewConfig,
          account_equity: equity,
          prompt_variant: variant,
        }
      )
      if (result.success && result.data) {
        setPreview(result.data.system_prompt || '')
        setLintWarnings(result.data.lint_warnings || [])
      } else {
        setPreviewError(
          result.message || t('promptStudio.previewFailed', language)
        )
      }
    } catch {
      setPreviewError(t('promptStudio.previewFailed', language))
    } finally {
      setPreviewLoading(false)
    }
  }, [previewConfig, equity, variant, token, language])

  const runEstimate = useCallback(async () => {
    if (!previewConfig || !token) return
    setEstimateError('')
    try {
      const result = await httpClient.post<TokenEstimate>(
        `${API_BASE}/strategies/estimate-tokens`,
        { config: previewConfig }
      )
      if (result.success && result.data) {
        setEstimate(result.data)
      } else {
        setEstimate(null)
        setEstimateError(
          result.message || t('promptStudio.previewFailed', language)
        )
      }
    } catch {
      setEstimate(null)
      setEstimateError(t('promptStudio.previewFailed', language))
    }
  }, [previewConfig, token, language])

  // Load enabled AI models for the real-run test
  const loadModels = useCallback(async () => {
    if (!token) return
    try {
      const result = await httpClient.get<SafeModel[]>(`${API_BASE}/models`)
      if (result.success && Array.isArray(result.data)) {
        const enabled = result.data.filter((m) => m.enabled)
        const pool = enabled.length > 0 ? enabled : result.data
        setModels(pool)
        setSelectedModelId((prev) => {
          if (prev && pool.some((m) => m.id === prev)) return prev
          return pool[0]?.id || ''
        })
      }
    } catch {
      // model list failure is non-fatal — user can still use preview
    }
  }, [token])

  useEffect(() => {
    void loadModels()
  }, [loadModels])

  // Run the real AI test: send the current prompt config to the backend,
  // which fetches live market data and calls the selected model.
  const runTest = useCallback(async () => {
    if (!previewConfig || !token || !selectedModelId) {
      setTestError(t('promptStudio.selectModelFirst', language))
      return
    }
    setTestLoading(true)
    setTestError('')
    setTestResponse(null)
    try {
      const result = await httpClient.request<TestRunResponse>(
        `${API_BASE}/strategies/test-run`,
        {
          method: 'POST',
          data: {
            config: previewConfig,
            prompt_variant: variant,
            ai_model_id: selectedModelId,
            run_real_ai: true,
          },
          timeout: 120000,
        }
      )
      if (result.success && result.data) {
        setTestResponse(result.data)
        if (result.data.ai_error) {
          setTestError(result.data.ai_error)
        }
      } else {
        setTestError(result.message || t('promptStudio.testFailed', language))
      }
    } catch {
      setTestError(t('promptStudio.testFailed', language))
    } finally {
      setTestLoading(false)
    }
  }, [previewConfig, token, selectedModelId, variant, language])

  // Debounced preview + estimate
  useEffect(() => {
    if (!previewConfig) return
    if (previewTimer.current) clearTimeout(previewTimer.current)
    if (estimateTimer.current) clearTimeout(estimateTimer.current)
    previewTimer.current = setTimeout(() => {
      runPreview()
    }, 800)
    estimateTimer.current = setTimeout(() => {
      runEstimate()
    }, 800)
    return () => {
      if (previewTimer.current) clearTimeout(previewTimer.current)
      if (estimateTimer.current) clearTimeout(estimateTimer.current)
    }
  }, [previewConfig, equity, variant, runPreview, runEstimate])

  const patchSection = (key: keyof PromptSections, value: string) => {
    setSections((prev) => ({ ...prev, [key]: value }))
    setSavedFlag(false)
  }

  const handleSave = async () => {
    if (!selectedStrategy) return
    setSaving(true)
    setSavedFlag(false)
    try {
      await strategyApi.updateStrategy(selectedStrategy.id, {
        config: {
          prompt_sections: sections,
          custom_prompt: customPrompt,
        },
      })
      setSavedFlag(true)
      if (savedTimer.current) clearTimeout(savedTimer.current)
      savedTimer.current = setTimeout(() => setSavedFlag(false), 2500)
      // Refresh so the saved config is the new base
      await loadStrategies()
    } catch {
      // strategyApi throws; surface generic failure
    } finally {
      setSaving(false)
    }
  }

  const handleReset = async () => {
    if (!selectedStrategy) return
    if (!window.confirm(t('promptStudio.resetConfirm', language))) return
    setResetting(true)
    try {
      const defaultConfig = await strategyApi.getDefaultStrategyConfig()
      setSections(readSections(defaultConfig))
      setCustomPrompt(readCustomPrompt(defaultConfig))
      setSavedFlag(false)
    } finally {
      setResetting(false)
    }
  }

  const handleCopy = async () => {
    if (!preview) return
    try {
      await navigator.clipboard.writeText(preview)
      setCopiedFlag(true)
      if (copiedTimer.current) clearTimeout(copiedTimer.current)
      copiedTimer.current = setTimeout(() => setCopiedFlag(false), 2000)
    } catch {
      setCopiedFlag(false)
    }
  }

  const sectionFields: Array<{
    key: keyof PromptSections
    labelKey: string
    hintKey: string
    placeholderKey: string
    rows: number
  }> = [
    {
      key: 'role_definition',
      labelKey: 'roleDefinition',
      hintKey: 'roleDefinitionHint',
      placeholderKey: 'roleDefinition',
      rows: 4,
    },
    {
      key: 'trading_frequency',
      labelKey: 'tradingFrequency',
      hintKey: 'tradingFrequencyHint',
      placeholderKey: 'tradingFrequency',
      rows: 4,
    },
    {
      key: 'entry_standards',
      labelKey: 'entryStandards',
      hintKey: 'entryStandardsHint',
      placeholderKey: 'entryStandards',
      rows: 5,
    },
    {
      key: 'decision_process',
      labelKey: 'decisionProcess',
      hintKey: 'decisionProcessHint',
      placeholderKey: 'decisionProcess',
      rows: 6,
    },
  ]

  const estimateLevelClass = (level: string | undefined) => {
    if (level === 'danger') return 'text-fxos-danger'
    if (level === 'warning') return 'text-fxos-gold'
    return 'text-fxos-success'
  }

  return (
    <DeepVoidBackground className="min-h-screen">
      <div className="mx-auto max-w-7xl px-4 py-6 md:px-6">
        {/* Header */}
        <div className="mb-6 flex flex-wrap items-center justify-between gap-4">
          <div>
            <button
              onClick={() => navigate(ROUTES.strategy)}
              className="mb-2 text-xs font-mono uppercase tracking-widest text-fxos-text-muted hover:text-fxos-gold transition-colors"
            >
              {t('promptStudio.backToStrategy', language)}
            </button>
            <h1 className="text-xl font-semibold text-fxos-text">
              {t('promptStudio.title', language)}
            </h1>
            <p className="mt-1 text-sm text-fxos-text-muted">
              {t('promptStudio.subtitle', language)}
            </p>
          </div>

          <div className="flex items-center gap-3">
            <select
              value={selectedId}
              onChange={(e) => {
                setSelectedId(e.target.value)
                setSavedFlag(false)
              }}
              disabled={loadingStrategies || strategies.length === 0}
              className="rounded-lg border border-[var(--panel-border)] bg-fxos-bg px-3 py-2 text-sm text-fxos-text disabled:opacity-50"
            >
              {loadingStrategies ? (
                <option>{t('promptStudio.selectStrategy', language)}...</option>
              ) : strategies.length === 0 ? (
                <option>{t('promptStudio.noStrategy', language)}</option>
              ) : (
                strategies.map((s) => {
                  const cs = (s.config as StrategyConfig)?.ai_config
                    ?.coin_source
                  const tag =
                    cs?.source_type === 'static'
                      ? ` · ${t('promptStudio.sourceStatic', language)} (${
                          cs.static_coins?.length || 0
                        })`
                      : cs?.source_type === 'vergex_signal'
                        ? ` · ${t('promptStudio.sourceVergex', language)}`
                        : ` · ${cs?.source_type || ''}`
                  return (
                    <option key={s.id} value={s.id}>
                      {s.name}
                      {tag}
                    </option>
                  )
                })
              )}
            </select>
            <button
              onClick={handleReset}
              disabled={!selectedStrategy || resetting}
              className="inline-flex items-center gap-2 rounded-lg border border-[var(--panel-border)] px-3 py-2 text-sm text-fxos-text-muted hover:text-fxos-text disabled:opacity-50"
            >
              <RefreshCw
                className={`h-4 w-4 ${resetting ? 'animate-spin' : ''}`}
              />
              {t('promptStudio.reset', language)}
            </button>
            <button
              onClick={handleSave}
              disabled={!selectedStrategy || saving}
              className="inline-flex items-center gap-2 rounded-lg bg-fxos-gold px-4 py-2 text-sm font-semibold text-fxos-bg hover:bg-fxos-gold-highlight disabled:opacity-50"
            >
              {saving ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : savedFlag ? (
                <Check className="h-4 w-4" />
              ) : (
                <Save className="h-4 w-4" />
              )}
              {saving
                ? t('promptStudio.saving', language)
                : savedFlag
                  ? t('promptStudio.saved', language)
                  : t('promptStudio.save', language)}
            </button>
          </div>
        </div>

        {loadError && (
          <div className="mb-4 rounded-lg border border-fxos-danger/30 bg-fxos-danger/10 px-4 py-3 text-sm text-fxos-danger">
            {loadError}
          </div>
        )}

        {!selectedStrategy && !loadingStrategies && (
          <div className="rounded-lg border border-[var(--panel-border)] bg-fxos-bg-deeper p-8 text-center text-sm text-fxos-text-muted">
            {t('promptStudio.noStrategy', language)}
          </div>
        )}

        {selectedStrategy && (
          <div className="grid gap-6 lg:grid-cols-[1fr_420px]">
            {/* Left: editor */}
            <div className="space-y-4">
              {sectionFields.map((field) => (
                <section
                  key={field.key}
                  className="rounded-lg border border-[var(--panel-border)] bg-fxos-bg-lighter p-4"
                >
                  <div className="mb-1 flex items-baseline justify-between gap-2">
                    <div>
                      <div className="text-sm font-semibold text-fxos-text">
                        {t(`promptStudio.${field.labelKey}`, language)}
                      </div>
                      <div className="text-xs text-fxos-text-muted">
                        {t(`promptStudio.${field.hintKey}`, language)}
                      </div>
                    </div>
                    <span className="shrink-0 text-[10px] font-mono text-fxos-text-muted">
                      {t('promptStudio.charCount', language, {
                        count: sections[field.key].length,
                      })}
                    </span>
                  </div>
                  <textarea
                    value={sections[field.key]}
                    onChange={(e) => patchSection(field.key, e.target.value)}
                    rows={field.rows}
                    placeholder={t(
                      `promptStudio.${field.placeholderKey}`,
                      language
                    )}
                    className="mt-2 w-full resize-none rounded-lg border border-[var(--panel-border)] bg-fxos-bg px-3 py-2 font-mono text-sm text-fxos-text outline-none placeholder:text-fxos-text-muted/50 focus:border-fxos-gold/60"
                  />
                  <div className="mt-1 text-[10px] text-fxos-text-muted/70">
                    {t('promptStudio.emptyFallsBack', language)}
                  </div>
                </section>
              ))}

              {/* Custom prompt */}
              <section className="rounded-lg border border-[var(--panel-border)] bg-fxos-bg-lighter p-4">
                <div className="mb-1 flex items-baseline justify-between gap-2">
                  <div>
                    <div className="text-sm font-semibold text-fxos-text">
                      {t('promptStudio.customPrompt', language)}
                    </div>
                    <div className="text-xs text-fxos-text-muted">
                      {t('promptStudio.customPromptHint', language)}
                    </div>
                  </div>
                  <span className="shrink-0 text-[10px] font-mono text-fxos-text-muted">
                    {t('promptStudio.charCount', language, {
                      count: customPrompt.length,
                    })}
                  </span>
                </div>
                <textarea
                  value={customPrompt}
                  onChange={(e) => {
                    setCustomPrompt(e.target.value)
                    setSavedFlag(false)
                  }}
                  rows={4}
                  placeholder={t('promptStudio.customPrompt', language)}
                  className="mt-2 w-full resize-none rounded-lg border border-[var(--panel-border)] bg-fxos-bg px-3 py-2 font-mono text-sm text-fxos-text outline-none placeholder:text-fxos-text-muted/50 focus:border-fxos-gold/60"
                />
                <div className="mt-1 text-[10px] text-fxos-text-muted/70">
                  {t('promptStudio.emptyFallsBack', language)}
                </div>
              </section>
            </div>

            {/* Right: preview */}
            <aside className="space-y-4 lg:sticky lg:top-6 lg:self-start">
              <section className="rounded-lg border border-[var(--panel-border)] bg-fxos-bg-lighter p-4">
                <div className="mb-3 flex items-center justify-between">
                  <div className="text-sm font-semibold text-fxos-text">
                    {t('promptStudio.preview', language)}
                  </div>
                  <button
                    onClick={handleCopy}
                    disabled={!preview || previewLoading}
                    className="inline-flex items-center gap-1.5 rounded-lg border border-[var(--panel-border)] px-2.5 py-1.5 text-xs text-fxos-text-muted hover:text-fxos-text disabled:opacity-50"
                  >
                    {copiedFlag ? (
                      <Check className="h-3.5 w-3.5 text-fxos-success" />
                    ) : (
                      <ClipboardCopy className="h-3.5 w-3.5" />
                    )}
                    {copiedFlag
                      ? t('promptStudio.copied', language)
                      : t('promptStudio.copy', language)}
                  </button>
                </div>

                <div className="mb-3 grid gap-3 sm:grid-cols-2">
                  <label className="space-y-1">
                    <span className="text-xs text-fxos-text-muted">
                      {t('promptStudio.accountEquity', language)}
                    </span>
                    <input
                      type="number"
                      min={1}
                      value={equity}
                      onChange={(e) =>
                        setEquity(Math.max(1, Number(e.target.value) || 1))
                      }
                      className="w-full rounded-lg border border-[var(--panel-border)] bg-fxos-bg px-3 py-2 text-sm text-fxos-text"
                    />
                  </label>
                  <label className="space-y-1">
                    <span className="text-xs text-fxos-text-muted">
                      {t('promptStudio.variant', language)}
                    </span>
                    <select
                      value={variant}
                      onChange={(e) => setVariant(e.target.value as Variant)}
                      className="w-full rounded-lg border border-[var(--panel-border)] bg-fxos-bg px-3 py-2 text-sm text-fxos-text"
                    >
                      {VARIANT_OPTIONS.map((v) => (
                        <option key={v} value={v}>
                          {t(
                            `promptStudio.variant${v.charAt(0).toUpperCase()}${v.slice(1)}`,
                            language
                          )}
                        </option>
                      ))}
                    </select>
                  </label>
                </div>

                {previewError && (
                  <div className="mb-3 rounded-lg border border-fxos-danger/30 bg-fxos-danger/10 px-3 py-2 text-xs text-fxos-danger">
                    {previewError}
                  </div>
                )}

                {lintWarnings.length > 0 && (
                  <div className="mb-3 rounded-lg border border-fxos-gold/30 bg-fxos-gold/10 px-3 py-2 text-xs">
                    <div className="mb-1 font-semibold text-fxos-gold">
                      {t('promptStudio.lintWarnings', language) ||
                        'Prompt Lint Warnings'}
                    </div>
                    <ul className="list-inside list-disc space-y-0.5 text-fxos-text/80">
                      {lintWarnings.map((w, i) => (
                        <li key={i}>{w}</li>
                      ))}
                    </ul>
                  </div>
                )}

                <pre className="max-h-[420px] overflow-auto whitespace-pre-wrap rounded-lg border border-[var(--panel-border)] bg-fxos-bg-deeper p-3 font-mono text-xs leading-5 text-fxos-text/90">
                  {previewLoading
                    ? t('promptStudio.previewLoading', language)
                    : preview || t('promptStudio.previewLoading', language)}
                </pre>
              </section>

              {/* Token estimate */}
              <section className="rounded-lg border border-[var(--panel-border)] bg-fxos-bg-lighter p-4">
                <div className="mb-2 text-sm font-semibold text-fxos-text">
                  {t('promptStudio.tokenEstimate', language)}
                </div>
                {estimateError && (
                  <div className="mb-2 text-xs text-fxos-danger">
                    {estimateError}
                  </div>
                )}
                {estimate ? (
                  <div className="space-y-2">
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-fxos-text-muted">
                        {t('promptStudio.tokenTotal', language)}
                      </span>
                      <span className="font-mono text-fxos-text">
                        {estimate.total.toLocaleString()}
                      </span>
                    </div>
                    {estimate.breakdown && (
                      <div className="space-y-1 text-xs">
                        {Object.entries(estimate.breakdown).map(
                          ([key, value]) =>
                            typeof value === 'number' && value > 0 ? (
                              <div
                                key={key}
                                className="flex items-center justify-between text-fxos-text-muted"
                              >
                                <span>{key.replace(/_/g, ' ')}</span>
                                <span className="font-mono">
                                  {value.toLocaleString()}
                                </span>
                              </div>
                            ) : null
                        )}
                      </div>
                    )}
                    {estimate.model_limits &&
                      estimate.model_limits.length > 0 && (
                        <div className="space-y-1 border-t border-[var(--panel-border)] pt-2 text-xs">
                          {estimate.model_limits.map((ml) => (
                            <div
                              key={ml.name}
                              className="flex items-center justify-between"
                            >
                              <span className="text-fxos-text-muted">
                                {ml.name}
                              </span>
                              <span
                                className={`font-mono ${estimateLevelClass(ml.level)}`}
                              >
                                {ml.usage_pct}%
                              </span>
                            </div>
                          ))}
                        </div>
                      )}
                  </div>
                ) : (
                  <div className="text-xs text-fxos-text-muted">
                    {t('promptStudio.previewLoading', language)}
                  </div>
                )}
              </section>

              {/* Language contract note */}
              <div className="rounded-lg border border-fxos-gold/25 bg-fxos-gold/5 px-3 py-2.5 text-xs leading-5 text-fxos-text-muted">
                {t('promptStudio.englishOnly', language)}
              </div>

              {/* AI real-run test */}
              <section className="rounded-lg border border-[var(--panel-border)] bg-fxos-bg-lighter p-4">
                <div className="mb-3 flex items-center justify-between">
                  <div className="text-sm font-semibold text-fxos-text">
                    {t('promptStudio.aiTest', language)}
                  </div>
                  <button
                    type="button"
                    onClick={() => void runTest()}
                    disabled={testLoading || !selectedModelId}
                    className="inline-flex items-center gap-1.5 rounded-lg bg-fxos-gold px-2.5 py-1.5 text-xs font-semibold text-fxos-bg transition-colors hover:bg-fxos-gold/90 disabled:cursor-not-allowed disabled:opacity-45"
                  >
                    {testLoading ? (
                      <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    ) : (
                      <RefreshCw className="h-3.5 w-3.5" />
                    )}
                    {testLoading
                      ? t('promptStudio.testRunning', language)
                      : t('promptStudio.runTest', language)}
                  </button>
                </div>

                <label className="space-y-1">
                  <span className="text-xs text-fxos-text-muted">
                    {t('promptStudio.aiModelLabel', language)}
                  </span>
                  <select
                    value={selectedModelId}
                    onChange={(e) => setSelectedModelId(e.target.value)}
                    className="w-full rounded-lg border border-[var(--panel-border)] bg-fxos-bg px-3 py-2 text-sm text-fxos-text"
                  >
                    {models.length === 0 ? (
                      <option value="">
                        {t('promptStudio.noModels', language)}
                      </option>
                    ) : (
                      models.map((m) => (
                        <option key={m.id} value={m.id}>
                          {m.name} ({m.provider})
                        </option>
                      ))
                    )}
                  </select>
                </label>

                {testError && (
                  <div className="mt-3 rounded-lg border border-fxos-danger/30 bg-fxos-danger/10 px-3 py-2 text-xs text-fxos-danger">
                    {testError}
                  </div>
                )}

                {testResponse && (
                  <div className="mt-3 space-y-2">
                    {testResponse.candidate_count !== undefined && (
                      <div className="text-xs text-fxos-text-muted">
                        {t('promptStudio.candidateCount', language)}:{' '}
                        {testResponse.candidate_count}
                      </div>
                    )}
                    <div className="text-xs font-semibold text-fxos-text-muted">
                      {t('promptStudio.aiResponse', language)}
                    </div>
                    <pre className="max-h-[320px] overflow-auto whitespace-pre-wrap rounded-lg border border-[var(--panel-border)] bg-fxos-bg-deeper p-3 font-mono text-xs leading-5 text-fxos-text/90">
                      {testResponse.ai_response ||
                        t('promptStudio.noResponse', language)}
                    </pre>
                  </div>
                )}
              </section>
            </aside>
          </div>
        )}
      </div>
    </DeepVoidBackground>
  )
}

export default PromptStudioPage
