package protocol

import (
	"io"
	"net"
	"os"
)

func isEOF(err error) bool {
	return err == io.EOF
}

func isBrokenPipe(err error) bool {
	if sockerr, ok := err.(*net.OpError); ok {
		if syscallerr, ok := sockerr.Err.(*os.SyscallError); ok {
			if syscallerr.Err.Error() == "broken pipe" {
				return true
			}
		}
	}
	return false
}
