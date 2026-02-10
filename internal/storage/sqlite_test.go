package storage

import (
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestLogSearch_BasicAndDailyCount(t *testing.T) {
	s := newTestStore(t)

	// First search
	cnt, err := s.LogSearch("example.com", "user1", true)
	if err != nil {
		t.Fatal(err)
	}
	if cnt != 1 {
		t.Fatalf("want daily count 1, got %d", cnt)
	}

	// Second search same user
	cnt, err = s.LogSearch("example.org", "user1", false)
	if err != nil {
		t.Fatal(err)
	}
	if cnt != 2 {
		t.Fatalf("want daily count 2, got %d", cnt)
	}

	// Different user
	cnt, err = s.LogSearch("example.com", "user2", true)
	if err != nil {
		t.Fatal(err)
	}
	if cnt != 1 {
		t.Fatalf("want daily count 1 for user2, got %d", cnt)
	}
}

func TestGetDailyCheckCount(t *testing.T) {
	s := newTestStore(t)

	cnt, err := s.GetDailyCheckCount("nobody")
	if err != nil {
		t.Fatal(err)
	}
	if cnt != 0 {
		t.Fatalf("want 0, got %d", cnt)
	}

	s.LogSearch("a.com", "u1", true)
	s.LogSearch("b.com", "u1", false)

	cnt, err = s.GetDailyCheckCount("u1")
	if err != nil {
		t.Fatal(err)
	}
	if cnt != 2 {
		t.Fatalf("want 2, got %d", cnt)
	}
}

func TestGetFirstSearcher(t *testing.T) {
	s := newTestStore(t)

	// No entry
	id, err := s.GetFirstSearcher("test.com")
	if err != nil {
		t.Fatal(err)
	}
	if id != "" {
		t.Fatalf("want empty, got %q", id)
	}

	// Anon search should be ignored
	s.LogSearch("test.com", "anon", true)
	id, err = s.GetFirstSearcher("test.com")
	if err != nil {
		t.Fatal(err)
	}
	if id != "" {
		t.Fatalf("want empty for anon, got %q", id)
	}

	// Authenticated search
	s.LogSearch("test.com", "cus_abc", true)
	s.LogSearch("test.com", "cus_xyz", true)

	id, err = s.GetFirstSearcher("test.com")
	if err != nil {
		t.Fatal(err)
	}
	if id != "cus_abc" {
		t.Fatalf("want cus_abc, got %q", id)
	}
}

func TestRecordReferralCredit_NoDuplicate(t *testing.T) {
	s := newTestStore(t)

	ok, err := s.RecordReferralCredit("test.com", "searcher1", "buyer1", 100)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected first insert to succeed")
	}

	// Duplicate should be ignored
	ok, err = s.RecordReferralCredit("test.com", "searcher1", "buyer2", 200)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected duplicate to be ignored")
	}
}

func TestGetDiscoveryFeed_Empty(t *testing.T) {
	s := newTestStore(t)

	entries, err := s.GetDiscoveryFeed(50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("want 0 entries, got %d", len(entries))
	}
}

func TestCleanupOldData(t *testing.T) {
	s := newTestStore(t)

	// Insert some data
	s.LogSearch("a.com", "u1", true)
	if err := s.CleanupOldData(); err != nil {
		t.Fatal(err)
	}
	// Recent data should still be there
	cnt, _ := s.GetDailyCheckCount("u1")
	if cnt != 1 {
		t.Fatalf("recent data should survive cleanup, got count %d", cnt)
	}
}

func TestGetDiscoveryFeed_LimitCap(t *testing.T) {
	s := newTestStore(t)

	// Requesting > 200 should be capped
	entries, err := s.GetDiscoveryFeed(500, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Just verify it doesn't error (empty feed is fine)
	if entries == nil {
		t.Fatal("should return non-nil slice")
	}
}
