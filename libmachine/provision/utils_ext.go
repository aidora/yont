package provision

import (
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/machine/drivers"
	"github.com/docker/machine/libmachine/auth"
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

func ConfigureAuthForNone(d drivers.Driver, authOptions auth.AuthOptions) error {
	var (
		err error
	)

	machineName := d.GetMachineName()
	org := machineName
	bits := 2048

	ip, err := d.GetIP()
	if err != nil {
		return err
	}

	// copy certs to client dir for docker client
	machineDir := filepath.Join(utils.GetMachineDir(), machineName)

	if err := utils.CopyFile(authOptions.CaCertPath, filepath.Join(machineDir, "ca.pem")); err != nil {
		log.Fatalf("Error copying ca.pem to machine dir: %s", err)
	}

	if err := utils.CopyFile(authOptions.ClientCertPath, filepath.Join(machineDir, "cert.pem")); err != nil {
		log.Fatalf("Error copying cert.pem to machine dir: %s", err)
	}

	if err := utils.CopyFile(authOptions.ClientKeyPath, filepath.Join(machineDir, "key.pem")); err != nil {
		log.Fatalf("Error copying key.pem to machine dir: %s", err)
	}

	log.Debugf("generating server cert: %s ca-key=%s private-key=%s org=%s",
		authOptions.ServerCertPath,
		authOptions.CaCertPath,
		authOptions.PrivateKeyPath,
		org,
	)

	// TODO: Switch to passing just authOptions to this func
	// instead of all these individual fields
	err = utils.GenerateCert(
		[]string{ip},
		authOptions.ServerCertPath,
		authOptions.ServerKeyPath,
		authOptions.CaCertPath,
		authOptions.PrivateKeyPath,
		org,
		bits,
	)
	if err != nil {
		return fmt.Errorf("error generating server cert: %s", err)
	}

	return nil
}
