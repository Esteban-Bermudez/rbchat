package network

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func ComputeNetworkID() string {
	gw, err := defaultGateway()
	if err != nil {
		return ""
	}
	mac, err := gatewayMAC(gw)
	if err != nil {
		return ""
	}
	if mac == "" {
		return ""
	}
	h := sha256.Sum256([]byte(mac))
	return hex.EncodeToString(h[:8])
}

func defaultGateway() (net.IP, error) {
	switch runtime.GOOS {
	case "darwin":
		return gatewayDarwin()
	case "linux":
		return gatewayLinux()
	case "windows":
		return gatewayWindows()
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func gatewayDarwin() (net.IP, error) {
	out, err := exec.Command("route", "-n", "get", "default").Output()
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "    gateway: ") {
			ip := net.ParseIP(strings.TrimSpace(line[len("    gateway: "):]))
			if ip != nil {
				return ip, nil
			}
		}
	}
	return nil, fmt.Errorf("no default gateway found")
}

func gatewayLinux() (net.IP, error) {
	data, err := os.ReadFile("/proc/net/route")
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		if fields[1] != "00000000" {
			continue
		}
		gwHex := fields[2]
		if len(gwHex) != 8 {
			continue
		}
		bytes, err := hex.DecodeString(gwHex)
		if err != nil || len(bytes) != 4 {
			continue
		}
		ip := net.IP(bytes)
		return ip, nil
	}
	return nil, fmt.Errorf("no default gateway found")
}

func gatewayWindows() (net.IP, error) {
	out, err := exec.Command("route", "print").Output()
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	foundHeader := false
	for scanner.Scan() {
		line := scanner.Text()
		if !foundHeader {
			if strings.Contains(line, "Network Destination") && strings.Contains(line, "Netmask") && strings.Contains(line, "Gateway") {
				foundHeader = true
			}
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		if fields[0] == "0.0.0.0" && fields[1] == "0.0.0.0" {
			ip := net.ParseIP(fields[2])
			if ip != nil {
				return ip, nil
			}
		}
	}
	return nil, fmt.Errorf("no default gateway found")
}

func gatewayMAC(gw net.IP) (string, error) {
	var out []byte
	var err error
	if runtime.GOOS == "windows" {
		out, err = exec.Command("arp", "-a", gw.String()).Output()
	} else {
		out, err = exec.Command("arp", "-n", gw.String()).Output()
	}
	if err != nil {
		return "", err
	}
	return parseARPMac(string(out)), nil
}

func parseARPMac(output string) string {
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if runtime.GOOS == "windows" {
			fields := strings.Fields(line)
			if len(fields) >= 2 && strings.Count(fields[1], "-") == 5 {
				candidate := strings.ReplaceAll(fields[1], "-", ":")
				if isValidMAC(candidate) {
					return candidate
				}
			}
		} else {
			idx := strings.Index(line, " at ")
			if idx < 0 {
				continue
			}
			rest := line[idx+4:]
			end := strings.IndexAny(rest, " ")
			if end < 0 {
				end = len(rest)
			}
			candidate := rest[:end]
			if isValidMAC(candidate) {
				return candidate
			}
		}
	}
	return ""
}

func isValidMAC(mac string) bool {
	if strings.Count(mac, ":") != 5 && strings.Count(mac, "-") != 5 {
		return false
	}
	sep := ":"
	if strings.Count(mac, "-") == 5 {
		sep = "-"
	}
	parts := strings.Split(mac, sep)
	if len(parts) != 6 {
		return false
	}
	for _, p := range parts {
		if len(p) != 2 {
			return false
		}
		for _, c := range p {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}
