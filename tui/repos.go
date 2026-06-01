package tui

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func ScanRepos(cwd string) ([]Repo, error) {
	abs, _ := filepath.Abs(cwd)
	var repos []Repo

	filepath.WalkDir(abs, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		name := d.Name()
		if name == ".git" || name == ".yaks" || name == ".workspaces" || name == "node_modules" || name == ".jj" {
			return filepath.SkipDir
		}

		gitDir := filepath.Join(path, ".git")
		yaksDir := filepath.Join(path, ".yaks")
		gitInfo, gitErr := os.Stat(gitDir)
		yaksInfo, yaksErr := os.Stat(yaksDir)

		if gitErr == nil && gitInfo.IsDir() && yaksErr == nil && yaksInfo.IsDir() {
			remote := detectRemote(path)
			yaks, _ := LoadYaks(yaksDir)
			h := sha1.Sum([]byte(path))
			wfID := "yy-orch-" + hex.EncodeToString(h[:4])

			repos = append(repos, Repo{
				Name:    remoteName(path, remote),
				Root:    path,
				Remote:  remote,
				Yaks:    yaks,
				YaksDir: yaksDir,
				WFID:    wfID,
			})
		}
		return nil
	})

	return repos, nil
}

func detectRemote(repoRoot string) string {
	out, err := runGit(repoRoot, "remote", "get-url", "origin")
	if err != nil {
		return ""
	}
	url := strings.TrimSpace(out)
	url = strings.TrimPrefix(url, "https://github.com/")
	url = strings.TrimPrefix(url, "git@github.com:")
	url = strings.TrimSuffix(url, ".git")
	return url
}

func remoteName(repoRoot, remote string) string {
	if remote != "" {
		return remote
	}
	return filepath.Base(repoRoot)
}

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return string(out), err
}
