package pdfsync

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSyncDirsCopiesMissingFilesBothWays(t *testing.T) {
	exportDir := t.TempDir()
	archiveDir := t.TempDir()

	exportFile := filepath.Join(exportDir, "2026-001_client_2026-04-09.pdf")
	archiveFile := filepath.Join(archiveDir, "2026-002_client_2026-04-10.pdf")

	writeTestFile(t, exportFile, "export-only", time.Now().Add(-2*time.Hour))
	writeTestFile(t, archiveFile, "archive-only", time.Now().Add(-1*time.Hour))

	result, err := SyncDirs(exportDir, archiveDir)
	if err != nil {
		t.Fatalf("SyncDirs returned error: %v", err)
	}

	if result.ExportUpdated != 1 {
		t.Fatalf("expected ExportUpdated=1, got %d", result.ExportUpdated)
	}
	if result.ArchiveUpdated != 1 {
		t.Fatalf("expected ArchiveUpdated=1, got %d", result.ArchiveUpdated)
	}

	assertFileContents(t, filepath.Join(exportDir, filepath.Base(archiveFile)), "archive-only")
	assertFileContents(t, filepath.Join(archiveDir, filepath.Base(exportFile)), "export-only")
}

func TestSyncDirsUsesNewerFileWhenBothExist(t *testing.T) {
	exportDir := t.TempDir()
	archiveDir := t.TempDir()

	name := "2026-003_client_2026-04-11.pdf"
	exportFile := filepath.Join(exportDir, name)
	archiveFile := filepath.Join(archiveDir, name)

	writeTestFile(t, exportFile, "new-export", time.Now())
	writeTestFile(t, archiveFile, "old-archive", time.Now().Add(-1*time.Hour))

	result, err := SyncDirs(exportDir, archiveDir)
	if err != nil {
		t.Fatalf("SyncDirs returned error: %v", err)
	}

	if result.ExportUpdated != 0 {
		t.Fatalf("expected ExportUpdated=0, got %d", result.ExportUpdated)
	}
	if result.ArchiveUpdated != 1 {
		t.Fatalf("expected ArchiveUpdated=1, got %d", result.ArchiveUpdated)
	}

	assertFileContents(t, archiveFile, "new-export")
}

func writeTestFile(t *testing.T, path, content string, ts time.Time) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
	if err := touch(path, ts); err != nil {
		t.Fatalf("touch(%s): %v", path, err)
	}
}

func assertFileContents(t *testing.T, path, want string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	if string(data) != want {
		t.Fatalf("unexpected contents for %s: got %q want %q", path, string(data), want)
	}
}
