package zbxscr

import (
	"context"
	"fmt"
	"regexp"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// DockerGetContainerIPs gets the IPs for specified docker container
// `name` - is a regular expression for container search by name
func DockerGetContainerIPs(c context.Context, name string) ([]string, error) {

	var ips []string

	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}

	containers, err := cli.ContainerList(c, types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	for _, container := range containers {
		for _, cn := range container.Names {

			// Check container name for regexp match
			// If match obtain all container IPs
			if regexp.MustCompile(name).MatchString(cn[1:]) == true {
				for _, n := range container.NetworkSettings.Networks {
					if n.IPAddress != "" {
						ips = append(ips, n.IPAddress)
					}
				}
				return ips, nil
			}
		}
	}

	return nil, fmt.Errorf("Container not found")
}
