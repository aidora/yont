package provision

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/machine/libmachine/swarm"
	"github.com/docker/machine/utils"
)

func configureSwarm(p Provisioner, swarmOptions swarm.SwarmOptions) error {
	if p.GetDriver().DriverName() != "aiyara" {
		return configureSwarm0(p, swarmOptions)
	}

	return configureSwarmForAiyara(p, swarmOptions)
}

func configureSwarmForAiyara(p Provisioner, swarmOptions swarm.SwarmOptions) error {
	if !swarmOptions.IsSwarm {
		return nil
	}

	basePath := p.GetDockerOptionsDir()
	ip, err := p.GetDriver().GetIP()
	if err != nil {
		return err
	}

	tlsCaCert := path.Join(basePath, "ca.pem")
	tlsCert := path.Join(basePath, "server.pem")
	tlsKey := path.Join(basePath, "server-key.pem")
	masterArgs := fmt.Sprintf("--tlsverify --tlscacert=%s --tlscert=%s --tlskey=%s -H %s %s",
		tlsCaCert, tlsCert, tlsKey, swarmOptions.Host, swarmOptions.Discovery)
	nodeArgs := fmt.Sprintf("--addr %s:2376 %s", ip, swarmOptions.Discovery)

	u, err := url.Parse(swarmOptions.Host)
	if err != nil {
		return err
	}

	parts := strings.Split(u.Host, ":")
	port := parts[1]

	// TODO: Do not hardcode daemon port, ask the driver
	if err := utils.WaitForDocker(ip, 2376); err != nil {
		return err
	}

	if _, err := p.SSHCommand(fmt.Sprintf("sudo docker pull %s", swarm.AiyaraDockerImage)); err != nil {
		return err
	}

	dockerDir := p.GetDockerOptionsDir()

	// if master start master agent
	if swarmOptions.Master {
		log.Debug("launching swarm master")
		log.Debugf("master args: %s", masterArgs)
		if _, err = p.SSHCommand(fmt.Sprintf("sudo docker run -d -p %s:%s --restart=always --name swarm-agent-master -v %s:%s %s manage %s",
			port, port, dockerDir, dockerDir, swarm.AiyaraDockerImage, masterArgs)); err != nil {
			return err
		}
	}

	// start node agent
	log.Debug("launching swarm node")
	log.Debugf("node args: %s", nodeArgs)
	if _, err = p.SSHCommand(fmt.Sprintf("sudo docker run -d --restart=always --name swarm-agent -v %s:%s %s join %s",
		dockerDir, dockerDir, swarm.AiyaraDockerImage, nodeArgs)); err != nil {
		return err
	}

	return nil
}
