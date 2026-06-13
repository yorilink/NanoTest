package auth

import "testing"

func TestDemoTokenVerifier(t *testing.T) {
	verifier := DemoTokenVerifier{}

	accountID, err := verifier.Verify("demo:10001")
	if err != nil {
		t.Fatal(err)
	}
	if accountID != 10001 {
		t.Fatalf("accountID = %d, want 10001", accountID)
	}

	for _, token := range []string{"", "10001", "demo:", "demo:abc", "demo:0"} {
		if _, err := verifier.Verify(token); err != ErrInvalidToken {
			t.Fatalf("Verify(%q) error = %v, want ErrInvalidToken", token, err)
		}
	}
}
