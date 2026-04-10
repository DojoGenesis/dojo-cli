// Package plugins — installer.go provides Install/Uninstall for git-based plugin management.
package plugins

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// InstallResult holds the path and name of one installed plugin.
type InstallResult struct {
	Name string
	Path string
}

// Install clones a git URL and installs one or more plugins into destDir.
// Three cases are handled in order:
//  1. GitHub subdirectory URL (https://github.com/{o}/{r}/tree/{branch}/{path}) —
//     sparse-checkout of the subpath only.
//  2. Monorepo — full clone, no root plugin.json found, scan subdirs up to
//     depth 2 for plugin.json and extract each as its own plugin.
//  3. Root plugin — existing behaviour: clone root, validate plugin.json.
func Install(gitURL, destDir string) ([]InstallResult, error) {
	normalized := normalizeURL(gitURL)

	// Case 1: GitHub subdirectory URL.
	if cloneURL, subpath, branch, ok := parseGitHubSubdirURL(normalized); ok {
		return installSparse(cloneURL, subpath, branch, destDir)
	}

	// Cases 2 & 3: full clone then inspect.
	return installFull(normalized, destDir)
}

// InstallConfirmed is the interactive entry point for CLI use.
// It prints a security warning to stderr listing the URL, then — unless
// noConfirm is true — prompts the user for explicit y/n confirmation before
// proceeding. Pass noConfirm=true to skip the prompt (e.g. --yes flag).
func InstallConfirmed(gitURL, destDir string, noConfirm bool) ([]InstallResult, error) {
	fmt.Fprintf(os.Stderr, "\n  WARNING: Installing plugin from external source. Verify trust before proceeding.\n")
	fmt.Fprintf(os.Stderr, "  URL: %s\n\n", gitURL)

	if !noConfirm {
		fmt.Fprint(os.Stderr, "  Continue? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			return nil, fmt.Errorf("plugin install cancelled by user")
		}
	}

	return Install(gitURL, destDir)
}

// Uninstall removes a plugin directory.
func Uninstall(pluginName, pluginsRoot string) error {
	dest := filepath.Join(pluginsRoot, pluginName)

	info, err := os.Stat(dest)
	if err != nil {
		return fmt.Errorf("plugin %q not found at %s", pluginName, dest)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dest)
	}

	if !hasPluginJSON(dest) {
		return fmt.Errorf("%s does not appear to be a plugin (no plugin.json found)", dest)
	}

	if err := os.RemoveAll(dest); err != nil {
		return fmt.Errorf("failed to remove plugin: %w", err)
	}
	return nil
}

// installFull clones the repository and handles root-plugin vs monorepo detection.
func installFull(gitURL, destDir string) ([]InstallResult, error) {
	name := repoName(gitURL)
	if name == "" {
		return nil, fmt.Errorf("cannot extract repository name from URL %q", gitURL)
	}

	dest := filepath.Join(destDir, name)
	if _, err := os.Stat(dest); err == nil {
		return nil, fmt.Errorf("plugin already installed at %s", dest)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create plugins directory: %w", err)
	}

	cmd := exec.Command("git", "clone", "--depth", "1", gitURL, dest)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git clone failed: %s\n%s", err, strings.TrimSpace(string(out)))
	}

	// Case 3: root-level plugin.json — single plugin.
	if hasPluginJSON(dest) {
		return []InstallResult{{Name: name, Path: dest}}, nil
	}

	// Case 2: monorepo — scan subdirs for plugin.json.
	results, err := extractMonorepoPlugins(dest, destDir)
	if err != nil {
		_ = os.RemoveAll(dest)
		return nil, err
	}
	if len(results) == 0 {
		_ = os.RemoveAll(dest)
		return nil, fmt.Errorf("cloned repository contains no plugin.json — not a plugin repo (removed %s)", dest)
	}

	// Remove the full clone after extraction (each plugin now lives at its own path).
	_ = os.RemoveAll(dest)
	return results, nil
}

// extractMonorepoPlugins scans cloneDir up to depth 2 for plugin.json files,
// copies each matching subdir into destDir as an independent plugin, and
// returns one InstallResult per plugin found.
func extractMonorepoPlugins(cloneDir, destDir string) ([]InstallResult, error) {
	var results []InstallResult

	// Scan immediate children.
	top, err := os.ReadDir(cloneDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read cloned repo: %w", err)
	}

	for _, e := range top {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		level1 := filepath.Join(cloneDir, e.Name())

		// Depth 1: plugin.json directly inside a child dir.
		if hasPluginJSON(level1) {
			r, err := movePlugin(level1, destDir)
			if err != nil {
				return nil, err
			}
			results = append(results, r)
			continue
		}

		// Depth 2: plugin.json inside grandchild dirs.
		subs, err := os.ReadDir(level1)
		if err != nil {
			continue
		}
		for _, sub := range subs {
			if !sub.IsDir() || strings.HasPrefix(sub.Name(), ".") {
				continue
			}
			level2 := filepath.Join(level1, sub.Name())
			if hasPluginJSON(level2) {
				r, err := movePlugin(level2, destDir)
				if err != nil {
					return nil, err
				}
				results = append(results, r)
			}
		}
	}
	return results, nil
}

// movePlugin copies srcDir into destDir/{dirName} and returns the InstallResult.
// Uses os.CopyFS (Go 1.23+) where available; falls back to a recursive copy.
func movePlugin(srcDir, destDir string) (InstallResult, error) {
	name := filepath.Base(srcDir)
	dst := filepath.Join(destDir, name)

	if _, err := os.Stat(dst); err == nil {
		return InstallResult{}, fmt.Errorf("plugin already installed at %s", dst)
	}

	if err := copyDir(srcDir, dst); err != nil {
		_ = os.RemoveAll(dst)
		return InstallResult{}, fmt.Errorf("failed to copy plugin %q: %w", name, err)
	}
	return InstallResult{Name: name, Path: dst}, nil
}

// copyDir recursively copies src into dst.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}

// installSparse performs a sparse-checkout of a single subdirectory from a git repo.
func installSparse(cloneURL, subpath, branch, destDir string) ([]InstallResult, error) {
	name := filepath.Base(subpath)
	if name == "" || name == "." {
		return nil, fmt.Errorf("cannot determine plugin name from subpath %q", subpath)
	}

	dest := filepath.Join(destDir, name)
	if _, err := os.Stat(dest); err == nil {
		return nil, fmt.Errorf("plugin already installed at %s", dest)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create plugins directory: %w", err)
	}

	// Step 1: init an empty repo.
	if out, err := exec.Command("git", "init", dest).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("git init failed: %s\n%s", err, strings.TrimSpace(string(out)))
	}

	run := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = dest
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git %s: %s\n%s", args[0], err, strings.TrimSpace(string(out)))
		}
		return nil
	}

	// Step 2: configure sparse-checkout.
	if err := run("remote", "add", "origin", cloneURL); err != nil {
		_ = os.RemoveAll(dest)
		return nil, err
	}
	if err := run("sparse-checkout", "init", "--cone"); err != nil {
		_ = os.RemoveAll(dest)
		return nil, fmt.Errorf("sparse-checkout init failed (git 2.25+ required): %w", err)
	}
	if err := run("sparse-checkout", "set", subpath); err != nil {
		_ = os.RemoveAll(dest)
		return nil, err
	}

	// Step 3: fetch only the target branch, depth 1.
	if err := run("fetch", "--depth", "1", "origin", branch); err != nil {
		_ = os.RemoveAll(dest)
		return nil, err
	}
	if err := run("checkout", branch); err != nil {
		// Try FETCH_HEAD as fallback.
		if err2 := run("checkout", "FETCH_HEAD"); err2 != nil {
			_ = os.RemoveAll(dest)
			return nil, fmt.Errorf("checkout failed: %w (also tried FETCH_HEAD: %v)", err, err2)
		}
	}

	// After sparse checkout, the plugin lives at dest/{subpath}.
	// Promote it: copy the contents up to dest, then remove the scaffold.
	pluginSrc := filepath.Join(dest, subpath)
	if _, err := os.Stat(pluginSrc); err != nil {
		_ = os.RemoveAll(dest)
		return nil, fmt.Errorf("sparse checkout did not produce %s", pluginSrc)
	}

	// Temp rename dest, copy subdir back to dest.
	tmp := dest + ".__tmp"
	if err := os.Rename(dest, tmp); err != nil {
		_ = os.RemoveAll(dest)
		return nil, fmt.Errorf("failed to stage sparse checkout: %w", err)
	}
	if err := copyDir(filepath.Join(tmp, subpath), dest); err != nil {
		_ = os.RemoveAll(tmp)
		_ = os.RemoveAll(dest)
		return nil, fmt.Errorf("failed to promote sparse subpath: %w", err)
	}
	_ = os.RemoveAll(tmp)

	if !hasPluginJSON(dest) {
		_ = os.RemoveAll(dest)
		return nil, fmt.Errorf("subdirectory %q does not contain plugin.json", subpath)
	}

	return []InstallResult{{Name: name, Path: dest}}, nil
}

// parseGitHubSubdirURL detects a GitHub web URL pointing to a subdirectory:
//
//	https://github.com/{owner}/{repo}/tree/{branch}/{path...}
//
// Returns (cloneURL, subpath, branch, true) on match.
func parseGitHubSubdirURL(rawURL string) (cloneURL, subpath, branch string, ok bool) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host != "github.com" {
		return "", "", "", false
	}
	// Path segments: ["", owner, repo, "tree", branch, path...]
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 5 || parts[2] != "tree" {
		return "", "", "", false
	}
	cloneURL = fmt.Sprintf("https://github.com/%s/%s", parts[0], parts[1])
	branch = parts[3]
	subpath = strings.Join(parts[4:], "/")
	return cloneURL, subpath, branch, true
}

// repoName extracts a directory name from a git URL.
//
//	"https://github.com/DojoGenesis/plugins.git" -> "plugins"
//	"github.com/foo/bar" -> "bar"
func repoName(gitURL string) string {
	gitURL = strings.TrimRight(gitURL, "/")

	var pathStr string
	if u, err := url.Parse(gitURL); err == nil && u.Path != "" {
		pathStr = u.Path
	} else {
		pathStr = gitURL
	}

	name := filepath.Base(pathStr)
	name = strings.TrimSuffix(name, ".git")

	if name == "" || name == "." || name == "/" {
		return ""
	}
	return name
}

// normalizeURL prepends "https://" if the URL has no scheme.
func normalizeURL(gitURL string) string {
	gitURL = strings.TrimSpace(gitURL)
	if strings.Contains(gitURL, "://") {
		return gitURL
	}
	return "https://" + gitURL
}

// hasPluginJSON checks whether a directory contains plugin.json at
// the root or inside .claude-plugin/.
func hasPluginJSON(dir string) bool {
	candidates := []string{
		filepath.Join(dir, "plugin.json"),
		filepath.Join(dir, ".claude-plugin", "plugin.json"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}
