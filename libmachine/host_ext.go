package libmachine

import (
	"github.com/docker/machine/ssh"
)

func (h *Host) RunSSHCommand(command string) (ssh.Output, error) {
	var output ssh.Output

	addr, err := h.Driver.GetSSHHostname()
	if err != nil {
		return output, err
	}

	port, err := h.Driver.GetSSHPort()
	if err != nil {
		return output, err
	}

	auth := &ssh.Auth{}
	if driver, ok := h.Driver.(interface {
		GetSSHPasswd() string
	}); ok {
		passwd := driver.GetSSHPasswd()
		if passwd == "" {
			auth.Keys = []string{h.Driver.GetSSHKeyPath()}
		} else {
			auth.Passwords = []string{passwd}
		}
	} else {
		auth.Keys = []string{h.Driver.GetSSHKeyPath()}
	}

	client, err := ssh.NewClient(h.Driver.GetSSHUsername(), addr, port, auth)

	return client.Run(command)
}
