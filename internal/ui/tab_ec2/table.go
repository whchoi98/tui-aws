package tab_ec2

import (
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
	"tui-aws/internal/aws"
	"tui-aws/internal/store"
	"tui-aws/internal/ui/shared"
)

// DefaultColumns returns the full set of table columns.
func DefaultColumns() []shared.Column {
	return []shared.Column{
		{Key: "fav", Title: " ", Width: 2},
		{Key: "state_icon", Title: " ", Width: 2},
		{Key: "name", Title: "Name", Width: 20},
		{Key: "id", Title: "Instance ID", Width: 21},
		{Key: "state", Title: "State", Width: 10},
		{Key: "private_ip", Title: "Private IP", Width: 15},
		{Key: "type", Title: "Type", Width: 12},
		{Key: "az", Title: "AZ", Width: 5},
		{Key: "platform", Title: "Platform", Width: 10},
		{Key: "public_ip", Title: "Public IP", Width: 15},
		{Key: "launch_time", Title: "Launch Time", Width: 18},
		{Key: "sg", Title: "Security Groups", Width: 20},
		{Key: "key_pair", Title: "Key Pair", Width: 15},
		{Key: "iam_role", Title: "IAM Role", Width: 20},
	}
}

// CompactColumns returns a minimal set of columns for narrow terminals.
func CompactColumns() []shared.Column {
	return []shared.Column{
		{Key: "fav", Title: " ", Width: 2},
		{Key: "state_icon", Title: " ", Width: 2},
		{Key: "name", Title: "Name", Width: 20},
		{Key: "state", Title: "State", Width: 10},
		{Key: "private_ip", Title: "Private IP", Width: 15},
	}
}

// ColumnsForWidth returns the appropriate column set for the given terminal width.
func ColumnsForWidth(width int) []shared.Column {
	if width < 80 {
		return CompactColumns()
	}
	return DefaultColumns()
}

// RenderTable renders the EC2 instance table with header, rows, and scrolling.
func RenderTable(instances []aws.Instance, columns []shared.Column, cursor int, favs *store.Favorites, hist *store.History, profile, region string, width, height int) string {
	var b strings.Builder

	// Header
	header := shared.RenderRow(columns, func(col shared.Column) string {
		return col.Title
	}, nil)
	b.WriteString(shared.TableHeaderStyle.Width(width).Render(header))
	b.WriteString("\n")

	// Available rows: total height minus statusbar(1) + helpbar(1) + header(1) + possible search(1)
	maxRows := height - 4
	if maxRows < 1 {
		maxRows = 1
	}

	// Calculate scroll offset
	offset := 0
	if cursor >= maxRows {
		offset = cursor - maxRows + 1
	}

	for i := offset; i < len(instances) && i < offset+maxRows; i++ {
		inst := instances[i]
		row := shared.RenderRow(columns, func(col shared.Column) string {
			return cellValue(col.Key, inst, favs, hist, profile, region)
		}, func(col shared.Column) lipgloss.Style {
			return cellStyle(col.Key, inst, favs, hist, profile, region)
		})

		if i == cursor {
			row = shared.TableSelectedStyle.Width(width).Render(row)
		}
		b.WriteString(row)
		if i < offset+maxRows-1 && i < len(instances)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// cellValue returns raw text without ANSI styling.
func cellValue(key string, inst aws.Instance, favs *store.Favorites, hist *store.History, profile, region string) string {
	switch key {
	case "fav":
		if favs != nil && favs.IsFavorite(inst.InstanceID, profile, region) {
			return "★"
		}
		if hist != nil && hist.IsRecent(inst.InstanceID, profile, region) {
			return "⏱"
		}
		return " "
	case "state_icon":
		return inst.StateIcon()
	case "name":
		return inst.DisplayName()
	case "id":
		return inst.InstanceID
	case "state":
		return inst.State
	case "private_ip":
		return inst.PrivateIP
	case "public_ip":
		return inst.PublicIP
	case "type":
		return inst.InstanceType
	case "az":
		return inst.ShortAZ()
	case "platform":
		return inst.Platform
	case "launch_time":
		return inst.LaunchTimeFormatted()
	case "sg":
		return strings.Join(inst.SecurityGroups, ",")
	case "key_pair":
		return inst.KeyPair
	case "iam_role":
		return inst.IAMRole
	default:
		return ""
	}
}

// cellStyle returns the lipgloss style for a given column and instance.
func cellStyle(key string, inst aws.Instance, favs *store.Favorites, hist *store.History, profile, region string) lipgloss.Style {
	switch key {
	case "fav":
		if favs != nil && favs.IsFavorite(inst.InstanceID, profile, region) {
			return shared.FavoriteStyle
		}
		if hist != nil && hist.IsRecent(inst.InstanceID, profile, region) {
			return shared.RecentStyle
		}
	case "state_icon", "state":
		return shared.StateStyle(inst.State)
	}
	return lipgloss.Style{}
}

// SortInstances sorts instances by favorites, then recent history, then user-selected sort.
func SortInstances(instances []aws.Instance, favs *store.Favorites, hist *store.History, profile, region, sortBy, sortOrder string) []aws.Instance {
	sorted := make([]aws.Instance, len(instances))
	copy(sorted, instances)

	sort.SliceStable(sorted, func(i, j int) bool {
		// Priority 1: Favorites first
		iFav := favs != nil && favs.IsFavorite(sorted[i].InstanceID, profile, region)
		jFav := favs != nil && favs.IsFavorite(sorted[j].InstanceID, profile, region)
		if iFav != jFav {
			return iFav
		}

		// Priority 2: Recent history
		iRecent := hist != nil && hist.IsRecent(sorted[i].InstanceID, profile, region)
		jRecent := hist != nil && hist.IsRecent(sorted[j].InstanceID, profile, region)
		if iRecent != jRecent {
			return iRecent
		}

		// Priority 3: User-selected sort
		var less bool
		switch sortBy {
		case "id":
			less = sorted[i].InstanceID < sorted[j].InstanceID
		case "state":
			less = sorted[i].State < sorted[j].State
		case "type":
			less = sorted[i].InstanceType < sorted[j].InstanceType
		case "az":
			less = sorted[i].AvailabilityZone < sorted[j].AvailabilityZone
		default: // "name"
			less = sorted[i].DisplayName() < sorted[j].DisplayName()
		}
		if sortOrder == "desc" {
			return !less
		}
		return less
	})

	return sorted
}

// FilterBySearch filters instances by name, ID, or private IP.
func FilterBySearch(instances []aws.Instance, query string) []aws.Instance {
	if query == "" {
		return instances
	}
	q := strings.ToLower(query)
	var result []aws.Instance
	for _, inst := range instances {
		if strings.Contains(strings.ToLower(inst.Name), q) ||
			strings.Contains(strings.ToLower(inst.InstanceID), q) ||
			strings.Contains(inst.PrivateIP, q) {
			result = append(result, inst)
		}
	}
	return result
}

// FilterByState filters instances by their state.
func FilterByState(instances []aws.Instance, states map[string]bool) []aws.Instance {
	if len(states) == 0 {
		return instances
	}
	var result []aws.Instance
	for _, inst := range instances {
		if states[inst.State] {
			result = append(result, inst)
		}
	}
	return result
}
