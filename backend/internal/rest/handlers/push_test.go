package handlers

import "testing"

func TestIsAllowedPushEndpoint(t *testing.T) {
	allowed := []string{
		"https://fcm.googleapis.com/fcm/send/abc123",
		"https://android.googleapis.com/gcm/send/xyz",
		"https://updates.push.services.mozilla.com/wpush/v2/abc",
		"https://wns2-by3p.notify.windows.com/w/?token=abc",
		"https://web.push.apple.com/AB12cd",
	}
	for _, e := range allowed {
		if !isAllowedPushEndpoint(e) {
			t.Errorf("expected allowed: %s", e)
		}
	}

	denied := []string{
		"http://fcm.googleapis.com/fcm/send/abc", // not https
		"https://169.254.169.254/latest/meta-data",
		"https://localhost:6379/",
		"https://postgres:5432/",
		"https://fcm.googleapis.com.attacker.com/x", // suffix-spoof
		"https://evil.com/fcm/send/abc",
		"ftp://web.push.apple.com/x",
		"not-a-url",
		"",
	}
	for _, e := range denied {
		if isAllowedPushEndpoint(e) {
			t.Errorf("expected denied: %s", e)
		}
	}
}
