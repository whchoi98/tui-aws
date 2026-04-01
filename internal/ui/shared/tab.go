package shared

import (
	tea "charm.land/bubbletea/v2"
	"tui-aws/internal/config"
	"tui-aws/internal/store"
)

// TabModel is the interface that all tab implementations must satisfy.
type TabModel interface {
	Init(shared *SharedState) tea.Cmd
	Update(msg tea.Msg, shared *SharedState) (TabModel, tea.Cmd)
	View(shared *SharedState) string
	ShortHelp() string
}

// TabID identifies a tab in the tab bar.
type TabID int

const (
	TabEC2 TabID = iota
	TabASG
	TabEBS
	TabVPC
	TabSubnet
	TabRoutes
	TabSG
	TabVPCEndpoint
	TabTGW
	TabELB
	TabCloudFront
	TabWAF
	TabACM
	TabR53
	TabRDS
	TabS3
	TabECS
	TabEKS
	TabLambda
	TabCloudWatch
	TabIAM
	TabCheck
)

// Label returns the display label for the tab bar.
func (t TabID) Label() string {
	switch t {
	case TabEC2:
		return "EC2"
	case TabASG:
		return "ASG"
	case TabEBS:
		return "EBS"
	case TabVPC:
		return "VPC"
	case TabSubnet:
		return "Subnet"
	case TabRoutes:
		return "Routes"
	case TabSG:
		return "SG"
	case TabVPCEndpoint:
		return "VPCE"
	case TabTGW:
		return "TGW"
	case TabELB:
		return "ELB"
	case TabCloudFront:
		return "CF"
	case TabWAF:
		return "WAF"
	case TabACM:
		return "ACM"
	case TabR53:
		return "R53"
	case TabRDS:
		return "RDS"
	case TabS3:
		return "S3"
	case TabECS:
		return "ECS"
	case TabEKS:
		return "EKS"
	case TabLambda:
		return "Lambda"
	case TabCloudWatch:
		return "CW"
	case TabIAM:
		return "IAM"
	case TabCheck:
		return "Check"
	default:
		return "?"
	}
}

// AllTabs returns all tab IDs in display order.
func AllTabs() []TabID {
	return []TabID{
		TabEC2, TabASG, TabEBS,
		TabVPC, TabSubnet, TabRoutes, TabSG, TabVPCEndpoint, TabTGW,
		TabELB, TabCloudFront, TabWAF, TabACM,
		TabR53, TabRDS, TabS3,
		TabECS, TabEKS, TabLambda,
		TabCloudWatch, TabIAM,
		TabCheck,
	}
}

// NavigateToTab is a message that requests switching to a specific tab.
type NavigateToTab struct {
	Tab TabID
}

// SharedState holds state shared across all tabs.
type SharedState struct {
	Profile   string
	Region    string
	Profiles  []string
	Cfg       config.Config
	Favorites *store.Favorites
	History   *store.History
	Width     int
	Height    int
	Cache     map[string]CachedData
}

// CachedData holds cached data for a tab, keyed by profile+region.
type CachedData struct {
	Data interface{}
}

// CacheKey returns a cache key for the given profile and region.
func CacheKey(profile, region string) string {
	return profile + "::" + region
}

// GetCache retrieves cached data for the current profile and region.
func (s *SharedState) GetCache(prefix string) (CachedData, bool) {
	key := prefix + "::" + CacheKey(s.Profile, s.Region)
	d, ok := s.Cache[key]
	return d, ok
}

// SetCache stores cached data for the current profile and region.
func (s *SharedState) SetCache(prefix string, data CachedData) {
	if s.Cache == nil {
		s.Cache = make(map[string]CachedData)
	}
	key := prefix + "::" + CacheKey(s.Profile, s.Region)
	s.Cache[key] = data
}

// ClearCache removes all cached data.
func (s *SharedState) ClearCache() {
	s.Cache = make(map[string]CachedData)
}
