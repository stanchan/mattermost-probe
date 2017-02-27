package main

import (
	"flag"
	"io/ioutil"
	"os"

	"go.uber.org/zap"

	"github.com/stanchan/mattermost-probe/config"
	"github.com/stanchan/mattermost-probe/mattermost"
	"github.com/stanchan/mattermost-probe/metrics"
	"github.com/stanchan/mattermost-probe/probe"
	"github.com/stanchan/mattermost-probe/util"
	yaml "gopkg.in/yaml.v2"
)

func main() {
	configLocation := flag.String("config", "./config.yaml", "Config location")
	flag.Parse()

	file, err := ioutil.ReadFile(*configLocation)
	if err != nil {
		applicationExit("Config error - " + err.Error())
	}
	cfg := config.Config{}
	yaml.Unmarshal(file, &cfg)

	if err := cfg.Validate(); err != nil {
		applicationExit("Config error - " + err.Error())
	}

	server := metrics.NewServer()
	go server.Listen(cfg.BindAddr, cfg.Port)

	userA := mattermost.NewClient(cfg.Host, cfg.TeamID, server.ReportChannel)
	userB := mattermost.NewClient(cfg.Host, cfg.TeamID, server.ReportChannel)

	// Need real urls
	if err := userA.Establish(cfg.WSHost, cfg.UserA); err != nil {
		applicationExit("Could not establish user A - " + err.Error())
	}

	if err := userB.Establish(cfg.WSHost, cfg.UserB); err != nil {
		applicationExit("Could not establish user B - " + err.Error())
	}

	probes := []probe.Probe{}

	if cfg.BroadcastProbe.Enabled {
		bp := probe.NewBroadcastProbe(&cfg.BroadcastProbe, userA, userB)
		bp.TimingChannel = server.ReportChannel
		probes = append(probes, bp)
	}

	for _, p := range probes {
		if err := p.Setup(); err != nil {
			// TODO: Need a probe get name function
			applicationExit("Could not setup probe - " + err.Error())
		}
	}

	for _, p := range probes {
		if err := p.Start(); err != nil {
			// TODO: Need a probe get name function
			applicationExit("Could not start probe - " + err.Error())
		}
	}

	util.LogInfo("Inital Setup Complete")
	select {}
}

func applicationExit(msg string) {
	util.LogError("Application Error - ", zap.String("message", msg))
	os.Exit(1)
}
