package none

import (
	"strings"
)

func (d *Driver) GetIP() (string, error) {
	parts := strings.SplitN(d.URL, "://", 2)
	return strings.SplitN(parts[1], ":", 2)[0], nil
}
