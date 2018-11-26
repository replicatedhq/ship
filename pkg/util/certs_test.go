package util

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/cloudflare/cfssl/csr"
	"github.com/stretchr/testify/require"
)

func Test_makeKeyRequest(t *testing.T) {

	tests := []struct {
		name     string
		certKind string
		want     csr.BasicKeyRequest
		wantErr  bool
	}{
		{
			name:     "empty kind",
			certKind: "",
			want: csr.BasicKeyRequest{
				A: "rsa",
				S: 2048,
			},
		},
		{
			name:     "rsa",
			certKind: "rsa-2567",
			want: csr.BasicKeyRequest{
				A: "rsa",
				S: 2567,
			},
		},
		{
			name:     "P256",
			certKind: "P256",
			want: csr.BasicKeyRequest{
				A: "ecdsa",
				S: 256,
			},
		},
		{
			name:     "P384",
			certKind: "P384",
			want: csr.BasicKeyRequest{
				A: "ecdsa",
				S: 384,
			},
		},
		{
			name:     "P521",
			certKind: "P521",
			want: csr.BasicKeyRequest{
				A: "ecdsa",
				S: 521,
			},
		},
		{
			name:     "P224 - not acceptable",
			certKind: "P224",
			wantErr:  true,
		},
		{
			name:     "nonsense",
			certKind: "nonsense",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			got, err := makeKeyRequest(tt.certKind)
			if tt.wantErr {
				req.Error(err)
				return
			}
			req.NoError(err)
			req.Equal(tt.want, got)
		})
	}
}

func TestMakeCA(t *testing.T) {
	tests := []struct {
		name   string
		caKind string
	}{
		{
			name:   "empty kind",
			caKind: "",
		},
		{
			name:   "rsa",
			caKind: "rsa-4096",
		},
		{
			name:   "P256",
			caKind: "P256",
		},
		{
			name:   "P384",
			caKind: "P384",
		},
		{
			name:   "P521",
			caKind: "P521",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			ca, err := MakeCA(tt.caKind)
			req.NoError(err)

			// validate ca was generated properly
			_, err = tls.X509KeyPair([]byte(ca.Cert), []byte(ca.Key))
			req.NoError(err)

			block, _ := pem.Decode([]byte(ca.Cert))
			req.NotEqual(nil, block)
			parsedCA, err := x509.ParseCertificate(block.Bytes)
			req.NoError(err, "parse CA certificate")

			req.True(parsedCA.IsCA, "generated CA must be a CA")
		})
	}
}

func TestMakeCert(t *testing.T) {
	tests := []struct {
		name     string
		caKind   string
		certKind string
		hosts    []string
	}{
		{
			name:     "empty kind",
			caKind:   "",
			certKind: "",
			hosts:    []string{},
		},
		{
			name:     "mixed rsa with hostnames",
			caKind:   "rsa-4096",
			certKind: "rsa-2048",
			hosts:    []string{"example.com", "another.co", "1.2.3.4"},
		},
		{
			name:     "P256",
			caKind:   "P256",
			certKind: "P256",
		},
		{
			name:     "P384",
			caKind:   "P384",
			certKind: "P384",
			hosts:    []string{"first.xyz", "subdomain.another.co", "4.3.2.1"},
		},
		{
			name:     "P521",
			caKind:   "P521",
			certKind: "P521",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			ca, err := MakeCA(tt.caKind)
			req.NoError(err)

			// validate ca was generated properly
			_, err = tls.X509KeyPair([]byte(ca.Cert), []byte(ca.Key))
			req.NoError(err)

			block, _ := pem.Decode([]byte(ca.Cert))
			req.NotEqual(nil, block)
			parsedCA, err := x509.ParseCertificate(block.Bytes)
			req.NoError(err, "parse CA certificate")

			req.True(parsedCA.IsCA, "generated CA must be a CA")

			// make cert
			cert, err := MakeCert(tt.hosts, tt.certKind, ca.Cert, ca.Key)
			req.NoError(err)

			// validate cert was generated properly
			_, err = tls.X509KeyPair([]byte(cert.Cert), []byte(cert.Key))
			req.NoError(err)

			block, _ = pem.Decode([]byte(cert.Cert))
			req.NotEqual(nil, block)
			parsedCert, err := x509.ParseCertificate(block.Bytes)
			req.NoError(err, "parse new cert")

			req.False(parsedCert.IsCA, "generated cert must not be a CA")

			err = parsedCert.CheckSignatureFrom(parsedCA)
			req.NoError(err, "cert must be signed by CA")

			for _, host := range tt.hosts {
				err = parsedCert.VerifyHostname(host)
				req.NoError(err, "hostname %s must be present on cert", host)
			}
		})
	}
}
