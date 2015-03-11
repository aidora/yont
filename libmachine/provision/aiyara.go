package provision

import (
	"bytes"
	"fmt"
	"io/ioutil"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/machine/drivers"
	"github.com/docker/machine/libmachine/auth"
	"github.com/docker/machine/libmachine/provision/pkgaction"
	"github.com/docker/machine/libmachine/swarm"
	"github.com/docker/machine/ssh"
	"github.com/docker/machine/utils"
)

func init() {
	Register("Aiyara", &RegisteredProvisioner{
		New: NewAiyaraProvisioner,
	})
}

func NewAiyaraProvisioner(d drivers.Driver) Provisioner {
	return &AiyaraProvisioner{
		packages: []string{
			"curl",
		},
		Driver: d,
	}
}

type AiyaraProvisioner struct {
	packages      []string
	OsReleaseInfo *OsRelease
	Driver        drivers.Driver
	SwarmOptions  swarm.SwarmOptions
}

func (provisioner *AiyaraProvisioner) Service(name string, action pkgaction.ServiceAction) error {
	command := fmt.Sprintf("sudo service %s %s", name, action.String())

	if _, err := provisioner.SSHCommand(command); err != nil {
		return err
	}

	return nil
}

func (provisioner *AiyaraProvisioner) Package(name string, action pkgaction.PackageAction) error {
	log.Debug("Package doing nothing")
	return nil
}

func (provisioner *AiyaraProvisioner) dockerDaemonResponding() bool {
	if _, err := provisioner.SSHCommand("sudo docker version"); err != nil {
		log.Warn("Error getting SSH command to check if the daemon is up: %s", err)
		return false
	}

	// The daemon is up if the command worked.  Carry on.
	return true
}

func (provisioner *AiyaraProvisioner) installPublicKey() error {

	provisioner.SSHCommand("mkdir ~/.ssh")

	publicKey, err := ioutil.ReadFile(provisioner.Driver.GetSSHKeyPath() + ".pub")
	if err != nil {
		return err
	}
	command := fmt.Sprintf("echo \"%s\" | tee -a ~/.ssh/authorized_keys", string(publicKey))
	if _, err := provisioner.SSHCommand(command); err != nil {
		return err
	}

	return nil
}

const DockerBinary = "docker-1.6.0"

func (provisioner *AiyaraProvisioner) installCustomDocker() error {
	// if the old version running, stop it
	provisioner.Service("docker", pkgaction.Stop)

	// direct kill
	provisioner.SSHCommand("kill -9 `pidof docker`")

	// delete the old configuration
	provisioner.SSHCommand("rm /etc/default/docker")

	provisioner.SSHCommand("unlink /usr/bin/docker")

	provisioner.SSHCommand("mkdir -p /opt/docker")

	provisioner.SSHCommand("unlink /opt/docker/docker-*")

	if _, err := provisioner.SSHCommand("wget --no-check-certificate -q -O/opt/docker/" + DockerBinary + ".xz https://dl.dropboxusercontent.com/u/381580/docker/" + DockerBinary + ".xz && (cd /opt/docker && unxz -f " + DockerBinary + ".xz)"); err != nil {
		log.Debug("Failed getting Docker binary")
		return err
	}

	if _, err := provisioner.SSHCommand("chmod +x /opt/docker/" + DockerBinary + " && ln -s /opt/docker/" + DockerBinary + " /usr/bin/docker"); err != nil {
		log.Debug("Failed installing docker to /usr/bin")
		return err
	}

	// install init.d script
	if _, err := provisioner.SSHCommand("wget --no-check-certificate -q -O/etc/init.d/docker https://dl.dropboxusercontent.com/u/381580/docker/initd-docker && chmod +x /etc/init.d/docker"); err != nil {
		log.Debug("Failed getting init.d script")
		return err
	}

	provisioner.Service("docker", pkgaction.Start)

	return nil
}

func (provisioner *AiyaraProvisioner) Provision(swarmOptions swarm.SwarmOptions, authOptions auth.AuthOptions) error {

	log.Debug("Entering Provision")

	if err := provisioner.SetHostname(provisioner.Driver.GetMachineName()); err != nil {
		return err
	}

	if err := provisioner.installPublicKey(); err != nil {
		return err
	}

	if d0, ok := provisioner.Driver.(interface {
		ClearSSHPasswd()
	}); ok {
		d0.ClearSSHPasswd()
	}

	if err := provisioner.installCustomDocker(); err != nil {
		return err
	}

	if err := utils.WaitFor(provisioner.dockerDaemonResponding); err != nil {
		return err
	}

	if err := ConfigureAuth(provisioner, authOptions); err != nil {
		return err
	}

	if err := configureSwarm(provisioner, swarmOptions); err != nil {
		return err
	}

	return nil
}

func (provisioner *AiyaraProvisioner) Hostname() (string, error) {
	output, err := provisioner.SSHCommand("hostname")
	if err != nil {
		return "", err
	}

	var so bytes.Buffer
	if _, err := so.ReadFrom(output.Stdout); err != nil {
		return "", err
	}

	return so.String(), nil
}

func (provisioner *AiyaraProvisioner) SetHostname(hostname string) error {
	if _, err := provisioner.SSHCommand(fmt.Sprintf(
		"sudo hostname %s && echo %q | sudo tee /etc/hostname && echo \"127.0.0.1 %s\" | sudo tee -a /etc/hosts",
		hostname,
		hostname,
		hostname,
	)); err != nil {
		return err
	}

	return nil
}

func (provisioner *AiyaraProvisioner) GetDockerOptionsDir() string {
	return "/etc/docker"
}

func (provisioner *AiyaraProvisioner) SSHCommand(args string) (ssh.Output, error) {
	return drivers.RunSSHCommandFromDriver(provisioner.Driver, args)
}

func (provisioner *AiyaraProvisioner) CompatibleWithHost() bool {
	id := provisioner.OsReleaseInfo.Id
	return id == "debian"
}

func (provisioner *AiyaraProvisioner) SetOsReleaseInfo(info *OsRelease) {
	provisioner.OsReleaseInfo = info
}

func (provisioner *AiyaraProvisioner) GenerateDockerOptions(dockerPort int, authOptions auth.AuthOptions) (*DockerOptions, error) {
	defaultDaemonOpts := getDefaultDaemonOpts(provisioner.Driver.DriverName(), authOptions)
	aiyaraOpts := fmt.Sprintf("--label=architecture=%s", "arm")
	daemonOpts := fmt.Sprintf("--host=unix:///var/run/docker.sock --host=tcp://0.0.0.0:%d", dockerPort)
	daemonOptsDir := "/etc/default/docker"
	opts := fmt.Sprintf("%s %s %s", defaultDaemonOpts, aiyaraOpts, daemonOpts)
	daemonCfg := fmt.Sprintf("export DOCKER_OPTS=\\\"%s\\\"", opts)
	return &DockerOptions{
		EngineOptions:     daemonCfg,
		EngineOptionsPath: daemonOptsDir,
	}, nil
}

func (provisioner *AiyaraProvisioner) GetDriver() drivers.Driver {
	return provisioner.Driver
}
