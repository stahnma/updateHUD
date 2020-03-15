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
	"os/signal"
	"runtime"
	"strings"
	"syscall"
)

type pkg struct {
	Name    string
	Version string
	Source  string
}

type Updater struct {
	Hostname         string
	Ip               string
	UpdatesAvailable bool
	Uptime           string
	OS               string
	OSfamily         string
	OSversion        string
	Architecture     string
	PendingUpdates   []pkg
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
	return cn
}

var knt int
var f MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	getOSinfo()
	packagesack := outdated()
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
	jresp, err := json.Marshal(response)
	if err != nil {
		fmt.Println("error:", err)
	}
	knt++
	//token := client.Publish("homeagent/"+clientname()+"/updates", 0, false, jresp)
	token := client.Publish("homeagent/updates", 0, false, jresp)
	token.Wait()
	//token = client.Publish("homeagent/"+clientname(), 0, false, clientname())
	//token.Wait()
}

func main() {
	knt = 0
	fmt.Println(outdated())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	opts := MQTT.NewClientOptions().AddBroker("tcp://angel:1883")
	opts.SetClientID(clientname())
	opts.SetDefaultPublishHandler(f)
	topic := "$SYS/broker/load/messages/received/1min"
	opts.OnConnect = func(c MQTT.Client) {
		if token := c.Subscribe(topic, 0, f); token.Wait() && token.Error() != nil {
			panic(token.Error())
		}

	}

	client := MQTT.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	} else {
		fmt.Printf("Connected to server\n")
	}
	<-c
}
