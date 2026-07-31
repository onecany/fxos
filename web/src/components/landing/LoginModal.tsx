import { motion } from 'framer-motion'
import { X } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { t, Language } from '../../i18n/translations'
interface LoginModalProps {
  onClose: () => void
  language: Language
}

export default function LoginModal({ onClose, language }: LoginModalProps) {
  const navigate = useNavigate()

  return (
    <motion.div
      className="fixed inset-0 z-50 flex items-center justify-center p-4"
      style={{ background: 'rgba(0, 0, 0, 0.7)' }}
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      onClick={onClose}
    >
      <motion.div
        className="relative max-w-md w-full rounded-3xl p-8 shadow-[0_0_30px_rgba(45,212,191,0.10)] bg-[var(--panel-bg)] border border-[rgba(45,212,191,0.18)]"
        initial={{ scale: 0.9, y: 50 }}
        animate={{ scale: 1, y: 0 }}
        exit={{ scale: 0.9, y: 50 }}
        onClick={(e) => e.stopPropagation()}
      >
        <motion.button
          onClick={onClose}
          className="absolute top-4 right-4 text-fxos-text-muted hover:text-fxos-text"
          whileHover={{ scale: 1.1, rotate: 90 }}
          whileTap={{ scale: 0.9 }}
        >
          <X className="w-6 h-6" />
        </motion.button>
        <h2 className="text-2xl font-bold mb-6 text-fxos-text">
          {t('accessFxosPlatform', language)}
        </h2>
        <p className="text-sm mb-6 text-fxos-text-muted">
          {t('loginRegisterPrompt', language)}
        </p>
        <div className="space-y-3">
          <motion.button
            onClick={() => {
              navigate('/login')
              onClose()
            }}
            className="block w-full px-6 py-3 rounded-lg font-semibold text-center bg-fxos-gold text-fxos-bg"
            whileHover={{
              scale: 1.05,
              boxShadow: '0 10px 30px var(--fxos-gold-glow)',
            }}
            whileTap={{ scale: 0.95 }}
          >
            {t('signIn', language)}
          </motion.button>
        </div>
      </motion.div>
    </motion.div>
  )
}
