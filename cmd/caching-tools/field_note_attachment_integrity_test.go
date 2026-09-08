package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAttachmentIntegrityStatuses(t *testing.T) {
	store := newFieldNoteAttachmentStore(t.TempDir())
	ok, err := store.create("note-ok", "ok.png", pngFixture())
	if err != nil { t.Fatal(err) }
	items, err := store.verifyNote("note-ok")
	if err != nil { t.Fatal(err) }
	if len(items) != 1 || items[0].Status != "ok" || items[0].ExpectedSHA256 == "" || items[0].ExpectedSHA256 != items[0].ActualSHA256 { t.Fatalf("ok items=%#v", items) }

	if err := os.WriteFile(filepath.Join(store.noteDir("note-ok"), ok.ID+".bin"), []byte("changed bytes"), 0o600); err != nil { t.Fatal(err) }
	items, err = store.verifyNote("note-ok")
	if err != nil { t.Fatal(err) }
	if len(items) != 1 || items[0].Status != "mismatch" || items[0].ExpectedSHA256 == items[0].ActualSHA256 { t.Fatalf("mismatch items=%#v", items) }

	missing, err := store.create("note-missing", "missing.png", pngFixture())
	if err != nil { t.Fatal(err) }
	if err := os.Remove(filepath.Join(store.noteDir("note-missing"), missing.ID+".bin")); err != nil { t.Fatal(err) }
	items, err = store.verifyNote("note-missing")
	if err != nil { t.Fatal(err) }
	if len(items) != 1 || items[0].Status != "missing" { t.Fatalf("missing items=%#v", items) }
}

func TestAttachmentIntegrityReportCounts(t *testing.T) {
	items := []attachmentIntegrityItem{{Status:"ok"},{Status:"ok"},{Status:"mismatch"},{Status:"missing"},{Status:"unrecorded"}}
	report := buildAttachmentIntegrityReport(items)
	if report.Total != 5 || report.OK != 2 || report.Mismatch != 1 || report.Missing != 1 || report.Unrecorded != 1 { t.Fatalf("report=%#v", report) }
	if report.GeneratedAt == "" { t.Fatal("missing generated_at") }
}
