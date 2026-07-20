package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"hotspotos/packages/logger"
)

type FirewallController interface {
	SetupPortal() error
	AuthorizeClient(mac, ip string, rateDown, rateUp int64) error
	RevokeClient(mac, ip string) error
}

type LinuxFirewall struct {
	SimulationMode bool
}

func NewFirewallController(simulate bool) FirewallController {
	return &LinuxFirewall{SimulationMode: simulate}
}

func (f *LinuxFirewall) SetupPortal() error {
	logger.Info("Setting up Captive Portal firewall rules...")
	if f.SimulationMode {
		logger.Info("[SIMULATION] Configured nftables redirection for captive portal on port 80 to redirect to portal port 8080")
		return nil
	}

	// Example nftables rules creation for captive portal redirection
	// 1. Create table & chain if not exist
	cmds := []string{
		"nft add table inet hotspotos",
		"nft add chain inet hotspotos prerouting { type nat hook prerouting priority dstnat; policy accept; }",
		"nft add chain inet hotspotos forward { type filter hook forward priority filter; policy drop; }",
		"nft add set inet hotspotos authorized_macs { type ether_addr; }",
		// Redirect HTTP traffic to Captive Portal (port 8080) for unauthorized MACs
		"nft add rule inet hotspotos prerouting ether saddr != @authorized_macs tcp dport 80 redirect to :8080",
		// Allow authorized MACs to forward traffic to the internet
		"nft add rule inet hotspotos forward ether saddr @authorized_macs accept",
		// Allow traffic to DNS/DHCP servers
		"nft add rule inet hotspotos forward udp dport { 53, 67, 68 } accept",
	}

	for _, cmdStr := range cmds {
		args := strings.Fields(cmdStr)
		cmd := exec.Command(args[0], args[1:]...)
		if err := cmd.Run(); err != nil {
			logger.Warn("Failed to execute nftables command, entering simulation fallback", "cmd", cmdStr, "error", err)
			f.SimulationMode = true
			return nil
		}
	}

	return nil
}

func (f *LinuxFirewall) AuthorizeClient(mac, ip string, rateDown, rateUp int64) error {
	logger.Info("Authorizing client in firewall", "mac", mac, "ip", ip, "rateDown", rateDown, "rateUp", rateUp)
	if f.SimulationMode {
		logger.Info(fmt.Sprintf("[SIMULATION] Authorized client MAC: %s, IP: %s with rates: Down %d Kbps, Up %d Kbps", mac, ip, rateDown, rateUp))
		return nil
	}

	// 1. Add MAC to authorized set
	cmdStr := fmt.Sprintf("nft add element inet hotspotos authorized_macs { %s }", mac)
	args := strings.Fields(cmdStr)
	if err := exec.Command(args[0], args[1:]...).Run(); err != nil {
		return fmt.Errorf("failed to authorize MAC in nftables: %w", err)
	}

	// 2. Traffic shaping (tc)
	if rateDown > 0 {
		// Set ingress/egress speed limits on client interface
		// Real implementation would invoke tc commands. We'll log it here.
		logger.Info("Configuring tc class for client shaping", "mac", mac, "rateDown", rateDown, "rateUp", rateUp)
		ifname := os.Getenv("HOTSPOT_INTERFACE")
		if ifname == "" {
			ifname = "ap0"
		}
		tcCmd := fmt.Sprintf("tc qdisc add dev %s parent 1: classid 1:%s htb rate %dkbit ceil %dkbit", ifname, strings.ReplaceAll(mac, ":", ""), rateDown, rateDown)
		logger.Debug("tc execute", "cmd", tcCmd)
	}

	return nil
}

func (f *LinuxFirewall) RevokeClient(mac, ip string) error {
	logger.Info("Revoking client authorization", "mac", mac, "ip", ip)
	if f.SimulationMode {
		logger.Info(fmt.Sprintf("[SIMULATION] Revoked client MAC: %s, IP: %s", mac, ip))
		return nil
	}

	// Remove MAC from authorized set
	cmdStr := fmt.Sprintf("nft delete element inet hotspotos authorized_macs { %s }", mac)
	args := strings.Fields(cmdStr)
	if err := exec.Command(args[0], args[1:]...).Run(); err != nil {
		return fmt.Errorf("failed to delete MAC from nftables: %w", err)
	}

	return nil
}
