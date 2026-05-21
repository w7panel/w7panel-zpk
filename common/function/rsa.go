package function

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/docker/libtrust"
	"strings"
)

func LoadCertAndKey(certFileContent []byte, keyContent []byte) (pk libtrust.PublicKey, prk libtrust.PrivateKey, sigAlg string, err error) {
	cert, err := tls.X509KeyPair(certFileContent, keyContent)
	if err != nil {
		return
	}
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return
	}
	pk, err = libtrust.FromCryptoPublicKey(x509Cert.PublicKey)
	if err != nil {
		return
	}
	prk, err = libtrust.FromCryptoPrivateKey(cert.PrivateKey)
	_, sigAlg, errStr := prk.Sign(strings.NewReader("dummy"), 0)
	if errStr != nil {
		err = fmt.Errorf("failed to sign: %s", errStr)
		return
	}
	return
}
