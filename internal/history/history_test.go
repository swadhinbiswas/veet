package history

import (
	"testing"
	"time"

	"github.com/spf13/afero"
)

func TestAppendAndRead(t *testing.T) {
	fs := afero.NewMemMapFs()
	l := New(fs, "/log/history.log")
	now := time.Now()
	if err := l.Append(Entry{Time: now, App: "foo", Source: "apt", Status: "ok", FreedKB: 1234, Files: 5}); err != nil {
		t.Fatal(err)
	}
	if err := l.Append(Entry{Time: now.Add(-time.Minute), App: "bar", Source: "flatpak", Status: "failed", FreedKB: 0, Files: 0, Detail: "oops"}); err != nil {
		t.Fatal(err)
	}
	entries, err := l.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("read %d entries, want 2", len(entries))
	}
	// most recent first
	if entries[0].App != "foo" || entries[1].App != "bar" {
		t.Fatalf("ordering wrong: %+v", entries)
	}
	if entries[0].FreedKB != 1234 || entries[0].Files != 5 {
		t.Fatalf("entry fields wrong: %+v", entries[0])
	}
	if entries[1].Detail != "oops" {
		t.Fatalf("detail not round-tripped: %+v", entries[1])
	}
}

func TestReadMissingFile(t *testing.T) {
	l := New(afero.NewMemMapFs(), "/nope/history.log")
	entries, err := l.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no entries, got %d", len(entries))
	}
}
