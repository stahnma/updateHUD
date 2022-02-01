package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/joho/godotenv"
	"log"
	"net"
	"os"
	"os/exec"
	//	"os/signal"
	"runtime"
	"strings"
	//	"syscall"
	"time"
)

const delay = 10
const server = "spike"
const port = "1883"

// https://www.home-assistant.io/docs/mqtt/discovery/
type haconfig struct {
	//Deviceclass   string `json:"device_class"` //hostname
	Name              string `json:"name"`
	Statetopic        string `json:"state_topic"`
	Attributetemplate string `json:"json_attributes_template"`
	Attributetopic    string `json:"json_attributes_topic"`
	Uniqueid          string `json:"unique_id"`
}

type updatestate struct {
	State    bool   `json:"state"`
	Hostname string `json:"name"`
}

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
	OSfamily         string
	OSversion        string
	PendingUpdates   []pkg
	State            bool `json:"state"`
}

func getOSinfo() {
	if _, err := os.Stat("/etc/os-release"); err == nil {
		err := godotenv.Load("/etc/os-release")
		if err != nil {
			log.Fatalf("Error loading .env file")
		}
	}
	if runtime.GOOS == "darwin" {
		os.Setenv("ID", "Darwin")
		os.Setenv("", "macOS")
		var out []byte
		var err error
		out, err = exec.Command("/usr/sbin/system_profiler", "SPSoftwareDataType").CombinedOutput()
		//TODO trim leading whitespace on result
		if err != nil {
			log.Fatal(err)
		}
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			line := scanner.Text()
			if !(strings.Contains(line, "System Version")) {
				continue
			}
			os.Setenv("PRETTY_NAME", line)
			os.Setenv("VERSION_ID", strings.Fields(line)[3])
		}
	}
}

func myip() string {
	var ipa string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		os.Stderr.WriteString("Oops: " + err.Error() + "\n")
		os.Exit(1)
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ipa = ipnet.IP.String()
			}
		}
	}
	return ipa
}

func uptime() string {
	var out []byte
	var err error
	out, err = exec.Command("uptime").CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	return strings.TrimSuffix(string(out), "\n")
}

func hasupdates(packagesack []pkg) bool {
	var b bool
	if len(packagesack) > 0 {
		b = true
	} else {
		b = false
	}
	return b
}

func getArch() string {
	out, err := exec.Command("uname", "-m").CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	return strings.TrimSuffix(string(out), "\n")
}

func outdated() []pkg {
	var packagesack []pkg
	var out []byte
	var err error
	if runtime.GOOS == "darwin" {
		out, err = exec.Command("brew", "outdated").CombinedOutput()
		if err != nil {
			log.Fatal(err)
		}
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			var p pkg
			//fmt.Println(scanner.Text())
			p.Name = scanner.Text()
			p.Source = "brew"
			packagesack = append(packagesack, p)
		}
	}
	if runtime.GOOS == "linux" {
		if _, err := os.Stat("/usr/bin/apt"); err == nil {
			//fmt.Println("This is probably debian based")
			out, err = exec.Command("/usr/bin/apt", "list", "--upgradable").CombinedOutput()
			if err != nil {
				log.Fatal(err)
			}
			scanner := bufio.NewScanner(strings.NewReader(string(out)))
			for scanner.Scan() {
				var p pkg
				//fmt.Println(scanner.Text())
				line := scanner.Text()
				if strings.Contains(line, "WARNING") || strings.Contains(line, "Listing...") {
					continue
				}
				if len(strings.TrimSpace(line)) == 0 {
					continue
				}
				p.Name = strings.Fields(line)[0]
				p.Version = strings.Fields(line)[1]
				p.Source = "apt"
				packagesack = append(packagesack, p)
			}
		}
		if _, err := os.Stat("/usr/bin/yum"); err == nil {
			//fmt.Println("Probably Fedora/RHEL")
			// Will exit 100 if there are updates, 0 if there are none
			out, err = exec.Command("/usr/bin/dnf", "check-update").CombinedOutput()
			if err != nil {
				fmt.Println(err)
			}
			scanner := bufio.NewScanner(strings.NewReader(string(out)))
			for scanner.Scan() {
				var p pkg
				//fmt.Println(scanner.Text())
				line := scanner.Text()
				if strings.Contains(line, "expiration check") || len(strings.Fields(line)) < 3 {
					continue
				}
				if len(strings.TrimSpace(line)) == 0 {
					continue
				}
				p.Name = strings.Fields(line)[0]
				p.Version = strings.Fields(line)[1]
				p.Source = strings.Fields(line)[2]
				packagesack = append(packagesack, p)
			}
		}
	}
	return packagesack
}

func clientname() string {
	cn, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	return strings.Split(cn, ".")[0]
}

func send_the_messages(client MQTT.Client) {
	getOSinfo()
	packagesack := outdated()
	fmt.Println(packagesack)
	response := Updater{
		Hostname:         clientname(),
		Architecture:     getArch(),
		OS:               os.Getenv("PRETTY_NAME"),
		OSfamily:         os.Getenv("ID"),
		OSversion:        os.Getenv("VERSION_ID"),
		Ip:               myip(),
		UpdatesAvailable: hasupdates(packagesack),
		Uptime:           uptime(),
		PendingUpdates:   packagesack,
	}
	response.State = response.UpdatesAvailable

	upstate := updatestate{
		State:    response.UpdatesAvailable,
		Hostname: response.Hostname,
	}

	basetopic := "homeassistant/sensor/" + clientname()
	statetopic := basetopic + "/state"
	configtopic := basetopic + "/config"
	attributestopic := basetopic + "/attrs"

	hac := haconfig{
		Name:              "Updates",
		Statetopic:        statetopic,
		Attributetemplate: "{{ state }}",
		Attributetopic:    attributestopic,
		Uniqueid:          response.Hostname,
	}

	ustate, err := json.Marshal(upstate)
	if err != nil {
		fmt.Println("error:", err)
	}
	fmt.Println(ustate)

	jresp, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		fmt.Println("error:", err)
	}

	hacjson, err := json.Marshal(hac)
	if err != nil {
		fmt.Println("error:", err)
	}

	fmt.Println("===Publish config at " + configtopic)
	fmt.Println(string(hacjson))
	token := client.Publish(configtopic, 0, false, hacjson)
	token.Wait()

	fmt.Println("===Publish state at " + statetopic)
	fmt.Println(string(ustate))
	token = client.Publish(statetopic, 0, false, ustate)
	token.Wait()

	fmt.Println("===Publish attributes at " + attributestopic)
	token = client.Publish(attributestopic, 0, false, jresp)
	fmt.Println(string(jresp))
	token.Wait()
}

func main() {
	broker_string := "tcp://" + server + ":" + port
	opts := MQTT.NewClientOptions().AddBroker(broker_string)
	opts.SetClientID(clientname())
	client := MQTT.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	} else {
		fmt.Printf("Connected to " + server + "\n")
		for {
			send_the_messages(client)
			time.Sleep(delay * 1000 * time.Millisecond)
		}
	}
}
