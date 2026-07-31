# 🤖 FXOS — Open-Source AI Trading Terminal / 开源 AI 交易终端

FXOS is a self-hosted, open-source AI trading terminal. The flagship mode — **FXOS Autopilot** — reads the Claw402.ai signal board, verifies candidates with Signal Lab and liquidation structure, confirms timing with raw candles, and executes on Hyperliquid. Everything runs on your own machine; your keys never leave your server.

FXOS 是一个开源、自托管的 AI 交易终端。旗舰模式 **FXOS Autopilot(自动巡航)** 读取 Claw402.ai 信号面板,用 Signal Lab 与清算结构验证候选,用原始 K 线确认时机,并在 Hyperliquid 上执行——全部运行在你自己的机器上,密钥永不离开你的服务器。

---

## ✨ Features / 特性

| | |
|---|---|
| 🧠 **AI Autopilot** | End-to-end AI trading loop: signal → verify → timing → execute, with hard risk controls that the AI cannot override / 端到端 AI 交易闭环:信号 → 验证 → 时机 → 执行,叠加 AI 无法越过的硬风控 |
| 🛠️ **Strategy Studio** | Build custom strategies, run multiple AI traders side by side / 构建自定义策略,让多个 AI 交易员并行运行 |
| 🌍 **Multi-Market** | Hyperliquid perpetuals (BTC/ETH/SOL) + xyz synthetic markets (US stocks, indices, commodities, FX) / Hyperliquid 永续 + xyz 合成市场(美股、指数、大宗商品、外汇) |
| 🔌 **Multi-Exchange** | Hyperliquid, Binance, Bybit, OKX, Bitget, KuCoin, Gate, Aster, Lighter / 多交易所支持 |
| 🌓 **Theme Toggle** | One-click dark/light switch, sci-fi neon aesthetic with softened palette — dark = deep navy + teal neon, light = cool moonlight / 一键深浅色切换,科幻霓虹风格,柔和配色——深色 = 深海军蓝 + 青色霓虹,浅色 = 冷调月白 |
| 🌐 **Trilingual UI** | 中文 / English / Bahasa Indonesia — every page including the landing page and FAQ follows the switch / 中文 / English / 印尼语——包括首页与 FAQ 在内的所有页面随切换联动 |
| 🔒 **Self-Hosted & Secure** | AES-256 encrypted keys, Hyperliquid agent wallet (trade-only, no withdrawals), optional RSA transport encryption / AES-256 加密密钥,Hyperliquid 代理钱包(仅交易、不可提款),可选 RSA 传输加密 |
| 🖥️ **Terminal Dashboard** | Real-time execution log, decision reasoning chain, position management / 实时执行日志、决策推理链、持仓管理 |

---

## 🚀 Quick Start / 快速开始

### Option 1: One-line install (Linux/macOS, via Docker) / 一行安装(推荐)

```bash
curl -fsSL https://raw.githubusercontent.com/onecany/fxos/dev/scripts/install.sh | bash
```

### Option 2: Docker Compose

```bash
docker compose -f docker-compose.prod.yml up -d
```

### Option 3: Build from source / 源码构建

Requirements / 依赖: **Go 1.26+**, **Node 20+**, **TA-Lib** (`brew install ta-lib` / `apt-get install libta-lib0-dev`)

```bash
go run .                    # backend API on :8080
npm --prefix web run dev    # web UI on :3000
```

Open `http://127.0.0.1:3000` — the guided launch takes you to your first AI trade in about five minutes; ~$13 is enough to start.

打开 `http://127.0.0.1:3000`——引导式启动带你在大约五分钟内完成第一笔 AI 交易;约 13 美元即可起步。

**Before launch / 启动前准备:**
- AI fee wallet: a Base-chain USDC wallet (≥ `1 USDC`) for AI/data calls / Base 链 USDC 钱包(≥ `1 USDC`),用于支付 AI 调用与数据费用
- Hyperliquid account with trading authorization (≥ `12 USDC` margin) / 已授权的 Hyperliquid 账户(≥ `12 USDC` 保证金)

---

## 🎨 Theme & Language / 主题与语言

- **Theme / 主题**: the sun/moon button in the top bar toggles dark ⇄ light instantly, persisted in localStorage. The terminal components keep their dark look in both themes, like a real terminal. / 顶栏太阳/月亮按钮一键切换深色 ⇄ 浅色,localStorage 记忆;终端组件在两种主题下保持深色,像一台真终端。
- **Language / 语言**: top bar switches 中文 / EN / ID across the entire site, including the landing page and FAQ. / 顶栏切换 中文 / EN / ID,覆盖全站,含首页与 FAQ。

---

## 🗂️ Project Structure / 项目结构

```
fxos/
├── api/            # HTTP API handlers / API 接口
├── kernel/         # Strategy engine, AI pipeline / 策略引擎、AI 流水线
├── manager/        # Trader lifecycle management / 交易员生命周期管理
├── provider/       # Data providers (claw402, nofx, exchanges) / 数据提供方
├── store/          # SQLite persistence / 数据持久化
├── market/         # Market data / 行情数据
├── web/            # React frontend (React 18 + Vite + Tailwind) / 前端
└── docs/      # Documentation center / 文档中心
```

Web frontend / 前端: `web/` — React 18 · TypeScript · Vite · Tailwind CSS · lightweight-charts · framer-motion · SWR · Zustand

---

## 📚 Documentation / 文档中心

- [Documentation Center / 文档中心](docs/README.md) — all docs in one place / 全部文档入口
- [Getting Started / 快速开始](docs/getting-started/README.md)
- [User Guides / 使用指南](docs/guides/README.md)
- [Architecture / 架构](docs/architecture/README.md)
- [Roadmap / 路线图](docs/roadmap/README.md)
- [FAQ](docs/guides/faq.en.md) · [常见问题](docs/guides/faq.zh-CN.md)

---

## 🤝 Community & Support / 社区与支持

- 💬 [Telegram Developer Community](https://t.me/fxos_dev_community)
- 🐛 [Report Issues](https://github.com/onecany/fxos/issues)
- 🛡️ Security vulnerabilities → follow the responsible disclosure process in `docs/legal/` / 安全问题请走 `docs/legal/` 中的负责任披露流程

---

## 📄 License / 许可证

Apache License 2.0. See [LICENSE](LICENSE) for details.

**Disclaimer / 免责声明:** Trading involves substantial risk. FXOS is software, not investment advice. Trade only with funds you can afford to lose. / 交易存在重大亏损风险。FXOS 是软件,不是投资建议。请只用你能承受亏损的资金交易。
