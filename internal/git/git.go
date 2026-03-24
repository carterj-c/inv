package git

import (
	"os"
	"os/exec"
	"path/filepath"
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
	return run(dir, "commit", "-m", message, "--allow-empty=false")
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
	// Create a private repo
	if err := run(dir, "remote", "get-url", "origin"); err == nil {
		return nil // already has a remote
	}

	cmd := exec.Command("gh", "repo", "create", "invoice-config", "--private", "--source=.", "--push")
	cmd.Dir = dir
	return cmd.Run()
}

// HasGH returns true if the gh CLI is available.
func HasGH() bool {
	_, err := exec.LookPath("gh")
	return err == nil
}

func run(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}
