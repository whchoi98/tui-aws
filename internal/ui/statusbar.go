package ui

import "fmt"

func RenderStatusBar(profile, region, filter string, count int, width int) string {
	profilePart := StatusKeyStyle.Render("Profile: ") + profile
	regionPart := StatusKeyStyle.Render("Region: ") + region
	filterPart := StatusKeyStyle.Render("Filter: ") + filter
	countPart := fmt.Sprintf("[%d instances]", count)

	content := fmt.Sprintf(" %s  ┊  %s  ┊  %s  ┊  %s", profilePart, regionPart, filterPart, countPart)
	return StatusBarStyle.Width(width).Render(content)
}
