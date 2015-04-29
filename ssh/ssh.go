package ssh

import (
	"net"
	"time"

	log "github.com/Sirupsen/logrus"
)

func WaitForTCP(addr string) error {
	for {
		log.Debugf("Connecting to %s", addr)
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			continue
		}
		defer conn.Close()
		conn.SetDeadline(5 * time.Second)
		conn.SetReadDeadline(5 * time.Second)
		log.Debug(".. Trying to read ...")
		if _, err = conn.Read(make([]byte, 1)); err != nil {
			log.Debugf(".. Failed to read: %s", err)
			continue
		}
		break
	}
	return nil
}
