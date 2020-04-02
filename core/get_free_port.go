package core

import (
	"fmt"
	"net"

	"golang.org/x/exp/rand"
)

func GetFreePort() int32 {
	for {
		port := 49152 + rand.Int31n(16383)
		listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
		if err != nil {
			continue
		}
		err = listener.Close()
		if err != nil {
			continue
		}
		return port
	}
}
