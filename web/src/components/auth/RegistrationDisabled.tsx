import { useNavigate } from 'react-router-dom'
import { useLanguage } from '../../contexts/LanguageContext'
import { t } from '../../i18n/translations'

export function RegistrationDisabled() {
  const { language } = useLanguage()
  const navigate = useNavigate()

  const handleBackToLogin = () => {
    navigate('/login')
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-fxos-bg text-fxos-text">
      <div className="text-center max-w-md px-6">
        <img
          src="/icons/fxos.svg"
          alt="FXOS Logo"
          className="w-16 h-16 mx-auto mb-4"
        />
        <h1 className="text-2xl font-semibold mb-3">
          {t('registrationClosed', language)}
        </h1>
        <p className="text-sm text-fxos-text-muted">
          {t('registrationClosedMessage', language)}
        </p>
        <button
          className="mt-6 px-4 py-2 rounded text-sm font-semibold transition-colors bg-fxos-gold text-fxos-bg hover:opacity-90"
          onClick={handleBackToLogin}
        >
          {t('backToLogin', language)}
        </button>
      </div>
    </div>
  )
}
