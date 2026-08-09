package main

import (
	"fmt"
	"strconv"
	"strings"
)

type Version struct {
	Major int
	Minor int
	Patch int
}

func ParseVersion(v string) (Version, error) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("invalid semver: %s", v)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, fmt.Errorf("invalid major version in %s: %w", v, err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{}, fmt.Errorf("invalid minor version in %s: %w", v, err)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return Version{}, fmt.Errorf("invalid patch version in %s: %w", v, err)
	}
	return Version{Major: major, Minor: minor, Patch: patch}, nil
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func (v Version) Compare(other Version) int {
	switch {
	case v.Major != other.Major:
		if v.Major > other.Major {
			return 1
		}
		return -1
	case v.Minor != other.Minor:
		if v.Minor > other.Minor {
			return 1
		}
		return -1
	case v.Patch != other.Patch:
		if v.Patch > other.Patch {
			return 1
		}
		return -1
	}
	return 0
}

func (v Version) AtLeast(other Version) bool {
	return v.Compare(other) >= 0
}

type ComponentRequirement struct {
	Component      string
	MinVersion     Version
	Description    string
}

var componentRequirements = []ComponentRequirement{
	{Component: "core", MinVersion: Version{1, 0, 0}, Description: "Core system base version"},
	{Component: "hub", MinVersion: Version{1, 2, 0}, Description: "Hub requires core 1.0.0+"},
	{Component: "agent", MinVersion: Version{1, 1, 0}, Description: "Agent requires core 1.0.0+"},
	{Component: "database", MinVersion: Version{0, 0, 0}, Description: "Database migrations available from any version"},
	{Component: "plugins", MinVersion: Version{1, 0, 0}, Description: "Plugin system requires core 1.0.0+"},
}

var componentDeps = map[string][]string{
	"core":    {"hub", "agent", "plugins"},
	"hub":     {"core"},
	"agent":   {"core"},
	"plugins": {"core"},
	"database": {},
}

func CheckVersionCompat(from, to string, component string) error {
	fromVer, err := ParseVersion(from)
	if err != nil {
		return fmt.Errorf("parse from version %s: %w", from, err)
	}
	toVer, err := ParseVersion(to)
	if err != nil {
		return fmt.Errorf("parse to version %s: %w", to, err)
	}

	if toVer.Compare(fromVer) < 0 {
		return fmt.Errorf("downgrade not supported: %s -> %s", from, to)
	}
	if toVer.Compare(fromVer) == 0 {
		return fmt.Errorf("version unchanged: %s", from)
	}

	req := RequiredVersion(component)
	if !toVer.AtLeast(req) {
		return fmt.Errorf("version %s does not meet minimum requirement %s for component %s",
			to, req.String(), component)
	}

	if toVer.Major > fromVer.Major+1 {
		return fmt.Errorf("cannot skip major versions: upgrade path %s -> %s requires intermediate upgrades",
			from, to)
	}

	return nil
}

func RequiredVersion(component string) Version {
	for _, r := range componentRequirements {
		if r.Component == component {
			return r.MinVersion
		}
	}
	return Version{0, 0, 0}
}

func BreakingChanges(from, to string) []string {
	fromVer, err := ParseVersion(from)
	if err != nil {
		return []string{fmt.Sprintf("cannot parse from version: %s", from)}
	}
	toVer, err := ParseVersion(to)
	if err != nil {
		return []string{fmt.Sprintf("cannot parse to version: %s", to)}
	}

	var changes []string
	if fromVer.Major != toVer.Major {
		changes = append(changes, "Major version change: API breaking changes may be present")
		changes = append(changes, "Database migrations may be required")
		changes = append(changes, "Configuration file format may have changed")
	}
	if fromVer.Minor != toVer.Minor {
		changes = append(changes, "Minor version change: new features and deprecations")
	}
	if fromVer.Patch != toVer.Patch {
		changes = append(changes, "Patch version change: bug fixes and security patches")
	}
	return changes
}

func ComponentDependencies(component string) []string {
	return componentDeps[component]
}

func CheckPluginCompatibility(version string) error {
	v, err := ParseVersion(version)
	if err != nil {
		return fmt.Errorf("parse version for plugin check: %w", err)
	}
	minPlugin := Version{1, 0, 0}
	if !v.AtLeast(minPlugin) {
		return fmt.Errorf("plugins require core >= %s, current: %s", minPlugin.String(), version)
	}
	return nil
}
