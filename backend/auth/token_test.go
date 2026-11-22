package auth

import "testing"

func TestIssueVerifyRoundTrip(t *testing.T) {
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	s, persistent, err := NewSigner()
	if err != nil || !persistent {
		t.Fatalf("signer: %v persistent=%v", err, persistent)
	}
	tok, err := s.Issue("imad", "admin")
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.Verify(tok)
	if err != nil {
		t.Fatal(err)
	}
	if c.Username != "imad" || c.Role != "admin" {
		t.Fatalf("bad claims %+v", c)
	}
}

func TestTamperedTokenRejected(t *testing.T) {
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	s, _, _ := NewSigner()
	tok, _ := s.Issue("viewer", "viewer")
	// flip the role claim by re-signing with a different secret
	other := &Signer{secret: []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")}
	forged, _ := other.Issue("viewer", "admin")
	if _, err := s.Verify(forged); err == nil {
		t.Fatal("forged token accepted")
	}
	if _, err := s.Verify(tok + "x"); err == nil {
		t.Fatal("tampered signature accepted")
	}
}

func TestShortSecretRejected(t *testing.T) {
	t.Setenv("JWT_SECRET", "short")
	if _, _, err := NewSigner(); err == nil {
		t.Fatal("short secret accepted")
	}
}
