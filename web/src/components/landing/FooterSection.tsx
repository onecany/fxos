import { Github, Send, ExternalLink } from 'lucide-react'
import { t, Language } from '../../i18n/translations'
import { OFFICIAL_LINKS } from '../../constants/branding'

interface FooterSectionProps {
  language: Language
}

export default function FooterSection({ language }: FooterSectionProps) {
  const links = {
    social: [
      { name: 'GitHub', href: OFFICIAL_LINKS.github, icon: Github },
      { name: 'Telegram', href: OFFICIAL_LINKS.telegram, icon: Send },
    ],
    resources: [
      {
        name: language === 'zh' ? 'Documentation' : 'Documentation',
        href: 'https://github.com/onecany/fxos/blob/main/README.md',
      },
      { name: 'Issues', href: 'https://github.com/onecany/fxos/issues' },
      { name: 'Pull Requests', href: 'https://github.com/onecany/fxos/pulls' },
    ],
    supporters: [
      { name: 'Binance', href: 'https://www.binance.com/join?ref=FXOSENG' },
      { name: 'Bybit', href: 'https://partner.bybit.com/b/83856' },
      { name: 'OKX', href: 'https://www.okx.com/join/1865360' },
      { name: 'Bitget', href: 'https://www.bitget.com/referral/register?from=referral&clacCode=c8a43172' },
      { name: 'Gate.io', href: 'https://www.gatenode.xyz/share/VQBGUAxY' },
      { name: 'KuCoin', href: 'https://www.kucoin.com/r/broker/CXEV7XKK' },
      { name: 'Hyperliquid', href: 'https://app.hyperliquid.xyz/join/AITRADING' },
      { name: 'Aster DEX', href: 'https://www.asterdex.com/en/referral/fdfc0e' },
      { name: 'Lighter', href: 'https://app.lighter.xyz/?referral=68151432' },
    ],
  }

  return (
    <footer className="bg-fxos-bg border-t border-[rgba(45,212,191,0.16)]">
      <div className="max-w-6xl mx-auto px-4 py-8 md:py-12">
        {/* Top Section */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-8 md:gap-10 mb-8 md:mb-12">
          {/* Brand */}
          <div className="md:col-span-1">
            <div className="flex items-center gap-3 mb-4">
              <img src="/icons/fxos.svg" alt="FXOS Logo" className="w-8 h-8" />
              <span className="text-xl font-bold text-fxos-text">
                FXOS
              </span>
            </div>
            <p className="text-sm mb-6 text-fxos-text-muted">
              {t('futureStandardAI', language)}
            </p>
            {/* Social Icons */}
            <div className="flex items-center gap-3">
              {links.social.map((link) => (
                <a
                  key={link.name}
                  href={link.href}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="w-9 h-9 rounded-lg flex items-center justify-center bg-[rgba(255,255,255,0.06)] text-fxos-text transition-all hover:scale-110"
                  title={link.name}
                >
                  <link.icon className="w-4 h-4" />
                </a>
              ))}
            </div>
          </div>

          {/* Links */}
          <div>
            <h4 className="text-sm font-semibold mb-4 text-fxos-text">
              {t('links', language)}
            </h4>
            <ul className="space-y-3">
              {links.social.map((link) => (
                <li key={link.name}>
                  <a
                    href={link.href}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-sm transition-colors text-fxos-text-muted hover:text-fxos-danger"
                  >
                    {link.name}
                  </a>
                </li>
              ))}
            </ul>
          </div>

          {/* Resources */}
          <div>
            <h4 className="text-sm font-semibold mb-4 text-fxos-text">
              {t('resources', language)}
            </h4>
            <ul className="space-y-3">
              {links.resources.map((link) => (
                <li key={link.name}>
                  <a
                    href={link.href}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-sm text-fxos-text-muted transition-colors hover:text-fxos-danger inline-flex items-center gap-1"
                  >
                    {link.name}
                    <ExternalLink className="w-3 h-3 opacity-50" />
                  </a>
                </li>
              ))}
            </ul>
          </div>

          {/* Supporters */}
          <div>
            <h4 className="text-sm font-semibold mb-4 text-fxos-text">
              {t('supporters', language)}
            </h4>
            <div className="flex flex-wrap gap-2">
              {links.supporters.map((link) => (
                <a
                  key={link.name}
                  href={link.href}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-xs border border-[rgba(45,212,191,0.16)] bg-[rgba(255,255,255,0.04)] rounded px-3 py-1.5 text-fxos-text transition-all hover:border-fxos-accent hover:text-fxos-accent hover:bg-fxos-accent/10"
                >
                  {link.name}
                </a>
              ))}
            </div>
          </div>
        </div>

        {/* Bottom Section */}
        <div className="pt-6 text-center text-xs border-t border-[rgba(45,212,191,0.16)] text-fxos-text-muted">
          <p className="mb-2">{t('footerTitle', language)}</p>
          <p className="text-fxos-text-muted/80">{t('footerWarning', language)}</p>
        </div>
      </div>
    </footer>
  )
}
