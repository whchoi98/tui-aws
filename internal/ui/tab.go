package ui

import "tui-aws/internal/ui/shared"

// Re-export shared tab types so callers that import "ui" can still access them.
type (
	TabModel    = shared.TabModel
	TabID       = shared.TabID
	SharedState = shared.SharedState
	CachedData  = shared.CachedData
)

// Re-export constants
const (
	TabEC2    = shared.TabEC2
	TabVPC    = shared.TabVPC
	TabSubnet = shared.TabSubnet
	TabRoutes = shared.TabRoutes
	TabSG     = shared.TabSG
	TabCheck  = shared.TabCheck
)

// Re-export NavigateToTab
type NavigateToTab = shared.NavigateToTab

// Re-export functions
var (
	AllTabs  = shared.AllTabs
	CacheKey = shared.CacheKey
)
