package xui

import (
	"errors"
	"testing"
)

func TestBytesToTotalGB(t *testing.T) {
	cases := []struct {
		name  string
		input int64
		want  int64
	}{
		{"zero is unlimited", 0, 0},
		{"negative treated as unlimited", -1, 0},
		{"sub-GB clamped up to 1 GB", 500 * 1024 * 1024, bytesInGB},
		{"exactly 1 byte clamped up", 1, bytesInGB},
		{"1 GB stays 1 GB", bytesInGB, bytesInGB},
		{"5 GB stays 5 GB", 5 * bytesInGB, 5 * bytesInGB},
		{"50 GB stays 50 GB", 50 * bytesInGB, 50 * bytesInGB},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := bytesToTotalGB(c.input)
			if got != c.want {
				t.Fatalf("bytesToTotalGB(%d) = %d, want %d", c.input, got, c.want)
			}
		})
	}
}

func TestLooksLikeAuthIssue(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   []byte
		want   bool
	}{
		{"401 unauthorized", 401, []byte(`{"success":false}`), true},
		{"403 forbidden", 403, []byte(`{}`), true},
		{"302 redirect to login", 302, []byte(``), true},
		{"empty body 200", 200, []byte(``), true},
		{"HTML login page", 200, []byte(`<!DOCTYPE html><html>...`), true},
		{"plain text", 200, []byte(`unauthorized`), true},
		{"valid JSON", 200, []byte(`{"success":true,"obj":null}`), false},
		{"valid JSON array", 200, []byte(`[]`), false},
		{"500 with json (real error, not auth)", 500, []byte(`{"success":false,"msg":"db down"}`), false},
		{"404 not found is not auth", 404, []byte(``), false},
		{"404 with body is not auth", 404, []byte(`{"success":false,"msg":"not found"}`), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := looksLikeAuthIssue(c.status, c.body)
			if got != c.want {
				t.Fatalf("looksLikeAuthIssue(%d, %q) = %v, want %v", c.status, c.body, got, c.want)
			}
		})
	}
}

func TestIsDuplicateEmail(t *testing.T) {
	yes := []string{
		"Email Already Exists",
		"email already exists",
		"duplicate email",
		"Email уже существует",
		"client with email exists in another inbound",
	}
	no := []string{
		"",
		"some other error",
		"client not found",
		"timeout",
	}
	for _, msg := range yes {
		if !isDuplicateEmail(errors.New(msg)) {
			t.Errorf("isDuplicateEmail(%q) = false, want true", msg)
		}
	}
	for _, msg := range no {
		if msg == "" {
			if isDuplicateEmail(nil) {
				t.Errorf("isDuplicateEmail(nil) = true, want false")
			}
			continue
		}
		if isDuplicateEmail(errors.New(msg)) {
			t.Errorf("isDuplicateEmail(%q) = true, want false", msg)
		}
	}
}

func TestIsClientNotFound(t *testing.T) {
	yes := []string{
		"update client failed: status=404, body=...",
		"client not found",
		"object not found",
		"клиент не найден в системе",
		"Error getting traffics. (Inbound Not Found For Email: ghost)",
	}
	no := []string{
		"",
		"timeout",
		"duplicate email",
		"some other error",
	}
	for _, msg := range yes {
		if !isClientNotFound(errors.New(msg)) {
			t.Errorf("isClientNotFound(%q) = false, want true", msg)
		}
	}
	for _, msg := range no {
		if msg == "" {
			if isClientNotFound(nil) {
				t.Errorf("isClientNotFound(nil) = true, want false")
			}
			continue
		}
		if isClientNotFound(errors.New(msg)) {
			t.Errorf("isClientNotFound(%q) = true, want false", msg)
		}
	}
}

func TestClampMaxDevices(t *testing.T) {
	if got := clampMaxDevices(0); got != 3 {
		t.Errorf("clampMaxDevices(0) = %d, want 3", got)
	}
	if got := clampMaxDevices(-1); got != 3 {
		t.Errorf("clampMaxDevices(-1) = %d, want 3", got)
	}
	if got := clampMaxDevices(5); got != 5 {
		t.Errorf("clampMaxDevices(5) = %d, want 5", got)
	}
}
