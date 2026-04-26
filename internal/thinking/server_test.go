package thinking

import "testing"

func TestNewServerStartsEmpty(t *testing.T) {
	s := NewServer()
	if s == nil {
		t.Fatal("NewServer returned nil")
	}
	if got := s.HistoryLength(); got != 0 {
		t.Errorf("HistoryLength = %d, want 0", got)
	}
	if got := s.SessionConfidence(); got != 0 {
		t.Errorf("SessionConfidence = %v, want 0", got)
	}
}
