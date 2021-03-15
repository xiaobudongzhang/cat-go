package cat

import (
	"strconv"
	"strings"
)

type serverAddress struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	HttpPort int    `json:"http_port"`
}

func resolveServerAddresses(router string) (addresses []serverAddress) {
	for _, segment := range strings.Split(router, ";") {
		if len(segment) == 0 {
			continue
		}
		fragments := strings.Split(segment, ":")
		if len(fragments) != 2 {
			logger.Warning("%s isn't a valid server address.", segment)
			continue
		}

		if port, err := strconv.Atoi(fragments[1]); err != nil {
			logger.Warning("%s isn't a valid server address.", segment)
		} else {
			addresses = append(addresses, serverAddress{
				Host: fragments[0],
				Port: port,
			})
		}
	}
	return
}

func compareServerAddress(a, b *serverAddress) bool {
	if a == nil || b == nil {
		return false
	}
	if strings.Compare(a.Host, b.Host) == 0 {
		return a.Port == b.Port
	} else {
		return false
	}
}
