package mcp

// Provider name constants — kept in the mcp package so that client.go can
// reference them for default configuration without importing sub-packages.
// Provider sub-packages re-use these same values.
const (
	ProviderDeepSeek = "deepseek"
	ProviderOpenAI   = "openai"
	ProviderClaude   = "claude"
	ProviderQwen     = "qwen"
	ProviderGemini   = "gemini"
	ProviderGrok     = "grok"
	ProviderKimi     = "kimi"
	ProviderMiniMax  = "minimax"

	ProviderClaw402 = "claw402"
)

// Default provider endpoints and models, declared once here for unified
// management. Provider sub-packages reference these instead of declaring
// their own constants.
const (
	// Default DeepSeek configuration (used as fallback in NewClient)
	DefaultDeepSeekBaseURL = "https://api.deepseek.com"
	DefaultDeepSeekModel   = "deepseek-v4-flash"

	// Default OpenAI configuration
	DefaultOpenAIBaseURL = "https://api.openai.com/v1"
	DefaultOpenAIModel   = "gpt-5.4"

	// Default Claude (Anthropic) configuration
	DefaultClaudeBaseURL = "https://api.anthropic.com/v1"
	DefaultClaudeModel   = "claude-opus-4-6"

	// Default Qwen configuration (used by WithQwenConfig convenience option)
	DefaultQwenBaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	DefaultQwenModel   = "qwen3-max"

	// Default Gemini configuration
	DefaultGeminiBaseURL = "https://generativelanguage.googleapis.com/v1beta/openai"
	DefaultGeminiModel   = "gemini-3.1-pro"

	// Default Grok (xAI) configuration
	DefaultGrokBaseURL = "https://api.x.ai/v1"
	DefaultGrokModel   = "grok-3-latest"

	// Default Kimi (Moonshot) configuration
	DefaultKimiBaseURL = "https://api.moonshot.ai/v1" // Global endpoint (use api.moonshot.cn for China)
	DefaultKimiModel   = "moonshot-v1-auto"

	// Default MiniMax configuration (used by WithMiniMaxConfig convenience option)
	DefaultMiniMaxBaseURL = "https://api.minimax.io/v1"
	DefaultMiniMaxModel   = "MiniMax-M2.7"
)
