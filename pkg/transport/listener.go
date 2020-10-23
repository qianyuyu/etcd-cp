package transport

import "crypto/tls"

type TLSInfo struct {
	HandShakeFailure func(conn *tls.Conn, err error)
}
