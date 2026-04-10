package pdfsync

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/carter/inv/internal/config"
	"github.com/carter/inv/internal/model"
)

type Result struct {
	ExportUpdated  int
	ArchiveUpdated int
}

type fileEntry struct {
	path string
	info os.FileInfo
}

// Sync keeps the export directory and tracked archive directory aligned.
func Sync(global model.GlobalConfig) (Result, error) {
	return SyncDirs(exportDir(global), config.PDFArchiveDir())
}

// SyncDirs keeps two PDF directories aligned without deleting files.
// If the same file exists in both places and differs, the newer mtime wins.
// If mtimes are equal but contents differ, the export copy wins.
func SyncDirs(exportDir, archiveDir string) (Result, error) {
	var result Result

	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return result, fmt.Errorf("creating export dir: %w", err)
	}
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return result, fmt.Errorf("creating archive dir: %w", err)
	}

	exportFiles, err := listPDFs(exportDir)
	if err != nil {
		return result, err
	}
	archiveFiles, err := listPDFs(archiveDir)
	if err != nil {
		return result, err
	}

	names := make(map[string]struct{}, len(exportFiles)+len(archiveFiles))
	for name := range exportFiles {
		names[name] = struct{}{}
	}
	for name := range archiveFiles {
		names[name] = struct{}{}
	}

	ordered := make([]string, 0, len(names))
	for name := range names {
		ordered = append(ordered, name)
	}
	sort.Strings(ordered)

	for _, name := range ordered {
		exportFile, hasExport := exportFiles[name]
		archiveFile, hasArchive := archiveFiles[name]

		switch {
		case hasExport && !hasArchive:
			if err := copyFile(exportFile.path, filepath.Join(archiveDir, name), exportFile.info); err != nil {
				return result, err
			}
			result.ArchiveUpdated++

		case !hasExport && hasArchive:
			if err := copyFile(archiveFile.path, filepath.Join(exportDir, name), archiveFile.info); err != nil {
				return result, err
			}
			result.ExportUpdated++

		case hasExport && hasArchive:
			same, err := sameContents(exportFile.path, archiveFile.path)
			if err != nil {
				return result, err
			}
			if same {
				continue
			}

			switch newerSource(exportFile, archiveFile) {
			case "export":
				if err := copyFile(exportFile.path, archiveFile.path, exportFile.info); err != nil {
					return result, err
				}
				result.ArchiveUpdated++
			case "archive":
				if err := copyFile(archiveFile.path, exportFile.path, archiveFile.info); err != nil {
					return result, err
				}
				result.ExportUpdated++
			}
		}
	}

	return result, nil
}

func exportDir(global model.GlobalConfig) string {
	dir := global.Settings.ExportDir
	if dir == "" {
		dir = "~/invoices"
	}
	return config.ExpandPath(dir)
}

func listPDFs(dir string) (map[string]fileEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}

	files := make(map[string]fileEntry)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".pdf") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("reading file info for %s: %w", entry.Name(), err)
		}
		files[entry.Name()] = fileEntry{
			path: filepath.Join(dir, entry.Name()),
			info: info,
		}
	}
	return files, nil
}

func newerSource(exportFile, archiveFile fileEntry) string {
	exportMod := exportFile.info.ModTime()
	archiveMod := archiveFile.info.ModTime()

	if exportMod.After(archiveMod) {
		return "export"
	}
	if archiveMod.After(exportMod) {
		return "archive"
	}
	return "export"
}

func sameContents(a, b string) (bool, error) {
	infoA, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	infoB, err := os.Stat(b)
	if err != nil {
		return false, err
	}
	if infoA.Size() != infoB.Size() {
		return false, nil
	}

	fa, err := os.Open(a)
	if err != nil {
		return false, err
	}
	defer fa.Close()

	fb, err := os.Open(b)
	if err != nil {
		return false, err
	}
	defer fb.Close()

	bufA := make([]byte, 32*1024)
	bufB := make([]byte, 32*1024)

	for {
		nA, errA := fa.Read(bufA)
		nB, errB := fb.Read(bufB)

		if errA != nil && errA != io.EOF {
			return false, errA
		}
		if errB != nil && errB != io.EOF {
			return false, errB
		}
		if nA != nB || string(bufA[:nA]) != string(bufB[:nB]) {
			return false, nil
		}
		if errA == io.EOF && errB == io.EOF {
			return true, nil
		}
	}
}

func copyFile(src, dst string, info os.FileInfo) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return fmt.Errorf("opening %s: %w", dst, err)
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return fmt.Errorf("copying %s to %s: %w", src, dst, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", dst, err)
	}
	mod := info.ModTime()
	if err := os.Chtimes(dst, mod, mod); err != nil {
		return fmt.Errorf("updating times for %s: %w", dst, err)
	}
	return nil
}

func touch(path string, ts time.Time) error {
	return os.Chtimes(path, ts, ts)
}
