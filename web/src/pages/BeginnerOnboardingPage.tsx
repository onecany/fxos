import { useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ArrowRight, Copy, RefreshCw, Shield, Wallet, X } from 'lucide-react'
import { QRCodeSVG } from 'qrcode.react'
import { toast } from 'sonner'
import { useLanguage } from '../contexts/LanguageContext'
import { t } from '../i18n/translations'
import { api } from '../lib/api'
import type { BeginnerOnboardingResponse } from '../types'
import {
  setBeginnerWalletAddress,
  markBeginnerOnboardingCompleted,
} from '../lib/onboarding'

export function BeginnerOnboardingPage() {
  const { language } = useLanguage()
  const navigate = useNavigate()
  const [data, setData] = useState<BeginnerOnboardingResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [refreshingBalance, setRefreshingBalance] = useState(false)
  const hasRequestedRef = useRef(false)
  const isZh = language === 'zh'

  const loadOnboarding = async (showLoading: boolean) => {
    if (showLoading) {
      setLoading(true)
    } else {
      setRefreshingBalance(true)
    }

    setError('')
    try {
      const result = await api.prepareBeginnerOnboarding()
      setData(result)
      setBeginnerWalletAddress(result.address)
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : isZh
            ? t('failedPrepareWallet', language)
            : t('failedPrepareWallet', language)
      )
    } finally {
      if (showLoading) {
        setLoading(false)
      } else {
        setRefreshingBalance(false)
      }
    }
  }

  useEffect(() => {
    if (hasRequestedRef.current) {
      return
    }
    hasRequestedRef.current = true
    void loadOnboarding(true)
  }, [])

  // Poll the balance while the user is depositing so the page updates on its
  // own. Uses the lightweight current-wallet endpoint (server caches 30s), not
  // the heavier prepare call that re-writes .env.
  const walletAddress = data?.address
  useEffect(() => {
    if (!walletAddress) return
    let cancelled = false
    const timer = setInterval(() => {
      void api
        .getCurrentBeginnerWallet()
        .then((wallet) => {
          if (cancelled || !wallet.found || !wallet.balance_usdc) return
          setData((prev) =>
            prev && prev.address === wallet.address
              ? { ...prev, balance_usdc: wallet.balance_usdc! }
              : prev
          )
        })
        .catch(() => {
          // transient — the manual refresh button still works
        })
    }, 15000)
    return () => {
      cancelled = true
      clearInterval(timer)
    }
  }, [walletAddress])

  const noticeText = useMemo(
    () => t('thisWalletPaysForModel', language),
    [language]
  )

  const copyText = async (value: string) => {
    try {
      await navigator.clipboard.writeText(value)
      toast.success(t('addressCopied', language))
    } catch {
      toast.error(t('beginnerCopyFailed', language))
    }
  }

  const handleContinue = () => {
    markBeginnerOnboardingCompleted()
    navigate('/traders')
  }

  return (
    <div className="fixed inset-0 z-[80]">
      <div className="absolute inset-0 bg-black/58 backdrop-blur-[2px]" />
      <div className="relative flex min-h-screen items-start justify-center overflow-y-auto px-3 py-12 sm:px-6 md:items-center md:py-10">
        <button
          type="button"
          onClick={handleContinue}
          className="fixed right-4 top-4 z-10 inline-flex h-10 w-10 items-center justify-center rounded-full border border-[var(--panel-border)] bg-fxos-text/5 text-fxos-text-muted transition hover:border-[var(--panel-border)] hover:bg-fxos-text/10 hover:text-fxos-text sm:right-6 sm:top-6"
          aria-label={t('skip', language)}
        >
          <X className="h-5 w-5" />
        </button>
        <div className="w-full max-w-[1120px]">
          <div className="mb-4 flex flex-col gap-3 sm:mb-5 sm:gap-4 lg:flex-row lg:items-end lg:justify-between">
            <div className="flex items-center gap-3 sm:gap-4">
              <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-[18px] border border-fxos-gold/20 bg-fxos-gold/8 text-fxos-gold sm:h-14 sm:w-14 sm:rounded-[22px]">
                <Shield className="h-5 w-5 sm:h-6 sm:w-6" />
              </div>
              <div>
                <div className="text-[10px] font-semibold uppercase tracking-[0.2em] text-fxos-gold/80 sm:text-[11px] sm:tracking-[0.34em]">
                  {t('beginnerGuard', language)}
                </div>
                <h1 className="mt-1 max-w-[720px] text-[20px] font-bold leading-[1.04] tracking-[-0.03em] text-fxos-text sm:mt-2 sm:text-[27px] md:text-[35px] xl:text-[42px]">
                  {t('walletReady', language)}
                </h1>
              </div>
            </div>

            <div className="pb-1 text-[11px] tracking-[0.1em] text-fxos-text-muted sm:text-[12px] md:text-right md:whitespace-nowrap lg:pb-2 lg:text-[13px] lg:tracking-[0.12em]">
              Claw402 + DeepSeek <span className="mx-1 sm:mx-2">·</span>
              {t('payPerCall', language)}
            </div>
          </div>

          <div className="overflow-hidden rounded-[20px] border border-[var(--panel-border)] bg-fxos-bg-lighter shadow-lg backdrop-blur-2xl sm:rounded-[28px] lg:rounded-[32px]">
            {loading ? (
              <div className="flex min-h-[260px] items-center justify-center px-6 text-sm text-fxos-text-muted sm:min-h-[390px]">
                {t('preparingWallet', language)}
              </div>
            ) : data ? (
              <div className="grid md:grid-cols-[0.82fr_1.18fr]">
                <section className="flex flex-col justify-center px-5 py-6 sm:px-8 md:min-h-[430px]">
                  <div className="mx-auto w-full max-w-[248px] text-center">
                    <div className="mx-auto inline-flex rounded-[28px] border border-[var(--panel-border)] bg-white p-4 shadow-sm">
                      <QRCodeSVG value={data.address} size={164} level="M" />
                    </div>

                    <div className="mt-4 text-[15px] font-medium text-fxos-text">
                      {t('depositAddress', language)}
                    </div>

                    <div className="mt-3 flex items-center justify-between gap-2 rounded-xl border border-fxos-success/20 bg-fxos-success/10 px-3 py-2.5 sm:mt-4 sm:gap-3 sm:rounded-[24px] sm:px-5 sm:py-3.5">
                      <div className="text-left">
                        <div className="flex items-baseline gap-3 font-mono font-bold tracking-tight text-fxos-success">
                          <span className="text-[22px]">
                            {data.balance_usdc}
                          </span>
                          <span className="text-[20px]">USDC</span>
                        </div>
                      </div>
                      <button
                        type="button"
                        onClick={() => void loadOnboarding(false)}
                        disabled={refreshingBalance}
                        className="inline-flex h-12 w-12 items-center justify-center rounded-2xl border border-fxos-success/20 bg-fxos-bg-deeper text-fxos-success transition hover:bg-fxos-success/10 disabled:cursor-not-allowed disabled:opacity-60"
                        aria-label={t('refreshBalance', language)}
                      >
                        <RefreshCw
                          className={`h-4 w-4 ${refreshingBalance ? 'animate-spin' : ''}`}
                        />
                      </button>
                    </div>

                    <div className="mt-4 text-sm text-fxos-text-muted">
                      {t('balanceLasts', language)}
                    </div>

                    {/* the wall every true beginner hits: where does USDC come from? */}
                    <div className="mt-4 rounded-xl border border-fxos-gold/20 bg-fxos-gold/10 px-4 py-3 text-left text-[12px] leading-5 text-fxos-text sm:mt-5 sm:rounded-2xl sm:px-5 sm:py-4 sm:text-[13px] sm:leading-6">
                      <div className="mb-1 font-semibold">
                        {t('dontHaveUSDC', language)}
                      </div>
                      {language === 'zh' ? (
                        <>在 Binance、OKX 或 Coinbase 购买 USDC，然后提现到上方地址——交易所询问时选择 <b>Base 网络</b>。通常一分钟内到账。仅支持 Base 网络的 USDC。</>
                      ) : (
                        <>Buy USDC on Binance, OKX or Coinbase, then withdraw it to the address above — and pick the <b>Base network</b> when the exchange asks. It usually arrives in about a minute. Only send USDC on Base.</>
                      )}
                    </div>
                  </div>
                </section>

                <section className="border-t border-[var(--panel-border)] px-5 py-6 sm:px-8 md:border-l md:border-t-0 md:px-9">
                  <div className="space-y-5">
                    <div>
                      <div className="mb-3 flex items-center gap-2 text-sm font-medium text-fxos-gold">
                        <Wallet className="h-4 w-4" />
                        <span>{t('walletAddressLabel', language)}</span>
                      </div>
                      <div className="flex items-stretch gap-3">
                        <div className="min-w-0 flex-1 overflow-hidden rounded-xl border border-[var(--panel-border)] bg-fxos-bg-deeper px-3 py-2.5 font-mono text-[11px] text-fxos-text sm:rounded-2xl sm:px-5 sm:text-[13px] md:text-[14px]">
                          <div className="break-all">{data.address}</div>
                        </div>
                        <button
                          type="button"
                          onClick={() =>
                            copyText(data.address)
                          }
                          className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border border-[var(--panel-border)] bg-fxos-text/5 text-fxos-text transition hover:border-[var(--panel-border)] hover:bg-fxos-text/10 hover:text-fxos-text sm:h-14 sm:w-14 sm:rounded-2xl"
                          aria-label={t('copyAddress', language)}
                        >
                          <Copy className="h-5 w-5" />
                        </button>
                      </div>
                    </div>

                    <div className="pt-1">
                      <div className="mb-3 flex items-center gap-2 text-sm font-medium text-fxos-gold">
                        <Shield className="h-4 w-4" />
                        <span>
                          {t('privateKeyLabel', language)}
                        </span>
                      </div>
                      <div className="flex items-stretch gap-3">
                        <div className="min-w-0 flex-1 overflow-hidden rounded-[20px] border border-fxos-gold/20 bg-fxos-gold/10 px-4 py-3 font-mono text-[11px] leading-5 text-fxos-text sm:rounded-[24px] sm:px-5 sm:text-[13px] sm:leading-6">
                          <div className="break-all">
                            {data.private_key}
                          </div>
                        </div>
                        <div className="flex shrink-0 flex-col justify-end">
                          <button
                            type="button"
                            onClick={() =>
                              copyText(
                                data.private_key
                              )
                            }
                            className="inline-flex h-11 w-11 items-center justify-center rounded-xl border border-fxos-gold/20 bg-fxos-gold/10 text-fxos-gold transition hover:bg-fxos-gold/15 sm:h-14 sm:w-14 sm:rounded-2xl"
                            aria-label={t('copyPrivateKey', language)}
                          >
                            <Copy className="h-5 w-5" />
                          </button>
                        </div>
                      </div>
                    </div>

                    <div
                      className="rounded-xl border border-[var(--panel-border)] bg-fxos-bg-deeper px-4 py-3 text-[11px] leading-5 text-fxos-text-muted sm:rounded-[24px] sm:px-5 sm:py-3.5 sm:leading-6"
                    >
                      <span className="mr-2 text-fxos-text-muted">•</span>
                      {noticeText}
                    </div>

                    {data.env_warning ? (
                      <div className="rounded-2xl border border-fxos-gold/20 bg-fxos-gold/10 px-4 py-3 text-sm text-fxos-gold">
                        {data.env_warning}
                      </div>
                    ) : null}

                    {error ? (
                      <div className="rounded-2xl border border-fxos-danger/20 bg-fxos-danger/10 px-4 py-3 text-sm text-fxos-danger">
                        {error}
                      </div>
                    ) : null}

                    <button
                      type="button"
                      onClick={handleContinue}
                      className="mt-1 flex w-full items-center justify-center gap-3 rounded-[24px] bg-fxos-gold px-5 py-3.5 text-[16px] font-bold text-fxos-bg transition hover:bg-fxos-gold-highlight sm:text-[18px]"
                    >
                      <span>{t('continueSetup', language)}</span>
                      <ArrowRight className="h-5 w-5" />
                    </button>

                    {data.env_saved ? (
                      <div className="pt-1 text-xs text-fxos-text-muted">
                        {t('walletSavedToEnv', language, { path: data.env_path || '.env' })}
                      </div>
                    ) : null}
                  </div>
                </section>
              </div>
            ) : null}
          </div>
        </div>
      </div>
    </div>
  )
}
