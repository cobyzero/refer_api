package middleware

import "testing"

func TestDefaultAuditAction(t *testing.T) {
	if got := DefaultAuditAction("PATCH", "/api/users/123"); got != "PATCH /api/users/123" {
		t.Errorf("got %q", got)
	}
	if got := DefaultAuditAction("POST", ""); got != "POST /" {
		t.Errorf("got %q", got)
	}
}

func TestEntityFromPath(t *testing.T) {
	cases := map[string]string{
		"/api/users/123":   "users",
		"/api/orders":      "orders",
		"/api/auth/login":  "auth",
		"/health":          "",
		"/socket.io/poll":  "",
		"api/stars/redeem": "stars",
	}
	for path, want := range cases {
		if got := entityFromPath(path); got != want {
			t.Errorf("entityFromPath(%q) = %q, want %q", path, got, want)
		}
	}
}
