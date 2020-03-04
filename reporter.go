package main

import (
	"encoding/json"
	"fmt"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
)

type Updater struct {
	Hostname         string
	Ip               string
	UpdatesAvailable string
	Uptime           string
	ConsulOnline     bool
	OS               string
	OSfamily         string
	OSVersion        string
	Architecture     string
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

func outdated() string {
	var out []byte
	var err error
	if runtime.GOOS == "darwin" {
		out, err = exec.Command("brew", "outdated").CombinedOutput()
		if err != nil {
			log.Fatal(err)
		}
	}
	if runtime.GOOS == "linux" {
		if _, err := os.Stat("/usr/bin/apt"); err == nil {
			fmt.Println("This is probably debian based")
			out, err = exec.Command("/usr/bin/apt", "list", "--upgradable").CombinedOutput()
			if err != nil {
				log.Fatal(err)
			}
		}
		if _, err := os.Stat("/usr/bin/yum"); err == nil {
			fmt.Println("Probably Fedora/RHEL")
			// Will exit 100 if there are updates, 0 if there are none
			out, err = exec.Command("/usr/bin/dnf", "check-update").CombinedOutput()
			if err != nil {
				fmt.Println(err)
			}
			if err == nil {
				out = nil
			}
		}
	}
	return strings.TrimSuffix(string(out), "\n")
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
	fmt.Printf("MSG: %s\n", msg.Payload())
	//text := fmt.Sprintf("this is result msg #%d!", knt)
	//	text := outdated()
	//	text += "hostname: " + clientname() + "\n"
	//	text += "ipaddress:" + myip() + "\n"
	response := Updater{
		Hostname:         clientname(),
		Ip:               myip(),
		UpdatesAvailable: outdated(),
		Uptime:           uptime(),
	}
	jresp, err := json.Marshal(response)
	if err != nil {
		fmt.Println("error:", err)
	}
	knt++
	token := client.Publish("homeagent/updates", 0, false, jresp)
	token.Wait()
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
