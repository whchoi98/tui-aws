package tab_lambda

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

func DefaultColumns() []shared.Column {
	return []shared.Column{
		{Key: "name", Title: "Name", Width: 30},
		{Key: "runtime", Title: "Runtime", Width: 12},
		{Key: "memory", Title: "Mem", Width: 6},
		{Key: "timeout", Title: "T/O", Width: 4},
		{Key: "state", Title: "State", Width: 8},
		{Key: "vpc", Title: "VPC", Width: 15},
		{Key: "modified", Title: "Last Modified", Width: 18},
	}
}

func CompactColumns() []shared.Column {
	return []shared.Column{
		{Key: "name", Title: "Name", Width: 30},
		{Key: "runtime", Title: "Runtime", Width: 12},
		{Key: "state", Title: "State", Width: 8},
		{Key: "modified", Title: "Last Modified", Width: 18},
	}
}

func ColumnsForWidth(width int) []shared.Column {
	if width < 100 {
		return shared.ExpandNameColumn(CompactColumns(), width)
	}
	return shared.ExpandNameColumn(DefaultColumns(), width)
}

func RenderTable(functions []aws.LambdaFunction, columns []shared.Column, cursor, width, height int) string {
	var b strings.Builder

	header := shared.RenderRow(columns, func(col shared.Column) string {
		return col.Title
	}, nil)
	b.WriteString(shared.TableHeaderStyle.Width(width).Render(header))
	b.WriteString("\n")

	maxRows := height - 4
	if maxRows < 1 {
		maxRows = 1
	}

	offset := 0
	if cursor >= maxRows {
		offset = cursor - maxRows + 1
	}

	for i := offset; i < len(functions) && i < offset+maxRows; i++ {
		fn := functions[i]
		row := shared.RenderRow(columns, func(col shared.Column) string {
			return cellValue(col.Key, fn)
		}, func(col shared.Column) lipgloss.Style {
			return cellStyle(col.Key, fn)
		})

		if i == cursor {
			row = shared.TableSelectedStyle.Width(width).Render(row)
		}
		b.WriteString(row)
		if i < offset+maxRows-1 && i < len(functions)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func cellValue(key string, fn aws.LambdaFunction) string {
	switch key {
	case "name":
		return fn.Name
	case "runtime":
		return fn.Runtime
	case "memory":
		return fmt.Sprintf("%dMB", fn.MemorySize)
	case "timeout":
		return fmt.Sprintf("%ds", fn.Timeout)
	case "state":
		if fn.State == "" {
			return "Active"
		}
		return fn.State
	case "vpc":
		if fn.VpcID == "" {
			return "-"
		}
		return fn.VpcID
	case "modified":
		return fn.LastModified
	default:
		return ""
	}
}

func cellStyle(key string, fn aws.LambdaFunction) lipgloss.Style {
	if key == "state" {
		return lambdaStateStyle(fn.State)
	}
	return lipgloss.Style{}
}

func lambdaStateStyle(state string) lipgloss.Style {
	switch state {
	case "Active", "":
		return shared.StateRunning
	case "Pending":
		return shared.StatePending
	case "Inactive":
		return shared.StateTerminated // gray
	case "Failed":
		return shared.StateStopped
	default:
		return lipgloss.Style{}
	}
}

func renderStatusBar(profile, region string, count, width int) string {
	profilePart := shared.StatusKeyStyle.Render("Profile: ") + profile
	regionPart := shared.StatusKeyStyle.Render("Region: ") + region
	countPart := fmt.Sprintf("[%d Functions]", count)
	content := fmt.Sprintf(" %s  |  %s  |  %s", profilePart, regionPart, countPart)
	return shared.StatusBarStyle.Width(width).Render(content)
}
