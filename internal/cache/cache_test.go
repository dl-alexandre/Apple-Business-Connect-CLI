package cache

import (
	"testing"
	"time"
)

func TestCacheSetGetAndDelete(t *testing.T) {
	c := New(t.TempDir(), time.Hour)
	if err := c.Set("abc123", []byte(`{"value":1}`)); err != nil {
		t.Fatal(err)
	}
	got, ok := c.Get("abc123")
	if !ok || string(got) != `{"value":1}` {
		t.Fatalf("Get() = %q, %t", got, ok)
	}
	if err := c.Delete("abc123"); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.Get("abc123"); ok {
		t.Fatal("Get() returned deleted value")
	}
}

func TestCacheExpiresEntries(t *testing.T) {
	c := New(t.TempDir(), -time.Second)
	if err := c.Set("expired", []byte("old")); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.Get("expired"); ok {
		t.Fatal("Get() returned expired value")
	}
}
