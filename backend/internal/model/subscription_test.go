package model

import (
	"testing"
	"time"
)

func TestSubscription_IsActive(t *testing.T) {
	now := time.Now()
	future := now.Add(24 * time.Hour)
	past := now.Add(-24 * time.Hour)

	cases := []struct {
		name string
		sub  Subscription
		want bool
	}{
		{
			name: "active with future expiry and traffic remaining",
			sub:  Subscription{Status: SubscriptionStatusActive, ExpiresAt: &future, TrafficLimit: 1000, TrafficUsed: 500},
			want: true,
		},
		{
			name: "expired by time",
			sub:  Subscription{Status: SubscriptionStatusActive, ExpiresAt: &past, TrafficLimit: 1000, TrafficUsed: 0},
			want: false,
		},
		{
			name: "exhausted by traffic",
			sub:  Subscription{Status: SubscriptionStatusActive, ExpiresAt: &future, TrafficLimit: 1000, TrafficUsed: 1000},
			want: false,
		},
		{
			name: "cancelled status",
			sub:  Subscription{Status: SubscriptionStatusCancelled, ExpiresAt: &future, TrafficLimit: 1000},
			want: false,
		},
		{
			name: "unlimited traffic active",
			sub:  Subscription{Status: SubscriptionStatusActive, ExpiresAt: &future, TrafficLimit: 0},
			want: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.sub.IsActive()
			if got != c.want {
				t.Errorf("IsActive() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestSubscription_RemainingTrafficGB(t *testing.T) {
	gb := int64(1024 * 1024 * 1024)

	if got := (&Subscription{TrafficLimit: 0}).RemainingTrafficGB(); got != -1 {
		t.Errorf("unlimited should return -1, got %v", got)
	}

	if got := (&Subscription{TrafficLimit: 5 * gb, TrafficUsed: 2 * gb}).RemainingTrafficGB(); got != 3.0 {
		t.Errorf("5-2 GB remaining: got %v want 3.0", got)
	}

	// Used > limit shouldn't go negative
	if got := (&Subscription{TrafficLimit: gb, TrafficUsed: 5 * gb}).RemainingTrafficGB(); got != 0 {
		t.Errorf("over-used: got %v want 0", got)
	}
}

func TestSubscription_DaysRemaining(t *testing.T) {
	if got := (&Subscription{}).DaysRemaining(); got != -1 {
		t.Errorf("nil expiry should return -1, got %v", got)
	}

	now := time.Now()
	in3d := now.Add(3 * 24 * time.Hour).Add(time.Hour) // a bit over 3 days
	if got := (&Subscription{ExpiresAt: &in3d}).DaysRemaining(); got != 3 {
		t.Errorf("DaysRemaining = %v, want 3", got)
	}

	past := now.Add(-1 * time.Hour)
	if got := (&Subscription{ExpiresAt: &past}).DaysRemaining(); got != 0 {
		t.Errorf("expired DaysRemaining = %v, want 0", got)
	}
}
