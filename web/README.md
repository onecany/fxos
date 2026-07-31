# FXOS Web Dashboard

AI 自动交易监控终端，支持中英印尼三语切换。

## 技术栈

- **React 18** - UI 框架
- **TypeScript** - 类型安全
- **Vite** - 构建工具
- **Tailwind CSS** - 样式框架
- **SWR** - 数据获取和缓存
- **Zustand** - 状态管理
- **Recharts** - 图表库
- **Framer Motion** - 动画
- **Lucide React** - 图标库

## 安装依赖

```bash
npm install
```

## 运行开发服务器

```bash
npm run dev
```

访问 http://localhost:3000

## 构建生产版本

```bash
npm run build
```

## 语言支持

支持三种语言切换：**中文 / English / Bahasa Indonesia**

点击顶部导航栏右侧的语言按钮即可切换，所有页面文本同步更新。

## 功能特性

### Terminal Dashboard（终端仪表盘）
- **编排拓扑** - 实时展示 AI 决策流程（净流入 → 信号 → 执行 → 持仓）
- **风险雷达** - 净敞口、杠杆、保证金、集中度、回撤、未实现盈亏
- **执行日志** - 实时 AI 决策记录，可点击查看详情弹窗
- **成本/清算图** - 多头/空头成本和清算层级可视化
- **订单簿** - Hyperliquid L2 深度数据
- **信号矩阵** - Vergex/Claw402 信号排名热力图
- **市场净流入** - 资金流向排名
- **按币种统计** - 各币种交易/胜率/盈亏
- **边际分析** - 持仓时间 vs 盈亏分析

### 策略工作室
- **策略创建/编辑** - 可视化配置交易参数
- **策略风格预设** - 稳健/均衡/积极三种模式
- **Claw402 集成** - 一键加载 Vergex 看板
- **AI 模型配置** - 支持 DeepSeek、OpenAI、Claude 等

### 设置页面
- **账户管理** - 修改密码
- **AI 模型配置** - 添加/编辑/删除模型
- **交易所配置** - 添加/编辑/删除交易所
- **Telegram 配置** - 连接 Telegram 机器人

### 决策详情弹窗
点击执行日志中的任意周期，弹出详情弹窗：
- **摘要** - 账户状态 + 决策卡片 + 执行日志
- **CoT 分析** - 结构化 7 步决策流程
- **系统提示词** - 完整系统提示词
- **用户提示词** - 完整用户提示词
- **原始响应** - AI 原始响应

## 项目结构

```
web/
├── src/
│   ├── components/
│   │   ├── auth/           # 登录/注册组件
│   │   ├── charts/         # 图表组件
│   │   ├── common/         # 通用组件（HeaderBar, LanguageSwitcher等）
│   │   ├── landing/        # 落地页组件
│   │   ├── modals/         # 弹窗组件
│   │   ├── strategy/       # 策略组件
│   │   ├── terminal/       # 终端仪表盘组件
│   │   └── trader/         # 交易员管理组件
│   ├── contexts/           # React Context（语言、认证）
│   ├── i18n/               # 国际化翻译
│   ├── lib/                # API 客户端、工具函数
│   ├── pages/              # 页面组件
│   ├── router/             # 路由配置
│   ├── stores/             # Zustand 状态管理
│   └── types/              # TypeScript 类型定义
├── index.html
├── vite.config.ts
├── tailwind.config.js
└── package.json
```

## 注意事项

1. **确保后端 API 服务已启动**（默认端口 8080）
2. **Node.js 版本要求**：>= 20.0.0
3. **网络连接**：需要访问交易所 API
