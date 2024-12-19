package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type pkg struct {
	Name    string
	Version string
	Source  string
}

type Updater struct {
	Hostname         string
	Architecture     string
	Ip               string
	UpdatesAvailable bool
	Uptime           string
	OS               string
	OSFamily         string
	OSVersion        string
	PendingUpdates   []pkg
	State            bool `json:"state"`
}

func getOSInfo() (string, string, string) {
	if _, err := os.Stat("/etc/os-release"); err == nil {
		err := godotenv.Load("/etc/os-release")
		if err != nil {
			log.Fatalf("Error loading .env file")
		}
	}

	if runtime.GOOS == "darwin" {
		os.Setenv("ID", "Darwin")
		os.Setenv("OSFamily", "macOS")
		var out []byte
		var err error
		out, err = exec.Command("/usr/sbin/system_profiler", "SPSoftwareDataType").CombinedOutput()
		if err != nil {
			log.Fatal(err)
		}
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "System Version") {
				os.Setenv("PRETTY_NAME", line)
				os.Setenv("VERSION_ID", strings.Fields(line)[3])
			}
		}
	}
	return os.Getenv("PRETTY_NAME"), os.Getenv("ID"), os.Getenv("VERSION_ID")
}

func getIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatalf("Error retrieving network interfaces: %v", err)
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}
	return "Unknown"
}

func getUptime() string {
	out, err := exec.Command("uptime").CombinedOutput()
	if err != nil {
		log.Fatalf("Error retrieving uptime: %v", err)
	}
	return strings.TrimSuffix(string(out), "\n")
}

func hasUpdates(packages []pkg) bool {
	return len(packages) > 0
}

func getArch() string {
	out, err := exec.Command("uname", "-m").CombinedOutput()
	if err != nil {
		log.Fatalf("Error retrieving architecture: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func checkRPMUpdates() []pkg {
	var packagesack []pkg
	var out []byte
	var err error

	if _, err := os.Stat("/usr/bin/yum"); err == nil {
		out, err = exec.Command("/usr/bin/yum", "check-update").CombinedOutput()
	} else if _, err := os.Stat("/usr/bin/dnf"); err == nil {
		out, err = exec.Command("/usr/bin/dnf", "check-update").CombinedOutput()
	} else {
		return packagesack // No yum or dnf found
	}

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() != 0 {
			if exitError.ExitCode() == 100 {
				// Updates are available, continue processing
			} else {
				log.Fatalf("Error running yum/dnf check-update: %v", err)
			}
		}
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		// Ignore lines with insufficient fields or headers/footers
		if len(fields) < 3 || strings.Contains(line, "Loaded plugins") || strings.TrimSpace(line) == "" {
			continue
		}

		// Assuming format: NAME VERSION REPOSITORY
		packagesack = append(packagesack, pkg{
			Name:    fields[0],
			Version: fields[1],
			Source:  fields[2],
		})
	}

	return packagesack
}

func checkAptUpdates() []pkg {
	var packagesack []pkg
	out, err := exec.Command("/usr/bin/apt", "list", "--upgradable").CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) > 1 && !strings.Contains(line, "WARNING") && !strings.Contains(line, "Listing...") {
			packagesack = append(packagesack, pkg{
				Name:    fields[0],
				Version: fields[1],
				Source:  "apt",
			})
		}
	}
	return packagesack
}

func checkBrewUpdates() []pkg {
	var packagesack []pkg
	out, err := exec.Command("brew", "outdated").CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		packagesack = append(packagesack, pkg{
			Name:   line,
			Source: "brew",
		})
	}
	return packagesack
}

func outdated() []pkg {
	var packagesack []pkg

	if runtime.GOOS == "darwin" {
		packagesack = checkBrewUpdates()
	} else if runtime.GOOS == "linux" {
		if _, err := os.Stat("/usr/bin/apt"); err == nil {
			packagesack = append(packagesack, checkAptUpdates()...)
		}
		packagesack = append(packagesack, checkRPMUpdates()...)
	}

	return packagesack
}

func buildUpdater() Updater {
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("Error retrieving hostname: %v", err)
	}
	osName, osFamily, osVersion := getOSInfo()
	packages := outdated()

	updater := Updater{
		Hostname:         strings.Split(hostname, ".")[0],
		Architecture:     getArch(),
		Ip:               getIP(),
		UpdatesAvailable: hasUpdates(packages),
		Uptime:           getUptime(),
		OS:               osName,
		OSFamily:         osFamily,
		OSVersion:        osVersion,
		PendingUpdates:   packages,
		State:            hasUpdates(packages),
	}
	return updater
}

func displayUpdater(updater Updater) {
	data, err := json.MarshalIndent(updater, "", "  ")
	if err != nil {
		log.Fatalf("Error marshalling updater struct: %v", err)
	}
	fmt.Println(string(data))
}

func main() {
	for {
		updater := buildUpdater()
		displayUpdater(updater)
		time.Sleep(10 * time.Second)
	}
}
