package aws

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func DefaultCredentialsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aws", "credentials")
}

func DefaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aws", "config")
}

// InstanceRoleProfile is a sentinel value representing the EC2 instance's
// IAM role (no explicit --profile flag needed).
const InstanceRoleProfile = "(instance role)"

func ParseProfiles(credentialsPath, configPath string) []string {
	seen := map[string]bool{}
	for _, name := range parseSections(credentialsPath, false) {
		seen[name] = true
	}
	for _, name := range parseSections(configPath, true) {
		seen[name] = true
	}
	named := make([]string, 0, len(seen))
	for name := range seen {
		named = append(named, name)
	}
	sort.Strings(named)
	// Prepend instance role option so it appears first in the selector
	return append([]string{InstanceRoleProfile}, named...)
}

func parseSections(path string, isConfig bool) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var names []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "[") || !strings.HasSuffix(line, "]") {
			continue
		}
		name := line[1 : len(line)-1]
		if isConfig {
			if strings.HasPrefix(name, "profile ") {
				name = strings.TrimPrefix(name, "profile ")
			}
		}
		name = strings.TrimSpace(name)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}
