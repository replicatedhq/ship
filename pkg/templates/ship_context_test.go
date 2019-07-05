package templates

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"

	"github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func makeTestShipCtx(t *testing.T) ShipContext {
	req := require.New(t)

	testLogger := logger.TestLogger{T: t}
	testViper := viper.New()

	mmFs := afero.NewMemMapFs()
	mmAfero := afero.Afero{Fs: mmFs}

	testManager, err := state.NewDisposableManager(&testLogger, mmAfero, testViper)
	req.NoError(err)

	return ShipContext{
		Logger:  &testLogger,
		Manager: testManager,
	}
}

// tests that generated CAs can be referenced again + that more than one CA can be generated
func TestShipContext_makeCa_repeat(t *testing.T) {
	tests := []struct {
		name         string
		caName       string
		secondCaName string
		caType       string
	}{
		{
			name:         "default RSA",
			caName:       "rsa_test",
			secondCaName: "another_name",
			caType:       "rsa-2048",
		},
		{
			name:         "4096 RSA",
			caName:       "rsa_test",
			secondCaName: "another_name",
			caType:       "rsa-4096",
		},
		{
			name:         "P256",
			caName:       "p256_test",
			secondCaName: "another_name",
			caType:       "P256",
		},
		{
			name:         "P384",
			caName:       "p384_test",
			secondCaName: "another_name",
			caType:       "P384",
		},
		{
			name:         "P521",
			caName:       "p521_test",
			secondCaName: "another_name",
			caType:       "P521",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			ctx := makeTestShipCtx(t)

			// cert generation and basic existence testing
			firstKey := ctx.makeCaKey(tt.caName, tt.caType)
			firstKeyDup := ctx.makeCaKey(tt.caName, tt.caType)
			secondKey := ctx.makeCaKey(tt.secondCaName, tt.caType)

			req.NotEqual("", firstKey, "generated keys should not be the empty string")
			req.Equal(firstKey, firstKeyDup, "generated keys did not match")

			req.NotEqual(firstKey, secondKey, "keys with different names should not match")

			firstCert := ctx.getCaCert(tt.caName)
			firstCertDup := ctx.getCaCert(tt.caName)
			secondCert := ctx.getCaCert(tt.secondCaName)

			req.NotEqual("", firstCert, "generated certs should not be the empty string")
			req.Equal(firstCert, firstCertDup, "generated certs did not match")

			req.NotEqual(firstCert, secondCert, "certs with different names should not match")

			// validate certs were generated properly
			_, err := tls.X509KeyPair([]byte(firstCert), []byte(firstKey))
			req.NoError(err)
			_, err = tls.X509KeyPair([]byte(secondCert), []byte(secondKey))
			req.NoError(err)

			block, _ := pem.Decode([]byte(firstCert))
			req.NotEqual(nil, block)
			firstParsedCA, err := x509.ParseCertificate(block.Bytes)
			req.NoError(err, "parse first CA certificate")
			req.True(firstParsedCA.IsCA, "first CA must be a CA")

			block, _ = pem.Decode([]byte(secondCert))
			req.NotEqual(nil, block)
			secondParsedCA, err := x509.ParseCertificate(block.Bytes)
			req.NoError(err, "parse second CA certificate")
			req.True(secondParsedCA.IsCA, "second CA must be a CA")
		})
	}
}

// generates two certs, covering different sets of hosts, with the same CA
// certs must cover the specified domains, be of the desired type, and be signed by the CA
func TestShipContext_makeCert_repeat(t *testing.T) {
	tests := []struct {
		name           string
		firstCertName  string
		secondCertName string
		firstCertType  string
		secondCertType string
		firstHosts     string
		secondHosts    string
	}{
		{
			name:           "default RSA",
			firstCertName:  "firstCert",
			secondCertName: "secondCert",
			firstCertType:  "rsa-2048",
			secondCertType: "rsa-2048",
			firstHosts:     "first.example.com",
			secondHosts:    "second.example.com,www.second.example.com",
		},
		{
			name:           "mixed RSA",
			firstCertName:  "firstCert",
			secondCertName: "secondCert",
			firstCertType:  "rsa-2048",
			secondCertType: "rsa-4096",
			firstHosts:     "first.example.com",
			secondHosts:    "second.example.com,www.second.example.com",
		},
		{
			name:           "RSA and ECDSA",
			firstCertName:  "firstCert",
			secondCertName: "secondCert",
			firstCertType:  "P256",
			secondCertType: "rsa-2048",
			firstHosts:     "first.example.com",
			secondHosts:    "second.example.com,www.second.example.com",
		},
		{
			name:           "default cert types",
			firstCertName:  "firstCert",
			secondCertName: "secondCert",
			firstHosts:     "first.example.com",
			secondHosts:    "second.example.com,www.second.example.com",
		},
		{
			name:           "mixed ECDSA",
			firstCertName:  "firstCert",
			secondCertName: "secondCert",
			firstCertType:  "P384",
			secondCertType: "P521",
			firstHosts:     "first.example.com",
			secondHosts:    "second.example.com,www.second.example.com",
		},
		{
			name:           "ECDSA CA, Second cert RSA",
			firstCertName:  "firstCert",
			secondCertName: "secondCert",
			firstCertType:  "P521",
			secondCertType: "rsa-2048",
			firstHosts:     "first.example.com",
			secondHosts:    "second.example.com,www.second.example.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			ctx := makeTestShipCtx(t)

			// cert generation and basic existence testing
			firstKey := ctx.makeCertKey(tt.firstCertName, "caName", tt.firstHosts, tt.firstCertType)
			firstKeyDup := ctx.makeCertKey(tt.firstCertName, "caName", tt.firstHosts, tt.firstCertType)
			secondKey := ctx.makeCertKey(tt.secondCertName, "caName", tt.secondHosts, tt.secondCertType)

			req.NotEqual("", firstKey, "generated keys should not be the empty string")
			req.Equal(firstKey, firstKeyDup, "generated keys did not match")

			req.NotEqual(firstKey, secondKey, "keys with different names should not match")

			firstCert := ctx.getCert(tt.firstCertName)
			firstCertDup := ctx.getCert(tt.firstCertName)
			secondCert := ctx.getCert(tt.secondCertName)

			req.NotEqual("", firstCert, "generated certs should not be the empty string")
			req.Equal(firstCert, firstCertDup, "generated certs did not match")

			req.NotEqual(firstCert, secondCert, "certs with different names should not match")

			caCert := ctx.getCaCert("caName")
			req.NotEqual("", caCert, "CA cert should not be the empty string")

			// cert validation

			// cert must match key
			_, err := tls.X509KeyPair([]byte(firstCert), []byte(firstKey))
			req.NoError(err)

			// cert must match desired hostnames
			block, _ := pem.Decode([]byte(firstCert))
			req.NotEqual(nil, block)
			firstParsedCert, err := x509.ParseCertificate(block.Bytes)
			req.NoError(err, "parse first certificate")

			for _, host := range strings.Split(tt.firstHosts, ",") {
				err = firstParsedCert.VerifyHostname(host)
				req.NoError(err, "First cert should have hostname %s", host)
			}

			// cert must not be a CA
			req.False(firstParsedCert.IsCA, "First cert must not be CA")

			// second cert must match key
			_, err = tls.X509KeyPair([]byte(secondCert), []byte(secondKey))
			req.NoError(err)

			// second cert must match desired hostnames
			block, _ = pem.Decode([]byte(secondCert))
			req.NotEqual(nil, block)
			secondParsedCert, err := x509.ParseCertificate(block.Bytes)
			req.NoError(err, "parse second certificate")

			for _, host := range strings.Split(tt.secondHosts, ",") {
				err = secondParsedCert.VerifyHostname(host)
				req.NoError(err, "Second cert should have hostname %s", host)
			}

			// second cert must not be a CA
			req.False(secondParsedCert.IsCA, "Second cert must not be CA")

			// validate that certs are signed by the CA
			block, _ = pem.Decode([]byte(caCert))
			req.NotEqual(nil, block)
			parsedCA, err := x509.ParseCertificate(block.Bytes)
			req.NoError(err, "parse CA certificate")
			req.True(parsedCA.IsCA, "CA must be a CA")

			err = firstParsedCert.CheckSignatureFrom(parsedCA)
			req.NoError(err, "first cert must be signed by CA")
			err = secondParsedCert.CheckSignatureFrom(parsedCA)
			req.NoError(err, "second cert must be signed by CA")
		})
	}
}
