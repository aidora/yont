package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"

	"github.com/docker/machine/libmachine"
	"github.com/docker/machine/libmachine/auth"
	"github.com/docker/machine/libmachine/engine"
	"github.com/docker/machine/libmachine/swarm"
	"github.com/docker/machine/utils"
)

var extSharedCreateFlags = append(sharedCreateFlags, cli.IntFlag{
	Name:  "machine-num",
	Usage: "Number of machines to create",
	Value: 1,
})

type ExtensibleDriverOptions struct {
	c   *cli.Context
	ext map[string]string
}

func (o *ExtensibleDriverOptions) String(key string) string {
	if value, ok := o.ext[key]; ok {
		return value
	}
	return o.c.String(key)
}

func (o *ExtensibleDriverOptions) Int(key string) int {
	return o.c.Int(key)
}

func (o *ExtensibleDriverOptions) Bool(key string) bool {
	return o.c.Bool(key)
}

// Generate takes care of IP generation
func generate(pattern string) []string {
	re, _ := regexp.Compile(`\[(.+):(.+)\]`)
	submatch := re.FindStringSubmatch(pattern)
	if submatch == nil {
		return []string{pattern}
	}

	from, err := strconv.Atoi(submatch[1])
	if err != nil {
		return []string{pattern}
	}
	to, err := strconv.Atoi(submatch[2])
	if err != nil {
		return []string{pattern}
	}

	template := re.ReplaceAllString(pattern, "%d")

	var result []string
	for val := from; val <= to; val++ {
		entry := fmt.Sprintf(template, val)
		result = append(result, entry)
	}

	return result
}

func cmdCreate(c *cli.Context) {
	driver := c.String("driver")
	if driver != "aiyara" {
		cmdCreate0(c)
		return
	}

	baseName := c.Args().First()
	if baseName == "" {
		cli.ShowCommandHelp(c, "create")
		log.Fatal("You must specify a machine name")
	}

	certInfo := getCertPathInfo(c)

	if err := setupCertificates(
		certInfo.CaCertPath,
		certInfo.CaKeyPath,
		certInfo.ClientCertPath,
		certInfo.ClientKeyPath); err != nil {
		log.Fatalf("Error generating certificates: %s", err)
	}

	defaultStore, err := getDefaultStore(
		c.GlobalString("storage-path"),
		certInfo.CaCertPath,
		certInfo.CaKeyPath,
	)
	if err != nil {
		log.Fatal(err)
	}

	mcn, err := newMcn(defaultStore)
	if err != nil {
		log.Fatal(err)
	}

	ipPattern := c.String("aiyara-host-range")
	if ipPattern == "" {
		log.Fatal("Host range must be specified")
	}

	for _, ip := range generate(ipPattern) {
		parts := strings.Split(ip, ".")
		name := fmt.Sprintf("rack-%s-%s-%s", parts[2], baseName, parts[3])

		hostOptions := &libmachine.HostOptions{
			AuthOptions: &auth.AuthOptions{
				CaCertPath:     certInfo.CaCertPath,
				PrivateKeyPath: certInfo.CaKeyPath,
				ClientCertPath: certInfo.ClientCertPath,
				ClientKeyPath:  certInfo.ClientKeyPath,
				ServerCertPath: filepath.Join(utils.GetMachineDir(), name, "server.pem"),
				ServerKeyPath:  filepath.Join(utils.GetMachineDir(), name, "server-key.pem"),
			},
			EngineOptions: &engine.EngineOptions{},
			SwarmOptions: &swarm.SwarmOptions{
				IsSwarm:   c.Bool("swarm"),
				Master:    c.Bool("swarm-master"),
				Discovery: c.String("swarm-discovery"),
				Address:   c.String("swarm-addr"),
				Host:      c.String("swarm-host"),
			},
		}

		ext := map[string]string{"aiyara-host-ip": ip}
		_, err := mcn.Create(name, driver, hostOptions, &ExtensibleDriverOptions{c, ext})
		if err != nil {
			log.Errorf("Error creating machine: %s", err)
			log.Warn("You will want to check the provider to make sure the machine and associated resources were properly removed.")
			log.Fatal("Error creating machine")
		}

		info := ""
		userShell := filepath.Base(os.Getenv("SHELL"))

		switch userShell {
		case "fish":
			info = fmt.Sprintf("%s env %s | source", c.App.Name, name)
		default:
			info = fmt.Sprintf(`eval "$(%s env %s)"`, c.App.Name, name)
		}

		log.Infof("%q has been created and is now the active machine.", name)

		if info != "" {
			log.Infof("To point your Docker client at it, run this in your shell: %s", info)
		}

	} // for

}
