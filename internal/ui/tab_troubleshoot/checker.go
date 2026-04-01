package tab_troubleshoot

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	"tui-aws/internal/aws"
)

// CheckStep represents a single step in the connectivity check.
type CheckStep struct {
	Name    string // "Source SG Outbound", "Source NACL Outbound", etc.
	Pass    bool
	Detail  string // matched rule or "NOT FOUND"
	Skipped bool   // true if previous step failed
}

// CheckResult holds the result of a connectivity check.
type CheckResult struct {
	Steps      []CheckStep
	Reachable  bool
	BlockedAt  string // which step blocked, empty if reachable
	Suggestion string // fix suggestion if blocked
}

// CheckConnectivity performs local validation of SG + Route + NACL rules.
// srcInst and dstInst are the source and destination EC2 instances.
// protocol is "tcp", "udp", or "all". port is the destination port number as string.
func CheckConnectivity(
	srcInst, dstInst aws.Instance,
	protocol string,
	port string,
	routeTables []aws.RouteTable,
	securityGroups []aws.SecurityGroup,
	nacls []aws.NetworkACL,
	subnets []aws.Subnet,
) CheckResult {
	result := CheckResult{}
	blocked := false

	// Step 1: Source SG Outbound
	step1 := checkSourceSGOutbound(srcInst, dstInst.PrivateIP, protocol, port, securityGroups)
	result.Steps = append(result.Steps, step1)
	if !step1.Pass {
		blocked = true
		result.BlockedAt = step1.Name
		result.Suggestion = fmt.Sprintf("Add outbound rule %s %s to %s/32 in source SG",
			strings.ToUpper(protocol), port, dstInst.PrivateIP)
	}

	// Step 2: Source NACL Outbound
	step2 := CheckStep{Name: "Source NACL Outbound", Skipped: blocked}
	if !blocked {
		step2 = checkNACLRules(srcInst.SubnetID, dstInst.PrivateIP, protocol, port, nacls, false)
		if !step2.Pass {
			blocked = true
			result.BlockedAt = step2.Name
			result.Suggestion = fmt.Sprintf("Add outbound NACL rule allowing %s %s to %s/32",
				strings.ToUpper(protocol), port, dstInst.PrivateIP)
		}
	}
	result.Steps = append(result.Steps, step2)

	// Step 3: Source Route
	step3 := CheckStep{Name: "Source Route", Skipped: blocked}
	if !blocked {
		step3 = checkRoute(srcInst.SubnetID, srcInst.VpcID, srcInst.VpcCIDR, dstInst.PrivateIP, routeTables)
		if !step3.Pass {
			blocked = true
			result.BlockedAt = step3.Name
			result.Suggestion = fmt.Sprintf("Add route for %s/32 in source subnet route table",
				dstInst.PrivateIP)
		}
	}
	result.Steps = append(result.Steps, step3)

	// Step 4: Destination NACL Inbound
	step4 := CheckStep{Name: "Dest NACL Inbound", Skipped: blocked}
	if !blocked {
		step4 = checkNACLRules(dstInst.SubnetID, srcInst.PrivateIP, protocol, port, nacls, true)
		if !step4.Pass {
			blocked = true
			result.BlockedAt = step4.Name
			result.Suggestion = fmt.Sprintf("Add inbound NACL rule allowing %s %s from %s/32",
				strings.ToUpper(protocol), port, srcInst.PrivateIP)
		}
	}
	result.Steps = append(result.Steps, step4)

	// Step 5: Destination SG Inbound
	step5 := CheckStep{Name: "Dest SG Inbound", Skipped: blocked}
	if !blocked {
		step5 = checkDestSGInbound(dstInst, srcInst.PrivateIP, protocol, port, securityGroups)
		if !step5.Pass {
			blocked = true
			result.BlockedAt = step5.Name
			result.Suggestion = fmt.Sprintf("Add inbound rule %s %s from %s/32 in destination SG",
				strings.ToUpper(protocol), port, srcInst.PrivateIP)
		}
	}
	result.Steps = append(result.Steps, step5)

	result.Reachable = !blocked
	return result
}

// checkSourceSGOutbound checks if source instance's SGs allow outbound traffic.
func checkSourceSGOutbound(srcInst aws.Instance, destIP, protocol, port string, allSGs []aws.SecurityGroup) CheckStep {
	step := CheckStep{Name: "Source SG Outbound"}

	for _, sg := range allSGs {
		if sg.VpcID != srcInst.VpcID {
			continue
		}
		if !instanceHasSG(srcInst, sg.Name) {
			continue
		}
		for _, rule := range sg.OutboundRules {
			if protocolMatches(rule.Protocol, protocol) &&
				portMatches(rule.PortRange, port) &&
				cidrContains(rule.Source, destIP) {
				step.Pass = true
				step.Detail = fmt.Sprintf("%s: %s %s -> %s ALLOW",
					sg.Name, rule.Protocol, rule.PortRange, rule.Source)
				return step
			}
		}
	}

	step.Pass = false
	step.Detail = "NOT FOUND"
	return step
}

// checkDestSGInbound checks if destination instance's SGs allow inbound traffic.
func checkDestSGInbound(dstInst aws.Instance, srcIP, protocol, port string, allSGs []aws.SecurityGroup) CheckStep {
	step := CheckStep{Name: "Dest SG Inbound"}

	for _, sg := range allSGs {
		if sg.VpcID != dstInst.VpcID {
			continue
		}
		if !instanceHasSG(dstInst, sg.Name) {
			continue
		}
		for _, rule := range sg.InboundRules {
			if protocolMatches(rule.Protocol, protocol) &&
				portMatches(rule.PortRange, port) &&
				cidrContains(rule.Source, srcIP) {
				step.Pass = true
				step.Detail = fmt.Sprintf("%s: %s %s <- %s ALLOW",
					sg.Name, rule.Protocol, rule.PortRange, rule.Source)
				return step
			}
		}
	}

	step.Pass = false
	step.Detail = "NOT FOUND"
	return step
}

// checkNACLRules checks NACL rules for a subnet. inbound=true checks inbound, false checks outbound.
// For outbound, checkIP is the destination IP. For inbound, checkIP is the source IP.
func checkNACLRules(subnetID, checkIP, protocol, port string, nacls []aws.NetworkACL, inbound bool) CheckStep {
	name := "Source NACL Outbound"
	if inbound {
		name = "Dest NACL Inbound"
	}
	step := CheckStep{Name: name}

	// Find NACL for subnet
	var nacl *aws.NetworkACL
	for i := range nacls {
		for _, sid := range nacls[i].Subnets {
			if sid == subnetID {
				nacl = &nacls[i]
				break
			}
		}
		if nacl != nil {
			break
		}
	}

	if nacl == nil {
		step.Pass = false
		step.Detail = "No NACL found for subnet " + subnetID
		return step
	}

	var rules []aws.NACLRule
	if inbound {
		rules = nacl.InboundRules
	} else {
		rules = nacl.OutboundRules
	}

	// Sort by rule number ascending — first match wins
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].RuleNumber < rules[j].RuleNumber
	})

	for _, rule := range rules {
		if protocolMatches(rule.Protocol, protocol) &&
			portMatches(rule.PortRange, port) &&
			cidrContains(rule.CIDRBlock, checkIP) {
			if strings.EqualFold(rule.Action, "allow") {
				step.Pass = true
				step.Detail = fmt.Sprintf("%s: Rule %d %s %s %s ALLOW",
					nacl.ID, rule.RuleNumber, rule.Protocol, rule.PortRange, rule.CIDRBlock)
			} else {
				step.Pass = false
				step.Detail = fmt.Sprintf("%s: Rule %d %s %s %s DENY",
					nacl.ID, rule.RuleNumber, rule.Protocol, rule.PortRange, rule.CIDRBlock)
			}
			return step
		}
	}

	step.Pass = false
	step.Detail = "No matching NACL rule found"
	return step
}

// checkRoute checks if a route exists from source subnet to destination IP.
func checkRoute(srcSubnetID, srcVpcID, srcVpcCIDR, destIP string, routeTables []aws.RouteTable) CheckStep {
	step := CheckStep{Name: "Source Route"}

	// Find route table for source subnet (explicit association first, then main RT)
	var rt *aws.RouteTable
	for i := range routeTables {
		for _, sid := range routeTables[i].Subnets {
			if sid == srcSubnetID {
				rt = &routeTables[i]
				break
			}
		}
		if rt != nil {
			break
		}
	}
	if rt == nil {
		// Fallback to main route table for the VPC
		for i := range routeTables {
			if routeTables[i].VpcID == srcVpcID && routeTables[i].IsMain {
				rt = &routeTables[i]
				break
			}
		}
	}

	if rt == nil {
		step.Pass = false
		step.Detail = "No route table found for subnet " + srcSubnetID
		return step
	}

	for _, route := range rt.Routes {
		// The "local" target covers VPC CIDR
		dest := route.Destination
		if route.Target == "local" && dest == srcVpcCIDR {
			if cidrContains(srcVpcCIDR, destIP) {
				step.Pass = true
				step.Detail = fmt.Sprintf("%s: %s -> %s (%s)",
					rt.ID, dest, route.Target, route.State)
				return step
			}
			continue
		}

		if cidrContains(dest, destIP) && strings.EqualFold(route.State, "active") {
			step.Pass = true
			step.Detail = fmt.Sprintf("%s: %s -> %s (%s)",
				rt.ID, dest, route.Target, route.State)
			return step
		}
	}

	step.Pass = false
	step.Detail = "No route to " + destIP
	return step
}

// instanceHasSG checks if an instance has the given security group name.
func instanceHasSG(inst aws.Instance, sgName string) bool {
	for _, name := range inst.SecurityGroups {
		if name == sgName {
			return true
		}
	}
	return false
}

// cidrContains checks if the given IP address falls within the CIDR range.
func cidrContains(cidr, ip string) bool {
	if cidr == "" || ip == "" {
		return false
	}

	// Handle non-CIDR sources (sg-xxx, pl-xxx) — skip
	if !strings.Contains(cidr, "/") && !strings.Contains(cidr, ".") && !strings.Contains(cidr, ":") {
		return false
	}

	// Ensure CIDR has a mask
	if !strings.Contains(cidr, "/") {
		cidr = cidr + "/32"
	}

	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	return ipNet.Contains(parsedIP)
}

// portMatches checks if a target port matches a rule's port range.
// Handles: "All" matches everything, single port "443" matches "443", range "1024-65535".
func portMatches(ruleRange, targetPort string) bool {
	if strings.EqualFold(ruleRange, "All") {
		return true
	}

	// Single port
	if !strings.Contains(ruleRange, "-") {
		return ruleRange == targetPort
	}

	// Port range "from-to"
	parts := strings.SplitN(ruleRange, "-", 2)
	if len(parts) != 2 {
		return false
	}
	from, err1 := strconv.Atoi(parts[0])
	to, err2 := strconv.Atoi(parts[1])
	target, err3 := strconv.Atoi(targetPort)
	if err1 != nil || err2 != nil || err3 != nil {
		return false
	}
	return target >= from && target <= to
}

// protocolMatches checks if a rule protocol matches the target protocol.
// "All" or "-1" matches everything; otherwise exact match (case-insensitive).
func protocolMatches(ruleProto, targetProto string) bool {
	rp := strings.ToLower(ruleProto)
	tp := strings.ToLower(targetProto)

	if rp == "all" || rp == "-1" {
		return true
	}
	if tp == "all" || tp == "-1" {
		return true
	}
	return rp == tp
}
