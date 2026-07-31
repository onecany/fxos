import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useLanguage } from '../../contexts/LanguageContext'
import { t } from '../../i18n/translations'
import { Header } from '../common/Header'
import { ArrowLeft, KeyRound, Copy, Check } from 'lucide-react'
import { toast } from 'sonner'

const RESET_PASSWORD_COMMAND = 'fxos reset-password --email you@example.com'

export function ResetPasswordPage() {
  const { language } = useLanguage()
  const navigate = useNavigate()
  const [copied, setCopied] = useState(false)

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(RESET_PASSWORD_COMMAND)
      setCopied(true)
      toast.success(t('copy', language))
      setTimeout(() => setCopied(false), 2000)
    } catch {
      toast.error(t('copy', language))
    }
  }

  return (
    <div className="min-h-screen bg-fxos-bg">
      <Header simple />

      <div
        className="flex items-center justify-center"
        style={{ minHeight: 'calc(100vh - 80px)' }}
      >
        <div className="w-full max-w-md">
          {/* Back to Login */}
          <button
            onClick={() => navigate('/login')}
            className="flex items-center gap-2 mb-6 text-sm text-fxos-text-muted hover:text-fxos-accent transition-colors"
          >
            <ArrowLeft className="w-4 h-4" />
            {t('backToLogin', language)}
          </button>

          {/* Logo */}
          <div className="text-center mb-8">
            <div className="w-16 h-16 mx-auto mb-4 flex items-center justify-center rounded-full bg-fxos-gold/10">
              <KeyRound className="w-8 h-8 text-fxos-gold" />
            </div>
            <h1 className="text-2xl font-bold text-fxos-text">
              {t('resetPasswordTitle', language)}
            </h1>
          </div>

          {/* CLI recovery instructions */}
          <div className="rounded-lg p-6 bg-[var(--panel-bg)] border border-[rgba(45,212,191,0.16)]">
            <p className="text-sm leading-relaxed mb-4 text-fxos-text">
              {t('resetPasswordCliIntro', language)}
            </p>

            <div className="flex items-center justify-between gap-3 rounded px-3 py-3 font-mono text-xs bg-[var(--panel-bg)] border border-[rgba(45,212,191,0.18)]">
              <code className="break-all text-fxos-accent">
                {RESET_PASSWORD_COMMAND}
              </code>
              <button
                type="button"
                onClick={handleCopy}
                className="shrink-0 btn-icon text-fxos-text-muted"
                aria-label={t('copy', language)}
              >
                {copied ? (
                  <Check className="w-4 h-4 text-fxos-success" />
                ) : (
                  <Copy className="w-4 h-4" />
                )}
              </button>
            </div>

            <p className="text-xs leading-relaxed mt-4 text-fxos-text-muted">
              {t('resetPasswordCliSecurityNote', language)}
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}
