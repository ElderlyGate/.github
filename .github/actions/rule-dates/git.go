package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Files maps a path, relative to the repository root, to its content.
type Files map[string]string

// git runs git in dir and returns its output.
func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(ee.Stderr)))
		}
		return "", err
	}
	return string(out), nil
}

// matches says whether a path fits one of the patterns (path.Match syntax).
func matches(p string, patterns []string) bool {
	for _, pat := range patterns {
		if ok, _ := path.Match(pat, p); ok {
			return true
		}
	}
	return false
}

// WorkFiles reads the files in the working tree under dir that fit the patterns.
func WorkFiles(dir string, patterns []string) (Files, error) {
	files := Files{}
	for _, pat := range patterns {
		found, err := filepath.Glob(filepath.Join(dir, filepath.FromSlash(pat)))
		if err != nil {
			return nil, err
		}
		for _, f := range found {
			b, err := os.ReadFile(f)
			if err != nil {
				return nil, err
			}
			rel, _ := filepath.Rel(dir, f)
			files[filepath.ToSlash(rel)] = string(b)
		}
	}
	return files, nil
}

// RevFiles reads the files at a revision that fit the patterns.
func RevFiles(dir, rev string, patterns []string) (Files, error) {
	list, err := git(dir, "ls-tree", "-r", "--name-only", rev)
	if err != nil {
		return nil, err
	}
	files := Files{}
	for _, p := range strings.Split(strings.TrimSpace(list), "\n") {
		if p == "" || !matches(p, patterns) {
			continue
		}
		content, err := git(dir, "show", rev+":"+p)
		if err != nil {
			return nil, err
		}
		files[p] = content
	}
	return files, nil
}

// Commit is one commit on the first-parent line, with the day it landed.
type Commit struct {
	SHA  string
	Date string
}

// History lists, oldest first, the first-parent commits up to rev that touched the patterns.
func History(dir, rev string, patterns []string) ([]Commit, error) {
	args := []string{"log", "--first-parent", "--reverse", "--format=%H %cs", rev, "--"}
	for _, p := range patterns {
		args = append(args, ":(glob)"+p)
	}
	out, err := git(dir, args...)
	if err != nil {
		return nil, err
	}
	var commits []Commit
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		if sha, date, ok := strings.Cut(l, " "); ok {
			commits = append(commits, Commit{SHA: sha, Date: date})
		}
	}
	return commits, nil
}

// CommitDay is the day rev was committed.
func CommitDay(dir, rev string) (string, error) {
	out, err := git(dir, "log", "-1", "--format=%cs", rev)
	return strings.TrimSpace(out), err
}

// sortedPaths gives the paths of files in a stable order.
func sortedPaths(files Files) []string {
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}
