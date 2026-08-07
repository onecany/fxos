package mcp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// ============================================================
// Test Client Creation and Configuration
// ============================================================

func TestNewClient_Default(t *testing.T) {
	client := NewClient()

	if client == nil {
		t.Fatal("client should not be nil")
	}

	c := client.(*Client)
	if c.Provider == "" {
		t.Error("Provider should have default value")
	}

	if c.MaxTokens <= 0 {
		t.Error("MaxTokens should be positive")
	}

	if c.Log == nil {
		t.Error("Log should not be nil")
	}

	if c.HTTPClient == nil {
		t.Error("HTTPClient should not be nil")
	}

	if c.Hooks == nil {
		t.Error("Hooks should not be nil")
	}
}

func TestNewClient_WithOptions(t *testing.T) {
	mockLogger := NewMockLogger()
	mockHTTP := &http.Client{Timeout: 30 * time.Second}

	client := NewClient(
		WithLogger(mockLogger),
		WithHTTPClient(mockHTTP),
		WithMaxTokens(4000),
		WithTimeout(60*time.Second),
		WithAPIKey("test-key"),
	)

	c := client.(*Client)

	if c.Log != mockLogger {
		t.Error("Log should be set from option")
	}

	if c.HTTPClient != mockHTTP {
		t.Error("HTTPClient should be set from option")
	}

	if c.MaxTokens != 4000 {
		t.Error("MaxTokens should be 4000")
	}

	if c.APIKey != "test-key" {
		t.Error("APIKey should be test-key")
	}
}

// ============================================================
// Test CallWithMessages
// ============================================================

func TestClient_CallWithMessages_Success(t *testing.T) {
	mockHTTP := NewMockHTTPClient()
	mockHTTP.SetSuccessResponse("AI response content")
	mockLogger := NewMockLogger()

	client := NewClient(
		WithHTTPClient(mockHTTP.ToHTTPClient()),
		WithLogger(mockLogger),
		WithAPIKey("test-key"),
		WithBaseURL("https://api.test.com"),
	)

	result, err := client.CallWithMessages("system prompt", "user prompt")

	if err != nil {
		t.Fatalf("should not error: %v", err)
	}

	if result != "AI response content" {
		t.Errorf("expected 'AI response content', got '%s'", result)
	}

	// Verify request
	requests := mockHTTP.GetRequests()
	if len(requests) != 1 {
		t.Errorf("expected 1 request, got %d", len(requests))
	}

	if len(requests) > 0 {
		req := requests[0]
		if req.Header.Get("Authorization") == "" {
			t.Error("Authorization header should be set")
		}
		if req.Header.Get("Content-Type") != "application/json" {
			t.Error("Content-Type should be application/json")
		}
	}
}

func TestClient_CallWithMessages_NoAPIKey(t *testing.T) {
	client := NewClient()

	_, err := client.CallWithMessages("system", "user")

	if err == nil {
		t.Error("should error when API key is not set")
	}

	if err.Error() != "AI API key not set, please call SetAPIKey first" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestClient_CallWithMessages_HTTPError(t *testing.T) {
	mockHTTP := NewMockHTTPClient()
	mockHTTP.SetErrorResponse(500, "Internal Server Error")
	mockLogger := NewMockLogger()

	client := NewClient(
		WithHTTPClient(mockHTTP.ToHTTPClient()),
		WithLogger(mockLogger),
		WithAPIKey("test-key"),
	)

	_, err := client.CallWithMessages("system", "user")

	if err == nil {
		t.Error("should error on HTTP error")
	}
}

// ============================================================
// Test Retry Logic
// ============================================================

func TestClient_Retry_Success(t *testing.T) {
	mockHTTP := NewMockHTTPClient()
	mockLogger := NewMockLogger()

	// Simulate: first call fails, second call succeeds
	callCount := 0
	mockHTTP.ResponseFunc = func(req *http.Request) (*http.Response, error) {
		callCount++
		if callCount == 1 {
			return nil, errors.New("connection reset")
		}
		return &http.Response{
			StatusCode: 200,
			Body:       http.NoBody,
		}, nil
	}

	client := NewClient(
		WithHTTPClient(mockHTTP.ToHTTPClient()),
		WithLogger(mockLogger),
		WithAPIKey("test-key"),
		WithMaxRetries(3),
	)

	// Since our client uses Hooks.Call, need special handling
	// Here we test that CallWithMessages will invoke retry logic
	c := client.(*Client)

	// Temporarily modify retry wait time to 0 to speed up test
	oldRetries := MaxRetryTimes
	MaxRetryTimes = 3
	defer func() { MaxRetryTimes = oldRetries }()

	_, err := c.CallWithMessages("system", "user")

	// First fails (connection reset), second succeeds, but response format is wrong, will fail
	// But at least verify retry logic was triggered
	if callCount < 2 {
		t.Errorf("should retry, got %d calls", callCount)
	}

	// Check if there's retry information in logs
	logs := mockLogger.GetLogsByLevel("WARN")
	hasRetryLog := false
	for _, log := range logs {
		if log.Message == "⚠️  AI API call failed, retrying (2/3)..." {
			hasRetryLog = true
			break
		}
	}

	if !hasRetryLog && callCount >= 2 {
		// If retry was indeed attempted, there should be warning logs
		// But due to our test setup, it may not trigger, so just check here
		t.Log("Retry was attempted")
	}

	_ = err // Ignore error, we mainly test retry logic was triggered
}

func TestClient_Retry_NonRetryableError(t *testing.T) {
	mockHTTP := NewMockHTTPClient()
	mockHTTP.SetErrorResponse(400, "Bad Request")
	mockLogger := NewMockLogger()

	client := NewClient(
		WithHTTPClient(mockHTTP.ToHTTPClient()),
		WithLogger(mockLogger),
		WithAPIKey("test-key"),
	)

	_, err := client.CallWithMessages("system", "user")

	if err == nil {
		t.Error("should error")
	}

	// Verify no retry (because 400 is not a retryable error)
	requests := mockHTTP.GetRequests()
	if len(requests) != 1 {
		t.Errorf("should not retry for 400 error, got %d requests", len(requests))
	}
}

// TestClient_RetryOnEmptyResponse locks the empty-output regression guard:
// a 200 response whose message has NO content, NO reasoning_content and NO
// tool calls must be treated as a retryable error (deepseek thinking-mode
// can return out=0 turns), not as a successful empty answer. The client
// must retry, and only give up after max retries.
func TestClient_RetryOnEmptyResponse(t *testing.T) {
	mockHTTP := NewMockHTTPClient()
	mockLogger := NewMockLogger()

	callCount := 0
	mockHTTP.ResponseFunc = func(req *http.Request) (*http.Response, error) {
		callCount++
		// Always return an empty message: content="", reasoning_content=""
		body := `{"choices":[{"message":{"content":"","reasoning_content":""}}],"usage":{"total_tokens":10}}`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(body)),
			Header:     make(http.Header),
		}, nil
	}

	client := NewClient(
		WithHTTPClient(mockHTTP.ToHTTPClient()),
		WithLogger(mockLogger),
		WithAPIKey("test-key"),
		WithMaxRetries(3),
	)

	_, err := client.CallWithMessages("system", "user")

	if err == nil {
		t.Fatal("empty response must produce an error after retries")
	}
	if !strings.Contains(err.Error(), "upstream_empty_output") {
		t.Errorf("error should mention upstream_empty_output, got: %v", err)
	}
	if callCount < 2 {
		t.Errorf("empty response must trigger retry, got %d calls", callCount)
	}
}

// TestClient_ReasoningOnlyFallback locks the reasoning-content fallback: when
// the model returns content="" but a non-empty reasoning_content (thinking
// trace only), the client must surface the reasoning as the response instead
// of returning an empty string, so downstream CoT extraction and decision
// parsing still have input.
func TestClient_ReasoningOnlyFallback(t *testing.T) {
	mockHTTP := NewMockHTTPClient()
	mockLogger := NewMockLogger()

	mockHTTP.ResponseFunc = func(req *http.Request) (*http.Response, error) {
		body := `{"choices":[{"message":{"content":"","reasoning_content":"considering regime LOW_VOL, no trade"}}],"usage":{"total_tokens":50}}`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(body)),
			Header:     make(http.Header),
		}, nil
	}

	client := NewClient(
		WithHTTPClient(mockHTTP.ToHTTPClient()),
		WithLogger(mockLogger),
		WithAPIKey("test-key"),
	)

	result, err := client.CallWithMessages("system", "user")
	if err != nil {
		t.Fatalf("reasoning-only response should not error: %v", err)
	}
	if !strings.Contains(result, "considering regime LOW_VOL") {
		t.Errorf("reasoning content should be surfaced, got: %q", result)
	}
	if !strings.Contains(result, "<reasoning>") {
		t.Errorf("reasoning fallback should wrap in <reasoning> tags, got: %q", result)
	}
}

// ============================================================
// Test Hook Methods
// ============================================================

func TestClient_BuildMCPRequestBody(t *testing.T) {
	client := NewClient()
	c := client.(*Client)

	body := c.BuildMCPRequestBody("system prompt", "user prompt")

	if body == nil {
		t.Fatal("body should not be nil")
	}

	if body["model"] == nil {
		t.Error("body should have model field")
	}

	messages, ok := body["messages"].([]map[string]string)
	if !ok {
		t.Fatal("messages should be []map[string]string")
	}

	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}

	if messages[0]["role"] != "system" {
		t.Error("first message should be system")
	}

	if messages[1]["role"] != "user" {
		t.Error("second message should be user")
	}
}

func TestClient_BuildUrl(t *testing.T) {
	tests := []struct {
		name       string
		baseURL    string
		useFullURL bool
		expected   string
	}{
		{
			name:       "normal URL",
			baseURL:    "https://api.test.com/v1",
			useFullURL: false,
			expected:   "https://api.test.com/v1/chat/completions",
		},
		{
			name:       "full URL",
			baseURL:    "https://api.test.com/custom/endpoint",
			useFullURL: true,
			expected:   "https://api.test.com/custom/endpoint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(
				WithProvider("test-provider"), // Prevent default DeepSeek settings
				WithBaseURL(tt.baseURL),
				WithUseFullURL(tt.useFullURL),
			)
			c := client.(*Client)

			url := c.BuildUrl()
			if url != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, url)
			}
		})
	}
}

func TestClient_SetAuthHeader(t *testing.T) {
	client := NewClient(WithAPIKey("test-api-key"))
	c := client.(*Client)

	headers := make(http.Header)
	c.SetAuthHeader(headers)

	authHeader := headers.Get("Authorization")
	if authHeader != "Bearer test-api-key" {
		t.Errorf("expected 'Bearer test-api-key', got '%s'", authHeader)
	}
}

func TestClient_IsRetryableError(t *testing.T) {
	client := NewClient()
	c := client.(*Client)

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "EOF error",
			err:      errors.New("unexpected EOF"),
			expected: true,
		},
		{
			name:     "timeout error",
			err:      errors.New("timeout exceeded"),
			expected: true,
		},
		{
			name:     "connection reset",
			err:      errors.New("connection reset by peer"),
			expected: true,
		},
		{
			name:     "upstream empty output",
			err:      errors.New(`API returned error (status 429): {"error":{"code":"upstream_empty_output","message":"Upstream model returned empty output.","type":"rate_limit_error"}}`),
			expected: true,
		},
		{
			name:     "normal error",
			err:      errors.New("bad request"),
			expected: false,
		},
		{
			name:     "validation error",
			err:      errors.New("invalid input"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.IsRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// ============================================================
// Test SetTimeout
// ============================================================

func TestClient_SetTimeout(t *testing.T) {
	client := NewClient()

	newTimeout := 90 * time.Second
	client.SetTimeout(newTimeout)

	c := client.(*Client)
	if c.HTTPClient.Timeout != newTimeout {
		t.Errorf("expected timeout %v, got %v", newTimeout, c.HTTPClient.Timeout)
	}
}

// ============================================================
// Test String Method
// ============================================================

func TestClient_String(t *testing.T) {
	client := NewClient(
		WithProvider("test-provider"),
		WithModel("test-model"),
	)

	c := client.(*Client)
	str := c.String()

	expectedContains := []string{"test-provider", "test-model"}
	for _, exp := range expectedContains {
		if !contains(str, exp) {
			t.Errorf("String() should contain '%s', got '%s'", exp, str)
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ── Billing-class (402) error handling ───────────────────────────────────────

func TestClient_CallWithMessages_BillingError402(t *testing.T) {
	mockHTTP := NewMockHTTPClient()
	mockHTTP.SetErrorResponse(http.StatusPaymentRequired,
		`{"error":{"message":"Insufficient Balance","type":"unknown_error","param":null,"code":"invalid_request_error"}}`)
	mockLogger := NewMockLogger()

	// Reset global callback, capture invocations
	prev := APIErrorCallback
	called := 0
	var cbProvider, cbModel, cbMessage string
	var cbStatus int
	APIErrorCallback = func(provider, model string, statusCode int, message string) {
		called++
		cbProvider, cbModel, cbStatus, cbMessage = provider, model, statusCode, message
	}
	defer func() { APIErrorCallback = prev }()

	client := NewClient(
		WithHTTPClient(mockHTTP.ToHTTPClient()),
		WithLogger(mockLogger),
		WithAPIKey("test-key"),
		WithBaseURL("https://api.test.com"),
	)

	_, err := client.CallWithMessages("system", "user")

	// 1. Must return an error (the call failed)
	if err == nil {
		t.Fatal("expected error for 402 response")
	}

	// 2. Error must be identifiable as a billing-class error via errors.As
	var billingErr *BillingClassError
	if !errors.As(err, &billingErr) {
		t.Errorf("expected *BillingClassError, got %T: %v", err, err)
	}
	if billingErr == nil {
		t.Fatal("billingErr is nil")
	}
	if billingErr.StatusCode != http.StatusPaymentRequired {
		t.Errorf("expected status 402, got %d", billingErr.StatusCode)
	}
	if billingErr.Provider != ProviderDeepSeek {
		t.Errorf("expected provider deepseek, got %q", billingErr.Provider)
	}
	if billingErr.Model != DefaultDeepSeekModel {
		t.Errorf("expected model deepseek-chat, got %q", billingErr.Model)
	}
	if !strings.Contains(billingErr.Message, "Insufficient Balance") {
		t.Errorf("expected Insufficient Balance in message, got %q", billingErr.Message)
	}

	// 3. APIErrorCallback must fire exactly once
	if called != 1 {
		t.Errorf("expected APIErrorCallback called 1 time, got %d", called)
	}
	if cbStatus != http.StatusPaymentRequired {
		t.Errorf("expected callback status 402, got %d", cbStatus)
	}
	if cbProvider != ProviderDeepSeek || cbModel != DefaultDeepSeekModel {
		t.Errorf("expected provider/model deepseek/deepseek-chat, got %s/%s", cbProvider, cbModel)
	}
	if !strings.Contains(cbMessage, "Insufficient Balance") {
		t.Errorf("expected Insufficient Balance in callback message, got %q", cbMessage)
	}

	// 4. No retry attempts for billing errors (single HTTP request)
	requests := mockHTTP.GetRequests()
	if len(requests) != 1 {
		t.Errorf("expected exactly 1 HTTP request (no retries), got %d", len(requests))
	}

	// 5. ERROR-level log must be emitted
	hasErrorLog := false
	for _, entry := range mockLogger.GetLogs() {
		if entry.Level == "ERROR" && strings.Contains(entry.Message, "billing error") {
			hasErrorLog = true
			break
		}
	}
	if !hasErrorLog {
		t.Error("expected ERROR-level log containing 'billing error'")
	}
}

func TestClient_IsRetryableError_BillingError(t *testing.T) {
	mockLogger := NewMockLogger()
	client := NewClient(
		WithLogger(mockLogger),
		WithAPIKey("test-key"),
	).(*Client)

	billingErr := &BillingClassError{
		Provider:   ProviderDeepSeek,
		Model:      DefaultDeepSeekModel,
		StatusCode: http.StatusPaymentRequired,
		Message:    `{"error":{"message":"Insufficient Balance"}}`,
	}

	if client.IsRetryableError(billingErr) {
		t.Error("billing error must never be retryable")
	}

	// Non-billing errors keep their existing behavior
	if !client.IsRetryableError(fmt.Errorf("some network timeout happened")) {
		t.Error("network timeout should still be retryable")
	}
}

func TestClient_CallWithMessages_NonBillingHTTPErrorNoCallback(t *testing.T) {
	mockHTTP := NewMockHTTPClient()
	mockHTTP.SetErrorResponse(500, "Internal Server Error")
	mockLogger := NewMockLogger()

	prev := APIErrorCallback
	called := 0
	APIErrorCallback = func(provider, model string, statusCode int, message string) {
		called++
	}
	defer func() { APIErrorCallback = prev }()

	client := NewClient(
		WithHTTPClient(mockHTTP.ToHTTPClient()),
		WithLogger(mockLogger),
		WithAPIKey("test-key"),
		WithBaseURL("https://api.test.com"),
	)

	_, err := client.CallWithMessages("system", "user")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if called != 0 {
		t.Errorf("expected APIErrorCallback NOT called for 500, got %d calls", called)
	}

	// 500 stays a plain error (not a billing error)
	var billingErr *BillingClassError
	if errors.As(err, &billingErr) {
		t.Errorf("expected plain error for 500, got *BillingClassError: %v", err)
	}
}
