import type { FAQCategoryContent } from './faqData'

/**
 * 中文 FAQ 内容树。结构与英文树一致(不含 icon,icon 由 faqData.ts
 * 按分类 id 统一挂载)。代码片段用反引号包裹,渲染时转为等宽样式。
 */
export const faqZhCategories: FAQCategoryContent[] = [
  // ───────────────────────── 入门 ─────────────────────────
  {
    id: 'getting-started',
    title: '快速上手',
    items: [
      {
        id: 'what-is-fxos',
        question: 'FXOS 是什么?',
        blocks: [
          {
            type: 'p',
            text: 'FXOS 是一个开源、自托管的 AI 交易终端。旗舰模式是 FXOS Autopilot(自动驾驶):一个 AI 代理读取 Claw402.ai 信号面板,用 Signal Lab 和清算结构验证候选币种,用原始 K 线确认时机,并在 Hyperliquid 上执行——全部运行在你自己的机器上,你的密钥永远不会离开你的服务器。',
          },
          {
            type: 'p',
            text: '除了 Autopilot,你还可以在 Strategy Studio(策略工坊)中构建自定义策略,让多个 AI 交易员并行运行,并对比它们的表现。',
          },
        ],
      },
      {
        id: 'what-do-i-need',
        question: '启动 Autopilot 之前需要准备什么?',
        blocks: [
          {
            type: 'p',
            text: '两个已入金的账户——配置页的引导式启动会带你走完这两步:',
          },
          {
            type: 'list',
            items: [
              'AI 费用钱包:一个 Base 链上的 USDC 钱包,用于支付 AI 模型和市场数据调用费用。启动至少需要 `1 USDC`。',
              '一个已授予交易授权、至少有 `12 USDC` 可用保证金的 Hyperliquid 账户。',
            ],
          },
          {
            type: 'p',
            text: '启动按钮会运行一次服务端预检,逐项检查所有前置条件,并直接指出缺失的那一步,所以你不可能启动一个配置到一半的机器人。',
          },
        ],
      },
      {
        id: 'which-markets',
        question: '它可以交易哪些市场?',
        blocks: [
          {
            type: 'p',
            text: 'Autopilot 交易 Hyperliquid 永续合约:加密货币主流币(BTC、ETH、SOL 等),加上覆盖美股、指数、大宗商品和外汇的 xyz 合成市场——一个账户就能让 AI 拥有多资产宇宙。',
          },
          {
            type: 'p',
            text: '在 Strategy Studio 中手动配置的交易员还可以连接 Binance、Bybit、OKX、Bitget、KuCoin、Gate、Aster 和 Lighter。',
          },
        ],
      },
      {
        id: 'ai-models',
        question: '它使用哪些 AI 模型?需要 API 密钥吗?',
        blocks: [
          {
            type: 'p',
            text: '不需要 API 密钥。FXOS 通过 Claw402 按量付费基础设施路由推理:你的 AI 费用钱包用 Base USDC 按次付费,终端按需调用支持的模型(DeepSeek 及其他前沿模型)。',
          },
          {
            type: 'p',
            text: '高级用户仍然可以在 配置 → 模型 中接入自己的服务商密钥(OpenAI、Claude、Gemini、DeepSeek、Qwen、Grok、Kimi,或任何 OpenAI 兼容端点)。',
          },
        ],
      },
      {
        id: 'is-it-profitable',
        question: '它会赚钱吗?',
        blocks: [
          {
            type: 'p',
            text: '没有人能保证这一点,任何向你打包票的人都值得怀疑。AI 执行的是一个系统化流程,但市场充满对抗,过去的表现永远不能保证未来的结果。',
          },
          {
            type: 'p',
            text: '看板在业绩展示上刻意诚实:它区分已实现与未实现盈亏,展示费用损耗链条(毛利润 − 费用 = 净利润)、盈利因子,以及基于你真实初始余额计算的最大回撤。盯住这些数字,从小资金开始,只用你能承受亏损的钱交易。',
          },
          {
            type: 'note',
            text: '交易存在重大的亏损风险。FXOS 是软件,不是投资建议。',
          },
        ],
      },
    ],
  },

  // ───────────────────────── 启动与钱包 ─────────────────────────
  {
    id: 'launch-wallets',
    title: '启动与钱包',
    items: [
      {
        id: 'ai-fee-wallet',
        question: '什么是 AI 费用钱包?',
        blocks: [
          {
            type: 'p',
            text: '一个 Base 链上专用的 EVM 钱包,用于支付 AI 模型调用和付费市场数据(x402 微支付)。它与你的交易保证金完全分离——永远不会触碰 Hyperliquid。',
          },
          {
            type: 'list',
            items: [
              '引导式设置会为你创建它(或复用已有的钱包)。',
              '只需向其地址存入 Base 网络上的 USDC。',
              '启动至少需要 `1 USDC`;余额显示会在入金后自动刷新。',
              '一个典型周期通常花费几分之一美分到几美分,取决于模型。',
            ],
          },
        ],
      },
      {
        id: 'fee-wallet-private-key',
        question: 'AI 费用钱包的私钥存放在哪里?',
        blocks: [
          {
            type: 'p',
            text: '密钥在服务器本地生成,以 AES-256 加密后存储在你自己的数据库中,并在引导界面只展示一次。请做好备份——如果丢失数据库,它无法恢复。',
          },
          {
            type: 'note',
            text: '这个钱包里只放费用资金。它的存在是为了支付 AI 调用,而不是存放积蓄。',
          },
        ],
      },
      {
        id: 'hyperliquid-authorization',
        question: 'Hyperliquid 授权是怎么工作的?安全吗?',
        blocks: [
          {
            type: 'p',
            text: 'FXOS 使用 Hyperliquid 代理钱包,你的主钱包密钥永远不会被共享。连接流程包含四个签名步骤:',
          },
          {
            type: 'steps',
            items: [
              '连接你的 EVM 钱包(Rabby、MetaMask、OKX、Coinbase Wallet)。',
              '批准一个全新生成的 FXOS 代理钱包——有效期 180 天,仅限交易。',
              '批准 builder 费用(一笔小额逐单费用,用于维持平台运行)。',
              '将代理密钥保存到你的 FXOS 服务器(加密存储)。',
            ],
          },
          {
            type: 'p',
            text: '代理钱包只能开仓和平仓,其他什么都做不了。它无法提款,你的保证金始终留在你自己的 Hyperliquid 账户内。',
          },
        ],
      },
      {
        id: 'launch-preflight',
        question: '启动预检会检查什么?',
        blocks: [
          {
            type: 'p',
            text: '在创建或修改任何东西之前,服务器会用实时数据验证整条链路:',
          },
          {
            type: 'list',
            items: [
              'AI 模型已启用且有可用凭证。',
              'AI 费用钱包密钥有效,Base USDC 余额至少 `1 USDC`(链上实时查询)。',
              'Hyperliquid 账户已授权(代理 + builder 费用)且可访问。',
              '交易资金:至少 `12 USDC`(含持仓中的权益)。',
            ],
          },
          {
            type: 'p',
            text: '每一项失败的检查都会给出确切的修复方法,并深链到对应的引导步骤。同样的检查在每次启动时都会在服务端强制执行,UI 无法被绕过。',
          },
        ],
      },
      {
        id: 'relaunch-behavior',
        question: '再次点击 Launch 会发生什么?',
        blocks: [
          {
            type: 'p',
            text: '启动是幂等的。如果 FXOS Autopilot 已存在,启动器会用当前策略配置更新它并重启——绝不会创建重复实例。如果机器人正处于一个周期中,重启可能需要最多一分钟,UI 会等待它完成。',
          },
        ],
      },
      {
        id: 'deposit-not-showing',
        question: '我存入了 USDC,但余额仍然显示为零。',
        blocks: [
          {
            type: 'list',
            items: [
              'AI 费用钱包:确认你在 Base 网络上把 USDC 发送到了显示的准确地址。余额约缓存 30 秒,设置面板每几秒自动重新检查一次。',
              'Hyperliquid:入金会进入你自己的 Hyperliquid 账户;余额步骤轮询实时账户状态。拿不准时,在引导面板里点 Refresh。',
              '如果链上 RPC 暂时不可达,面板会把余额标记为"未知"而不是零——等一分钟再试。',
            ],
          },
        ],
      },
    ],
  },

  // ───────────────────────── 交易与执行 ─────────────────────────
  {
    id: 'trading',
    title: '交易与执行',
    items: [
      {
        id: 'autopilot-pipeline',
        question: 'Autopilot 策略一步步是怎么运作的?',
        blocks: [
          {
            type: 'p',
            text: '每个周期都运行同一个四阶段漏斗——每个阶段都可能淘汰候选,只有通过全部四关的设置才会被交易:',
          },
          {
            type: 'steps',
            items: [
              '构建候选宇宙——拉取实时 Claw402.ai 排名,取排名靠前的候选(默认 10 个),覆盖加密货币主流币和 xyz 合成市场(美股、指数、大宗商品、外汇),每个候选带方向偏好和信号 z-score。',
              '验证每个候选——获取它的 Signal Lab 深度信号,以及价格附近的成本/清算结构:清算簇和成本基线能说明一波行情前方有燃料还是有阻力墙。',
              '确认时机——读取原始 15 分钟 OHLCV K 线(30 根),确认入场是在顺应结构,而不是追一波已经拉开的行情。',
              '决策与仓位——只有通过置信度阈值(默认 `78/100`)、风险回报比约 `3:1` 的设置才会在 10 倍杠杆下开仓;平仓永远先于开仓执行,每个周期都会同时评估多空两个方向的持仓。',
            ],
          },
          {
            type: 'p',
            text: '在 AI 之外还有第五层:硬风控(持仓数量上限、杠杆上限、保证金上限、交易节流)会拒绝任何违反它们的决策,无论模型有多自信。',
          },
        ],
      },
      {
        id: 'data-sources',
        question: '它使用哪些数据?哪些部分是付费的?',
        blocks: [
          {
            type: 'list',
            items: [
              'Claw402.ai 信号数据——排名面板、每个交易对的 Signal Lab 深度信号,以及市场净流入。这些是付费端点,按调用次数从你的 AI 费用钱包扣 USDC(x402 微支付)。',
              '成本/清算热力图——每个市场的聚合持仓成本和清算簇结构。',
              'Hyperliquid 市场数据——原始 OHLCV K 线和实时 L2 订单簿(免费公开数据)。',
              '通过代理钱包获取你的账户——权益、可用保证金、带盈亏的持仓。',
              '它自己的历史——已平仓交易回馈胜率、盈利因子和回撤到下一次提示中,让 AI 了解自己近期的状态。',
            ],
          },
          {
            type: 'note',
            text: '看板会放慢轮询付费 Claw402 端点(每几分钟一次)以节省你的费用钱包——行情面板则始终由免费数据源保持实时。',
          },
        ],
      },
      {
        id: 'decision-cycle',
        question: 'AI 多久做一次决策?',
        blocks: [
          {
            type: 'p',
            text: 'Autopilot 每 5–15 分钟运行一次扫描周期,具体取决于你启动时的配置(每个交易员可配置,最短 3 分钟)。第一个周期在启动后立即开始;单个周期通常需要 30–60 秒,因为 AI 在决策前会读取完整市场上下文。',
          },
        ],
      },
      {
        id: 'what-ai-sees',
        question: '每个周期 AI 能看到哪些信息?',
        blocks: [
          {
            type: 'list',
            items: [
              '你的账户:权益、可用保证金、带盈亏的持仓。',
              'Claw402 排名面板:带方向偏好的候选宇宙。',
              '每个候选的 Signal Lab 深度信号和成本/清算结构。',
              '用于确认时机的原始 OHLCV K 线。',
              '它自己的战绩:胜率、盈利因子、回撤、近期交易。',
            ],
          },
          {
            type: 'p',
            text: '每个周期都会保存为一条决策记录——看板上的执行日志(Execution Log)展示推理链、动作和任何被拦截的订单。',
          },
        ],
      },
      {
        id: 'leverage-and-risk',
        question: '它使用什么杠杆和风控?',
        blocks: [
          {
            type: 'p',
            text: 'Autopilot 默认使用 10 倍全仓保证金。硬风控运行在 AI 之外,AI 无法覆盖它们:',
          },
          {
            type: 'list',
            items: [
              '策略配置中的持仓数量上限——达到上限后拒绝新开仓。',
              '按资产类别区分的杠杆上限(BTC/ETH 对比山寨币)。',
              '交易节流阻止无效换手,例如开仓几分钟后平掉一个几乎没动的仓位。',
              '安全模式(见下)在 AI 本身出问题时保护持仓。',
            ],
          },
        ],
      },
      {
        id: 'safe-mode',
        question: '什么是安全模式?',
        blocks: [
          {
            type: 'p',
            text: '如果 AI 连续失败 3 个周期(服务商故障、费用钱包余额不足、响应异常),交易员进入安全模式:不再开新仓,已有持仓保留保护,循环继续重试。下一次 AI 调用成功时会自动退出安全模式。',
          },
          {
            type: 'p',
            text: '安全模式会以横幅形式显示在看板上并附上原因,绝不会悄无声息地发生。',
          },
        ],
      },
      {
        id: 'fee-wallet-empty-mid-run',
        question: 'AI 费用钱包在运行中途耗尽会怎样?',
        blocks: [
          {
            type: 'p',
            text: 'AI 调用开始失败,并显示清晰的"余额不足"状态。看板显示一条带钱包余额的持续红色横幅;连续三次失败后,机器人进入安全模式。向费用钱包补充 Base USDC,交易员会自动恢复——无需重启。',
          },
        ],
      },
      {
        id: 'trading-fees',
        question: '我需要支付哪些费用?',
        blocks: [
          {
            type: 'list',
            items: [
              '每笔订单的 Hyperliquid 交易手续费,加上已批准的 builder 费用。',
              '按次从费用钱包扣除的 AI/数据成本(每个周期几美分)。',
            ],
          },
          {
            type: 'p',
            text: '费用是高频策略的隐形杀手。看板的统计条展示完整链条——毛已实现盈亏,减去费用,等于净盈亏——让你一眼看出费用是否在吞噬利润。',
          },
        ],
      },
      {
        id: 'stop-and-manual',
        question: '如何停止机器人或手动平仓?',
        blocks: [
          {
            type: 'list',
            items: [
              '停止:使用配置页交易员列表上的 Stop 按钮。停止会暂停决策循环;已开持仓保持开启,由你自己管理。',
              '手动平仓:从看板持仓面板平掉任何仓位——手动平仓会同步回持仓历史。',
              '紧急情况:你随时可以直接在 Hyperliquid 上管理仓位;FXOS 永远不会把你锁在自己的账户之外。',
            ],
          },
        ],
      },
    ],
  },

  // ───────────────────────── 看板与指标 ─────────────────────────
  {
    id: 'dashboard',
    title: '看板与指标',
    items: [
      {
        id: 'metrics-meaning',
        question: '顶部指标具体是什么意思?',
        blocks: [
          {
            type: 'list',
            items: [
              '权益(Equity)——包含未实现盈亏的实时账户价值。',
              '总盈亏(含未实现)——权益对比你的初始余额;随持仓波动。',
              '已实现盈亏(已平仓)——仅已完结交易的结果;胜率、盈利因子和夏普比率都基于它计算。',
              '盈利因子——已平仓交易的毛盈利 ÷ 毛亏损;大于 1.0 表示已平仓账户净盈利。',
              '最大回撤——已实现权益曲线最深的峰值到谷底跌幅,基于你真实的初始余额计算。',
            ],
          },
        ],
      },
      {
        id: 'pl-contradiction',
        question: '为什么总盈亏为正,已实现盈亏却是负的?',
        blocks: [
          {
            type: 'p',
            text: '它们衡量的是不同的东西。已实现盈亏只统计已平仓交易;总盈亏还包含仍持仓的未实现收益。一个机器人的已平仓交易可能是亏损的,但它的持仓中带着足够多的未实现利润,使总盈亏翻绿——反过来也一样。查看 毛/费/净 链条,看看已实现结果中有多少是费用拖累。',
          },
        ],
      },
      {
        id: 'execution-log',
        question: '在哪里能看到 AI 为什么做(或拒绝)某件事?',
        blocks: [
          {
            type: 'p',
            text: '执行日志(Execution Log)面板列出每个周期:采取的动作、AI 调用耗时,以及任何被拦截的订单和触发的具体防护(节流、持仓上限、风控)。完整的推理链会随每条决策记录一起保存。',
          },
        ],
      },
    ],
  },

  // ───────────────────────── 安全 ─────────────────────────
  {
    id: 'security',
    title: '安全',
    items: [
      {
        id: 'key-storage',
        question: '我的密钥是如何存储的?',
        blocks: [
          {
            type: 'list',
            items: [
              '所有机密(代理密钥、费用钱包密钥、交易所 API 密钥)都以 AES-256 加密存储在你自己的数据库中。',
              '可选的 RSA 传输加密保护浏览器与服务器之间传输的机密。',
              'FXOS 是自托管的:不会向任何第三方服务器发送数据。代码开源,可审计。',
            ],
          },
        ],
      },
      {
        id: 'can-fxos-steal-funds',
        question: 'FXOS 能提取或盗走我的资金吗?',
        blocks: [
          {
            type: 'p',
            text: '不能。在 Hyperliquid 上,FXOS 只持有代理钱包,按协议设计它只能交易、不能提款。你的保证金始终留在你自己主钱包控制下的账户里。',
          },
          {
            type: 'note',
            text: '如果改用中心化交易所,请只给 API 密钥交易权限——关闭提现并设置 IP 白名单。',
          },
        ],
      },
      {
        id: 'registration-model',
        question: '为什么别人无法在我的实例上注册?',
        blocks: [
          {
            type: 'p',
            text: '按设计,一个实例是单操作者模式:第一个注册的账户成为操作者,注册随即关闭("System already initialized")。这防止陌生人向你暴露在公网的部署创建账户。每个操作者运行一个实例。',
          },
        ],
      },
    ],
  },

  // ───────────────────────── 自托管与排障 ─────────────────────────
  {
    id: 'self-hosting',
    title: '自托管与排障',
    items: [
      {
        id: 'how-to-install',
        question: '如何安装 FXOS?',
        blocks: [
          {
            type: 'p',
            text: 'Linux/macOS 上一行命令(通过 Docker 安装并启动一切):',
          },
          {
            type: 'list',
            items: [
              '脚本:`curl -fsSL https://raw.githubusercontent.com/onecany/fxos/main/scripts/install.sh | bash`',
              'Docker:下载 `docker-compose.prod.yml` 并运行 `docker compose -f docker-compose.prod.yml up -d`',
              'Windows:安装 Docker Desktop,然后使用上面的 Docker 方式。',
              '源码构建:Go 1.26+、Node 20+、TA-Lib(`brew install ta-lib` / `apt-get install libta-lib0-dev`),然后 `go run .` 和 `npm --prefix web run dev`。',
            ],
          },
          {
            type: 'p',
            text: '然后打开 `http://127.0.0.1:3000`——Web UI 在 3000 端口,API 在 8080 端口。',
          },
        ],
      },
      {
        id: 'how-to-update',
        question: '如何升级?',
        blocks: [
          {
            type: 'p',
            text: '重新运行安装脚本,或用 Docker:`docker compose -f docker-compose.prod.yml pull && docker compose -f docker-compose.prod.yml up -d`。你的数据库和密钥位于挂载的 `data/` 目录中,升级后依然保留。运行中的交易员会在后端恢复后自动重启。',
          },
        ],
      },
      {
        id: 'launch-blocked',
        question: '启动被某项失败的检查拦截了,怎么办?',
        blocks: [
          {
            type: 'p',
            text: '读一下提示:每次预检失败都会说明修复方法,并把你导向对应的设置步骤——给 AI 钱包充值、完成 Hyperliquid 授权,或存入交易用 USDC。余额是实时复查的,所以一旦你修好,启动就能通过。',
          },
        ],
      },
      {
        id: 'exchange-unreachable',
        question: '交易所账户显示"凭证无效"或"不可用"。',
        blocks: [
          {
            type: 'list',
            items: [
              '凭证无效:代理授权已过期(180 天)或保存的密钥已失效——重新连接 Hyperliquid 钱包;该流程提供一键续期。',
              '不可用:交易所 API 未响应;账户状态缓存 30 秒,等待后刷新即可。',
              '中心化交易所密钥:确认已开启交易权限、IP 白名单,以及合约/永续访问权限。',
            ],
          },
        ],
      },
      {
        id: 'where-are-logs',
        question: '日志在哪里?',
        blocks: [
          {
            type: 'list',
            items: [
              '后端:`docker logs fxos-trading`(或运行 `go run .` 的终端)。',
              '每个周期的 AI 推理与错误:看板上的执行日志。',
              '前端构建/运行问题:浏览器开发者工具控制台。',
            ],
          },
        ],
      },
      {
        id: 'port-conflicts',
        question: '端口 3000 或 8080 已被占用。',
        blocks: [
          {
            type: 'p',
            text: '停止冲突的服务,或在 compose 文件中重新映射发布端口(例如前端用 `"3100:80"`,API 用 `"8180:8080"`),然后重启容器。',
          },
        ],
      },
    ],
  },

  // ───────────────────────── 参与贡献 ─────────────────────────
  {
    id: 'contributing',
    title: '参与贡献',
    items: [
      {
        id: 'how-to-contribute',
        question: '如何贡献代码?',
        blocks: [
          {
            type: 'links',
            links: [
              { label: '路线图(Roadmap)', href: 'https://github.com/orgs/onecany/projects/3' },
              { label: '任务看板(Task Dashboard)', href: 'https://github.com/orgs/onecany/projects/5' },
              { label: 'CONTRIBUTING.md', href: 'https://github.com/onecany/fxos/blob/dev/docs/CONTRIBUTING.md' },
            ],
          },
          {
            type: 'steps',
            items: [
              '从上面的看板挑一个任务(按 good first issue / help wanted 筛选)并评论 "assign me"。',
              'Fork 仓库并从 `dev` 分支切出:`git checkout -b feat/your-topic`。',
              '遵循 Conventional Commits;推送前运行 `npm --prefix web run lint && npm --prefix web run build`。',
              '向 `onecany/fxos:dev` 发起 PR,引用对应 issue(`Closes #123`),UI 改动请附带截图。',
            ],
          },
        ],
      },
      {
        id: 'bounty-program',
        question: '有悬赏计划吗?',
        blocks: [
          {
            type: 'p',
            text: '有——部分精选 issue 带有现金悬赏,定期贡献者还能获得徽章、优先审核和测试版访问权限。',
          },
          {
            type: 'links',
            links: [
              { label: '带 bounty 标签的 issue', href: 'https://github.com/onecany/fxos/labels/bounty' },
              { label: '悬赏认领模板', href: 'https://github.com/onecany/fxos/blob/dev/.github/ISSUE_TEMPLATE/bounty_claim.md' },
            ],
          },
        ],
      },
      {
        id: 'report-bugs',
        question: '如何报告 bug?',
        blocks: [
          {
            type: 'p',
            text: '使用模板提交 GitHub issue:你做了什么、发生了什么、后端日志(`docker logs fxos-trading`)和截图。疑似安全问题,请按照 SECURITY.md 中的负责任披露流程,而不是公开 issue。',
          },
          {
            type: 'links',
            links: [
              { label: '新建 issue', href: 'https://github.com/onecany/fxos/issues/new/choose' },
              { label: 'SECURITY.md', href: 'https://github.com/onecany/fxos/blob/dev/docs/SECURITY.md' },
            ],
          },
        ],
      },
    ],
  },
]
