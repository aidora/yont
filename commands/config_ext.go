package commands

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"

	"github.com/codegangsta/cli"
	"github.com/docker/machine/utils"
)

func cmdConfig(c *cli.Context) {
	if c.Bool("swarm-master") {
		cmdConfig0(c)
		return
	}

	// original config, triggered when flag swarm-master == false
	_cmdConfig(c)
}

func cmdConfig0(c *cli.Context) {
	cfg, err := getMachineConfig(c)
	if err != nil {
		log.Fatal(err)
	}

	dockerHost, err := getHost(c).Driver.GetURL()
	if err != nil {
		log.Fatal(err)
	}

	certPath := cfg.clientCertPath
	keyPath := cfg.clientKeyPath

	if c.Bool("swarm") || c.Bool("swarm-master") {
		if !cfg.SwarmOptions.Master {
			log.Fatalf("%s is not a swarm master", cfg.machineName)
		}
		u, err := url.Parse(cfg.SwarmOptions.Host)
		if err != nil {
			log.Fatal(err)
		}
		parts := strings.Split(u.Host, ":")
		swarmPort := parts[1]

		// get IP of machine to replace in case swarm host is 0.0.0.0
		mUrl, err := url.Parse(dockerHost)
		if err != nil {
			log.Fatal(err)
		}
		mParts := strings.Split(mUrl.Host, ":")
		machineIp := mParts[0]

		dockerHost = fmt.Sprintf("tcp://%s:%s", machineIp, swarmPort)

		// config for starting as a swarm master
		if c.Bool("swarm-master") {
			certPath = filepath.Join(cfg.machineDir, "server.pem")
			keyPath = filepath.Join(cfg.machineDir, "server-key.pem")
		}

	}

	log.Debug(dockerHost)

	u, err := url.Parse(cfg.machineUrl)
	if err != nil {
		log.Fatal(err)
	}

	if u.Scheme != "unix" && getHost(c).Driver.DriverName() != "none" {

		// validate cert and regenerate if needed
		valid, err := utils.ValidateCertificate(
			u.Host,
			cfg.caCertPath,
			cfg.serverCertPath,
			cfg.serverKeyPath,
		)
		if err != nil {
			log.Fatal(err)
		}

		if !valid {
			log.Debugf("invalid certs detected; regenerating for %s", u.Host)

			if err := runActionWithContext("configureAuth", c); err != nil {
				log.Fatal(err)
			}
		}
	}

	fmt.Printf("--tlsverify --tlscacert=%q --tlscert=%q --tlskey=%q -H=%s",
		cfg.caCertPath, certPath, keyPath, dockerHost)
}
