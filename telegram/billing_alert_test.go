package telegram

import (
	"testing"
	"time"
)

func TestBillingAlertThrottle_AllowsOncePerWindow(t *testing.T) {
	th := newBillingAlertThrottle(time.Hour)

	if !th.allow("deepseek", 402) {
		t.Error("first alert for deepseek|402 should be allowed")
	}
	if th.allow("deepseek", 402) {
		t.Error("second alert within window should be throttled")
	}

	// Different provider/status pairs are independent
	if !th.allow("deepseek", 500) {
		t.Error("deepseek|500 is a different key, should be allowed")
	}
	if !th.allow("openai", 402) {
		t.Error("openai|402 is a different key, should be allowed")
	}
}

func TestBillingAlertThrottle_AllowsAfterWindowElapses(t *testing.T) {
	th := newBillingAlertThrottle(50 * time.Millisecond)

	if !th.allow("deepseek", 402) {
		t.Fatal("first alert should be allowed")
	}
	if th.allow("deepseek", 402) {
		t.Fatal("second alert within window should be throttled")
	}

	time.Sleep(60 * time.Millisecond)

	if !th.allow("deepseek", 402) {
		t.Error("alert after window elapses should be allowed again")
	}
}
