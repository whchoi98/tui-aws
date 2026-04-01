package tab_troubleshoot

import (
	"testing"

	"tui-aws/internal/aws"
)

func TestCIDRContains(t *testing.T) {
	tests := []struct {
		cidr   string
		ip     string
		expect bool
	}{
		{"10.0.0.0/8", "10.1.2.3", true},
		{"10.0.0.0/8", "192.168.1.1", false},
		{"0.0.0.0/0", "10.1.2.3", true},
		{"0.0.0.0/0", "192.168.1.1", true},
		{"10.1.0.0/16", "10.1.88.66", true},
		{"10.1.0.0/16", "10.2.0.1", false},
		{"10.1.88.66/32", "10.1.88.66", true},
		{"10.1.88.66/32", "10.1.88.67", false},
		// Non-CIDR sources should not match
		{"sg-12345", "10.0.0.1", false},
		{"pl-12345", "10.0.0.1", false},
		// Empty values
		{"", "10.0.0.1", false},
		{"10.0.0.0/8", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.cidr+"_"+tt.ip, func(t *testing.T) {
			got := cidrContains(tt.cidr, tt.ip)
			if got != tt.expect {
				t.Errorf("cidrContains(%q, %q) = %v, want %v", tt.cidr, tt.ip, got, tt.expect)
			}
		})
	}
}

func TestPortMatches(t *testing.T) {
	tests := []struct {
		ruleRange  string
		targetPort string
		expect     bool
	}{
		{"All", "443", true},
		{"All", "80", true},
		{"443", "443", true},
		{"443", "80", false},
		{"80", "80", true},
		{"1024-65535", "8080", true},
		{"1024-65535", "80", false},
		{"1024-65535", "1024", true},
		{"1024-65535", "65535", true},
		{"1024-65535", "1023", false},
		{"22", "22", true},
		{"22", "443", false},
	}

	for _, tt := range tests {
		t.Run(tt.ruleRange+"_"+tt.targetPort, func(t *testing.T) {
			got := portMatches(tt.ruleRange, tt.targetPort)
			if got != tt.expect {
				t.Errorf("portMatches(%q, %q) = %v, want %v", tt.ruleRange, tt.targetPort, got, tt.expect)
			}
		})
	}
}

func TestProtocolMatches(t *testing.T) {
	tests := []struct {
		ruleProto   string
		targetProto string
		expect      bool
	}{
		{"All", "tcp", true},
		{"-1", "tcp", true},
		{"tcp", "tcp", true},
		{"udp", "udp", true},
		{"tcp", "udp", false},
		{"tcp", "all", true},
		{"TCP", "tcp", true},
	}

	for _, tt := range tests {
		t.Run(tt.ruleProto+"_"+tt.targetProto, func(t *testing.T) {
			got := protocolMatches(tt.ruleProto, tt.targetProto)
			if got != tt.expect {
				t.Errorf("protocolMatches(%q, %q) = %v, want %v", tt.ruleProto, tt.targetProto, got, tt.expect)
			}
		})
	}
}

// buildTestData creates minimal test infrastructure for connectivity checks.
func buildTestData() (
	srcInst, dstInst aws.Instance,
	routeTables []aws.RouteTable,
	securityGroups []aws.SecurityGroup,
	nacls []aws.NetworkACL,
	subnets []aws.Subnet,
) {
	srcInst = aws.Instance{
		InstanceID:     "i-src",
		Name:           "web-server",
		PrivateIP:      "10.1.1.10",
		VpcID:          "vpc-1",
		VpcCIDR:        "10.1.0.0/16",
		SubnetID:       "subnet-src",
		SubnetCIDR:     "10.1.1.0/24",
		SecurityGroups: []string{"web-sg"},
	}

	dstInst = aws.Instance{
		InstanceID:     "i-dst",
		Name:           "db-primary",
		PrivateIP:      "10.1.2.10",
		VpcID:          "vpc-1",
		VpcCIDR:        "10.1.0.0/16",
		SubnetID:       "subnet-dst",
		SubnetCIDR:     "10.1.2.0/24",
		SecurityGroups: []string{"db-sg"},
	}

	routeTables = []aws.RouteTable{
		{
			ID:     "rtb-main",
			VpcID:  "vpc-1",
			IsMain: true,
			Routes: []aws.Route{
				{Destination: "10.1.0.0/16", Target: "local", State: "active"},
			},
		},
	}

	securityGroups = []aws.SecurityGroup{
		{
			ID:    "sg-web",
			Name:  "web-sg",
			VpcID: "vpc-1",
			OutboundRules: []aws.SGRule{
				{Protocol: "All", PortRange: "All", Source: "0.0.0.0/0"},
			},
			InboundRules: []aws.SGRule{
				{Protocol: "tcp", PortRange: "80", Source: "0.0.0.0/0"},
				{Protocol: "tcp", PortRange: "443", Source: "0.0.0.0/0"},
			},
		},
		{
			ID:    "sg-db",
			Name:  "db-sg",
			VpcID: "vpc-1",
			OutboundRules: []aws.SGRule{
				{Protocol: "All", PortRange: "All", Source: "0.0.0.0/0"},
			},
			InboundRules: []aws.SGRule{
				{Protocol: "tcp", PortRange: "3306", Source: "10.1.1.0/24"},
			},
		},
	}

	nacls = []aws.NetworkACL{
		{
			ID:      "acl-src",
			VpcID:   "vpc-1",
			Subnets: []string{"subnet-src"},
			InboundRules: []aws.NACLRule{
				{RuleNumber: 100, Protocol: "All", PortRange: "All", CIDRBlock: "0.0.0.0/0", Action: "allow"},
			},
			OutboundRules: []aws.NACLRule{
				{RuleNumber: 100, Protocol: "All", PortRange: "All", CIDRBlock: "0.0.0.0/0", Action: "allow"},
			},
		},
		{
			ID:      "acl-dst",
			VpcID:   "vpc-1",
			Subnets: []string{"subnet-dst"},
			InboundRules: []aws.NACLRule{
				{RuleNumber: 100, Protocol: "All", PortRange: "All", CIDRBlock: "0.0.0.0/0", Action: "allow"},
			},
			OutboundRules: []aws.NACLRule{
				{RuleNumber: 100, Protocol: "All", PortRange: "All", CIDRBlock: "0.0.0.0/0", Action: "allow"},
			},
		},
	}

	subnets = []aws.Subnet{
		{SubnetID: "subnet-src", VpcID: "vpc-1", CIDRBlock: "10.1.1.0/24"},
		{SubnetID: "subnet-dst", VpcID: "vpc-1", CIDRBlock: "10.1.2.0/24"},
	}

	return
}

func TestCheckConnectivity_AllAllowed(t *testing.T) {
	srcInst, dstInst, rts, sgs, nacls, subnets := buildTestData()

	// Test: web-server -> db-primary TCP/3306 (should be allowed by all)
	result := CheckConnectivity(srcInst, dstInst, "tcp", "3306", rts, sgs, nacls, subnets)

	if !result.Reachable {
		t.Errorf("Expected reachable, got blocked at %q", result.BlockedAt)
		for _, step := range result.Steps {
			t.Logf("  %s: pass=%v detail=%s skipped=%v", step.Name, step.Pass, step.Detail, step.Skipped)
		}
	}

	if len(result.Steps) != 5 {
		t.Errorf("Expected 5 steps, got %d", len(result.Steps))
	}

	for _, step := range result.Steps {
		if step.Skipped {
			t.Errorf("Step %q should not be skipped", step.Name)
		}
		if !step.Pass {
			t.Errorf("Step %q should pass, detail: %s", step.Name, step.Detail)
		}
	}
}

func TestCheckConnectivity_DestSGBlocks(t *testing.T) {
	srcInst, dstInst, rts, sgs, nacls, subnets := buildTestData()

	// Test: web-server -> db-primary TCP/443 (dest SG only allows 3306)
	result := CheckConnectivity(srcInst, dstInst, "tcp", "443", rts, sgs, nacls, subnets)

	if result.Reachable {
		t.Error("Expected not reachable, got reachable")
	}

	if result.BlockedAt != "Dest SG Inbound" {
		t.Errorf("Expected blocked at 'Dest SG Inbound', got %q", result.BlockedAt)
	}

	// First 4 steps should pass, step 5 should fail
	for i, step := range result.Steps {
		if i < 4 {
			if !step.Pass {
				t.Errorf("Step %d (%s) should pass", i, step.Name)
			}
			if step.Skipped {
				t.Errorf("Step %d (%s) should not be skipped", i, step.Name)
			}
		} else {
			if step.Pass {
				t.Errorf("Step %d (%s) should fail", i, step.Name)
			}
		}
	}

	if result.Suggestion == "" {
		t.Error("Expected a suggestion, got empty string")
	}
}

func TestCheckConnectivity_SourceSGBlocks(t *testing.T) {
	srcInst, dstInst, rts, sgs, nacls, subnets := buildTestData()

	// Override source SG to only allow port 80 outbound
	sgs[0].OutboundRules = []aws.SGRule{
		{Protocol: "tcp", PortRange: "80", Source: "0.0.0.0/0"},
	}

	result := CheckConnectivity(srcInst, dstInst, "tcp", "3306", rts, sgs, nacls, subnets)

	if result.Reachable {
		t.Error("Expected not reachable")
	}

	if result.BlockedAt != "Source SG Outbound" {
		t.Errorf("Expected blocked at 'Source SG Outbound', got %q", result.BlockedAt)
	}

	// Steps 2-5 should be skipped
	for i := 1; i < len(result.Steps); i++ {
		if !result.Steps[i].Skipped {
			t.Errorf("Step %d (%s) should be skipped", i, result.Steps[i].Name)
		}
	}
}

func TestCheckConnectivity_NACLDeny(t *testing.T) {
	srcInst, dstInst, rts, sgs, nacls, subnets := buildTestData()

	// Add a DENY rule before the ALLOW in source NACL outbound
	nacls[0].OutboundRules = []aws.NACLRule{
		{RuleNumber: 50, Protocol: "tcp", PortRange: "3306", CIDRBlock: "10.1.2.0/24", Action: "deny"},
		{RuleNumber: 100, Protocol: "All", PortRange: "All", CIDRBlock: "0.0.0.0/0", Action: "allow"},
	}

	result := CheckConnectivity(srcInst, dstInst, "tcp", "3306", rts, sgs, nacls, subnets)

	if result.Reachable {
		t.Error("Expected not reachable")
	}

	if result.BlockedAt != "Source NACL Outbound" {
		t.Errorf("Expected blocked at 'Source NACL Outbound', got %q", result.BlockedAt)
	}
}

func TestCheckConnectivity_CrossVPC(t *testing.T) {
	srcInst, dstInst, _, sgs, nacls, subnets := buildTestData()

	// Put destination in a different VPC
	dstInst.VpcID = "vpc-2"
	dstInst.VpcCIDR = "10.2.0.0/16"
	dstInst.PrivateIP = "10.2.1.10"

	// Route table with TGW route
	rts := []aws.RouteTable{
		{
			ID:     "rtb-src",
			VpcID:  "vpc-1",
			IsMain: true,
			Routes: []aws.Route{
				{Destination: "10.1.0.0/16", Target: "local", State: "active"},
				{Destination: "10.2.0.0/16", Target: "tgw-xxx", State: "active"},
			},
		},
	}

	// Update dest SG to be in vpc-2
	destSG := aws.SecurityGroup{
		ID:    "sg-db-vpc2",
		Name:  "db-sg",
		VpcID: "vpc-2",
		InboundRules: []aws.SGRule{
			{Protocol: "tcp", PortRange: "3306", Source: "10.1.0.0/16"},
		},
		OutboundRules: []aws.SGRule{
			{Protocol: "All", PortRange: "All", Source: "0.0.0.0/0"},
		},
	}
	sgs = append(sgs, destSG)

	// Add NACL for dest in vpc-2
	dstNACL := aws.NetworkACL{
		ID:      "acl-dst-vpc2",
		VpcID:   "vpc-2",
		Subnets: []string{"subnet-dst"},
		InboundRules: []aws.NACLRule{
			{RuleNumber: 100, Protocol: "All", PortRange: "All", CIDRBlock: "0.0.0.0/0", Action: "allow"},
		},
		OutboundRules: []aws.NACLRule{
			{RuleNumber: 100, Protocol: "All", PortRange: "All", CIDRBlock: "0.0.0.0/0", Action: "allow"},
		},
	}
	nacls = append(nacls, dstNACL)

	result := CheckConnectivity(srcInst, dstInst, "tcp", "3306", rts, sgs, nacls, subnets)

	if !result.Reachable {
		t.Errorf("Expected reachable, got blocked at %q", result.BlockedAt)
		for _, step := range result.Steps {
			t.Logf("  %s: pass=%v detail=%s skipped=%v", step.Name, step.Pass, step.Detail, step.Skipped)
		}
	}
}
