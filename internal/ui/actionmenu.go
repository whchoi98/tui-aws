package ui

import (
	"fmt"
	"strings"

	"tui-ssm/internal/aws"
)

type Action struct {
	Key   string
	Label string
}

type ActionMenuModel struct {
	Active   bool
	Instance aws.Instance
	Actions  []Action
	Cursor   int
}

func NewActionMenu(inst aws.Instance) ActionMenuModel {
	return ActionMenuModel{
		Active:   true,
		Instance: inst,
		Actions: []Action{
			{Key: "ssm", Label: "SSM Session (connect)"},
			{Key: "portfwd", Label: "Port Forwarding"},
			{Key: "sg", Label: "Security Groups"},
			{Key: "detail", Label: "Instance Details"},
		},
		Cursor: 0,
	}
}

func (a *ActionMenuModel) MoveUp() {
	if a.Cursor > 0 {
		a.Cursor--
	}
}

func (a *ActionMenuModel) MoveDown() {
	if a.Cursor < len(a.Actions)-1 {
		a.Cursor++
	}
}

func (a *ActionMenuModel) Selected() string {
	if a.Cursor < len(a.Actions) {
		return a.Actions[a.Cursor].Key
	}
	return ""
}

func (a *ActionMenuModel) Render(width int) string {
	if !a.Active {
		return ""
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("  %s (%s)\n", a.Instance.DisplayName(), a.Instance.InstanceID))
	b.WriteString("  ─────────────────────────\n")

	for i, action := range a.Actions {
		cursor := "  "
		if i == a.Cursor {
			cursor = "▸ "
		}
		b.WriteString(fmt.Sprintf("  %s%s\n", cursor, action.Label))
	}
	b.WriteString("\n  Enter: select  Esc: cancel")

	return OverlayStyle.Render(b.String())
}

func RenderSecurityGroups(inst aws.Instance) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("  Security Groups — %s\n", inst.DisplayName()))
	b.WriteString("  ─────────────────────────\n")

	if len(inst.SecurityGroups) == 0 {
		b.WriteString("  (none)\n")
	} else {
		for _, sg := range inst.SecurityGroups {
			b.WriteString(fmt.Sprintf("  • %s\n", sg))
		}
	}
	b.WriteString("\n  Press any key to close")
	return OverlayStyle.Render(b.String())
}

func RenderInstanceDetail(inst aws.Instance) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("  Instance Details — %s\n", inst.DisplayName()))
	b.WriteString("  ─────────────────────────────────\n")
	b.WriteString(fmt.Sprintf("  ID:         %s\n", inst.InstanceID))
	b.WriteString(fmt.Sprintf("  State:      %s %s\n", inst.StateIcon(), inst.State))
	b.WriteString(fmt.Sprintf("  Type:       %s\n", inst.InstanceType))
	b.WriteString(fmt.Sprintf("  AZ:         %s\n", inst.AvailabilityZone))
	b.WriteString(fmt.Sprintf("  Private IP: %s\n", inst.PrivateIP))
	if inst.PublicIP != "" {
		b.WriteString(fmt.Sprintf("  Public IP:  %s\n", inst.PublicIP))
	}
	if inst.VpcID != "" {
		b.WriteString(fmt.Sprintf("  VPC:        %s\n", inst.VpcID))
	}
	if inst.SubnetID != "" {
		b.WriteString(fmt.Sprintf("  Subnet:     %s\n", inst.SubnetID))
	}
	b.WriteString(fmt.Sprintf("  Platform:   %s\n", inst.Platform))
	b.WriteString(fmt.Sprintf("  Key Pair:   %s\n", inst.KeyPair))
	if inst.IAMRole != "" {
		b.WriteString(fmt.Sprintf("  IAM Role:   %s\n", inst.IAMRole))
	}
	b.WriteString(fmt.Sprintf("  Launch:     %s\n", inst.LaunchTimeFormatted()))
	if inst.SSMConnected {
		b.WriteString("  SSM:        ● Connected\n")
	} else {
		b.WriteString("  SSM:        ○ Not connected\n")
	}
	if len(inst.SecurityGroups) > 0 {
		b.WriteString(fmt.Sprintf("  SG:         %s\n", strings.Join(inst.SecurityGroups, ", ")))
	}
	b.WriteString("\n  Press any key to close")
	return OverlayStyle.Render(b.String())
}
