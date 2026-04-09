// Package plugins — installer.go provides Install/Uninstall for git-based plugin management.
package plugins

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Install clones a git repository into the plugins directory.
// url: git URL (e.g. "https://github.com/DojoGenesis/plugins.git" or "github.com/DojoGenesis/plugins")
// destDir: plugins root directory (e.g. ~/.dojo/plugins)
// Returns the path to the installed plugin directory.
func Install(gitURL, destDir string) (string, error) {
	// 1. Normalize URL: if no scheme, prepend "https://".
	normalized := normalizeURL(gitURL)

	// 2. Extract repo name from URL for the directory name.
	name := repoName(normalized)
	if name == "" {
		return "", fmt.Errorf("cannot extract repository name from URL %q", gitURL)
	}

	dest := filepath.Join(destDir, name)

	// 3. Check if dest already exists.
	if _, err := os.Stat(dest); err == nil {
		return "", fmt.Errorf("plugin already installed at %s", dest)
	}

	// 4. Ensure destDir exists.
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create plugins directory: %w", err)
	}

	// 5. Run: git clone --depth 1 <url> <dest>.
	cmd := exec.Command("git", "clone", "--depth", "1", normalized, dest)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git clone failed: %s\n%s", err, strings.TrimSpace(string(out)))
	}

	// 6. Validate: check that plugin.json exists.
	if !hasPluginJSON(dest) {
		// Clean up the cloned directory.
		_ = os.RemoveAll(dest)
		return "", fmt.Errorf("cloned repository does not contain plugin.json — removed %s", dest)
	}

	return dest, nil
}

// Uninstall removes a plugin directory.
func Uninstall(pluginName, pluginsRoot string) error {
	dest := filepath.Join(pluginsRoot, pluginName)

	// 1. Verify the directory exists.
	info, err := os.Stat(dest)
	if err != nil {
		return fmt.Errorf("plugin %q not found at %s", pluginName, dest)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dest)
	}

	// 2. Safety check: must contain plugin.json.
	if !hasPluginJSON(dest) {
		return fmt.Errorf("%s does not appear to be a plugin (no plugin.json found)", dest)
	}

	// 3. Remove the directory.
	if err := os.RemoveAll(dest); err != nil {
		return fmt.Errorf("failed to remove plugin: %w", err)
	}
	return nil
}

// repoName extracts the directory name from a git URL.
//
//	"https://github.com/DojoGenesis/plugins.git" -> "plugins"
//	"github.com/foo/bar" -> "bar"
func repoName(gitURL string) string {
	// Strip trailing slashes.
	gitURL = strings.TrimRight(gitURL, "/")

	// Parse as URL to get path; fall back to splitting on "/" if that fails.
	var pathStr string
	if u, err := url.Parse(gitURL); err == nil && u.Path != "" {
		pathStr = u.Path
	} else {
		pathStr = gitURL
	}

	// Take the last path segment.
	name := filepath.Base(pathStr)

	// Strip .git suffix.
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

// hasPluginJSON checks whether a directory contains a plugin.json at
// either the root level or inside .claude-plugin/.
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
