package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// IsRepo returns true if the directory is a git repository.
func IsRepo(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

// Init initializes a git repository in the given directory.
func Init(dir string) error {
	return run(dir, "init")
}

// CommitAll stages all changes and commits with the given message.
func CommitAll(dir, message string) error {
	if err := run(dir, "add", "-A"); err != nil {
		return err
	}
	return run(dir, "commit", "-m", message)
}

// Pull runs git pull if a remote is configured. Best-effort.
func Pull(dir string) error {
	if !HasRemote(dir) {
		return nil
	}
	return run(dir, "pull", "--rebase", "--quiet")
}

// Push runs git push if a remote is configured. Best-effort.
func Push(dir string) error {
	if !HasRemote(dir) {
		return nil
	}
	return run(dir, "push", "--quiet")
}

// HasRemote returns true if a remote named "origin" is configured.
func HasRemote(dir string) bool {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = dir
	return cmd.Run() == nil
}

// SetupGitHub attempts to create a private GitHub repo and set it as origin.
// Requires the `gh` CLI to be installed and authenticated.
func SetupGitHub(dir string) error {
	if HasRemote(dir) {
		return nil
	}
	if !HasGH() {
		return fmt.Errorf("gh CLI is not installed")
	}
	if !HasGHAuth() {
		return fmt.Errorf("gh CLI is not authenticated")
	}

	cmd := exec.Command(
		"gh",
		"repo",
		"create",
		"invoice-config",
		"--private",
		"--source=.",
		"--remote=origin",
		"--push",
	)
	cmd.Dir = dir
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return err
	}

	// Ensure future plain `git push` calls have an upstream branch configured.
	return run(dir, "push", "-u", "origin", "HEAD", "--quiet")
}

// HasGH returns true if the gh CLI is available.
func HasGH() bool {
	_, err := exec.LookPath("gh")
	return err == nil
}

// HasGHAuth returns true if `gh` is installed and has a valid authenticated account.
func HasGHAuth() bool {
	if !HasGH() {
		return false
	}

	cmd := exec.Command("gh", "auth", "status")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	text := strings.ToLower(string(output))
	return !strings.Contains(text, "failed to log in") &&
		!strings.Contains(text, "not logged into any") &&
		!strings.Contains(text, "invalid")
}

func run(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}
