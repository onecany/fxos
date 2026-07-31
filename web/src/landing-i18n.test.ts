import { describe, it, expect } from 'vitest'
import { t, translations, type Language } from './i18n/translations'

const LANDING_KEYS = [
  'landing.chip',
  'landing.heroTitle1',
  'landing.heroTitle2',
  'landing.heroSubtitle',
  'landing.liveFeeds',
  'landing.marketUs',
  'landing.marketCommodities',
  'landing.marketFx',
  'landing.marketPreIpo',
  'landing.ctaStart',
  'landing.ctaSee',
  'landing.promise',
  'landing.tickerGlobal',
  'landing.tickerRouting',
  'landing.tickerLatency',
  'landing.tickerModel',
  'landing.statStars',
  'landing.statForks',
  'landing.statContributors',
  'landing.statCommunity',
  'landing.assetClassSelect',
  'landing.proTraders1',
  'landing.proTraders2',
  'landing.agentTagline',
  'landing.classLabel',
  'landing.apyLabel',
  'landing.winLabel',
  'landing.riskLabel',
  'landing.riskHigh',
  'landing.riskMed',
  'landing.riskLow',
  'landing.initialize',
  'landing.agent1Desc',
  'landing.agent2Desc',
  'landing.agent3Desc',
  'landing.deployEyebrow',
  'landing.deployTitle1',
  'landing.deployTitle2',
  'landing.deployDesc',
  'landing.step1',
  'landing.step2',
  'landing.step3',
  'landing.featureInstallLabel',
  'landing.featureInstallDesc',
  'landing.featureKeysLabel',
  'landing.featureKeysDesc',
  'landing.feedStable',
  'landing.logSignal',
  'landing.logRisk',
  'landing.logMacro',
  'landing.logSys',
]

describe('landing page translations', () => {
  it('provides every landing key in all three languages', () => {
    const langs: Language[] = ['en', 'zh', 'id']
    for (const lang of langs) {
      for (const key of LANDING_KEYS) {
        const value = t(key, lang)
        expect(value, `${key} missing in ${lang}`).toBeTruthy()
        expect(value, `${key} unresolved in ${lang}`).not.toMatch(/^landing\./)
      }
    }
  })

  it('switches per-language values (zh differs from en)', () => {
    expect(t('landing.ctaStart', 'zh')).not.toBe(t('landing.ctaStart', 'en'))
    expect(t('landing.heroTitle2', 'zh')).not.toBe(
      t('landing.heroTitle2', 'en')
    )
    expect(t('landing.initialize', 'zh')).toBe('初始化')
  })

  it('resolves placeholder params in log templates', () => {
    expect(t('landing.logSignal', 'en', { z: '0.921' })).toContain('0.921')
    expect(t('landing.logRisk', 'zh', { pair: 'AAPL-USDC' })).toContain(
      'AAPL-USDC'
    )
    expect(t('landing.logMacro', 'id', { ms: 4 })).toContain('4')
  })
})
