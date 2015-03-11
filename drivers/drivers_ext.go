package drivers

import (
	"github.com/docker/machine/ssh"
)

func RunSSHCommandFromDriver(d Driver, args string) (ssh.Output, error) {
	var output ssh.Output

	host, err := d.GetSSHHostname()
	if err != nil {
		return output, err
	}

	port, err := d.GetSSHPort()
	if err != nil {
		return output, err
	}

	user := d.GetSSHUsername()

	keyPath := d.GetSSHKeyPath()

	auth := &ssh.Auth{}

	if d0, ok := d.(interface {
		GetSSHPasswd() string
	}); ok {
		passwd := d0.GetSSHPasswd()
		if passwd != "" {
			auth.Passwords = []string{passwd}
		} else {
			auth.Keys = []string{keyPath}
		}
	} else {
		auth.Keys = []string{keyPath}
	}

	client, err := ssh.NewClient(user, host, port, auth)
	if err != nil {
		return output, err
	}

	return client.Run(args)
}

/*
func getSSHCommandWithSSHPassFromDriver(d Driver, args ...string) (*exec.Cmd, error) {
	host, err := d.GetSSHHostname()
	if err != nil {
		return nil, err
	}

	port, err := d.GetSSHPort()
	if err != nil {
		return nil, err
	}

	user := d.GetSSHUsername()
	passwd := ""
	// this is the hack to make it return a password rather than key path
	if d0, ok := d.(interface {
		GetSSHPasswd() string
	}); ok {
		passwd = d0.GetSSHPasswd()
		if passwd == "" {
			keyPath := d.GetSSHKeyPath()
			return ssh.GetSSHCommand(host, port, user, keyPath, args...), nil
		}
	}

	return ssh.GetSSHCommandWithSSHPass(host, port, user, passwd, args...), nil
}
*/
