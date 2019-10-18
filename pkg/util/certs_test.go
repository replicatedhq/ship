package util

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

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

			caDuration, err := TimeToExpire([]byte(ca.Cert))
			req.NoError(err, "calculate time to expire")
			req.True(caDuration > (5*365*24-1)*time.Hour, "ca should expire in at least 5 years")
		})
	}
}

func TestRenewCA(t *testing.T) {
	tests := []struct {
		name    string
		inputCA CAType
	}{
		{
			name: "empty kind",
			inputCA: CAType{
				Cert: `-----BEGIN CERTIFICATE-----
MIIDADCCAeigAwIBAgIUXeGZuN9UjeF8oQEmvY+si5V8K3IwDQYJKoZIhvcNAQEL
BQAwGDEWMBQGA1UEAwwNZ2F0ZWtlZXBlcl9jYTAeFw0xOTEwMTgyMDQwMDBaFw0y
NDEwMTYyMDQwMDBaMBgxFjAUBgNVBAMMDWdhdGVrZWVwZXJfY2EwggEiMA0GCSqG
SIb3DQEBAQUAA4IBDwAwggEKAoIBAQDVUQxjYBhdU02GFmoDk5R6q/i7zKXwxnTQ
I88lA4ehYfyej7uEi/xKLm8oKkYzDz0OU8H6ysE7ySaVFnkA6X5kryFwsn9bxKwx
RcWP5fvMOJpQuPfL6eqTNnGQd2UNyzbsK5tmfgYX2oXxragNx8KttWciXrUXlZn+
toyIHm14jmwvdsxyEYxmR3jaI42IDUHoPBcbVhYd7w37zHwVQ0qTe41/eXLnJfRI
/IKYB3Nu/Nda3W2YVyNu0jooX5rbvpQZvcQSMxQtGSC5dd2C+f8mgcdyWryaAHt2
2X1EHob8D3QcgDj/tIbU/onfYDUmVmc0CL3vpUChphU2MGHyEvhZAgMBAAGjQjBA
MA4GA1UdDwEB/wQEAwIBBjAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBQBcy1F
YnjX9orTbz0Bpw0idCMf1DANBgkqhkiG9w0BAQsFAAOCAQEAn8KAfoZQwHNrFoQD
adlNPXkDb5lFVs/lWpL3RDMAUAFw+cjx/WGjeNs5SMgL5Y4N4tYzwNsjf51bdjCl
WsLm0uql5qBikmrH9u8FHJoCbVFxaqXJS2Ab74QfgmLqkr5HiKWBVUBRt4rI3nO6
23zqUXtiBEpWu+lBYuG40EUE6qd+IfWM7YEtQ2Dn842Iu7hq9U12iIQc03indGVV
vNzM1MXsffhsF1SsJzRkpRn5R1kuAYbWRoyga8BC3I8m6FIr7Bvv0dl+tXad/x1E
ZPuM49d3ZIRBYU1zeCcLDwr7dhUJllUmt4ckPYEBOoCGa7TeczKyG4Ft+QaljNo1
/IGbFA==
-----END CERTIFICATE-----`,
				Key: `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA1VEMY2AYXVNNhhZqA5OUeqv4u8yl8MZ00CPPJQOHoWH8no+7
hIv8Si5vKCpGMw89DlPB+srBO8kmlRZ5AOl+ZK8hcLJ/W8SsMUXFj+X7zDiaULj3
y+nqkzZxkHdlDcs27CubZn4GF9qF8a2oDcfCrbVnIl61F5WZ/raMiB5teI5sL3bM
chGMZkd42iONiA1B6DwXG1YWHe8N+8x8FUNKk3uNf3ly5yX0SPyCmAdzbvzXWt1t
mFcjbtI6KF+a276UGb3EEjMULRkguXXdgvn/JoHHclq8mgB7dtl9RB6G/A90HIA4
/7SG1P6J32A1JlZnNAi976VAoaYVNjBh8hL4WQIDAQABAoIBAQC4J+or+I/QMdRh
iAQp5kRuyvxHFNvFS28ZKXDxIWT7+93c/XUDbt51JDUuVaCY//TT45c5bcT4WiWG
3AnGsc0+Grsh0deFX/rP5s4x9ng0zEDco3K5hc3PHVdZQtno2KEnrlXQW8fi2/J6
vFKy4tu8nzjUQTLRk4OIlAwqjyouwiCu7WfIlK0yNhdvhUILseA2+y3ES2u0GZMk
3Q7tTqRQz1YsCIooNvhZMI6xWgPwO8qCb0wmrJ77QLlxBmPDHp8SjEBg2qp3elxv
0bPKkx+UFeI380NarFzv3OhEHXnpXoIELujZPcYIxpim8pW2yKS9DieBaprbORA3
u9iChSghAoGBANXpV8hpKThdqrakPfqlnroM/wOeVYv/IRsMcPeI6Y/K4u4P6byW
KPpBIoMPuDfsC5oahkx4plKHqKAbsGQ/dy/1nnLFSh9/Dtax12DKvYwDuo3dNw46
AOXtLMIBPfplTRddLpOt6phgux5EwmHWOYcmnLAo/Cf5OAYWvd2xIkJVAoGBAP9J
vZO7K68xnl78KsuFTd8ibvG8eDf0FdUK69SSJzVOflRmi36IOJCV8SWvdEi9nHXr
AT9XDT+rOrO7qAiWUESP8MpHDy8KA91MvPn/wkW6sZ9czxJobwn54S55aQq+VvKz
5PK4AjRJSy/2CZxJO8yiN9zydZqLpyw2liSpoon1AoGBAIpdB9nrI62A8MZ40FpL
PLNNarpVdTI70ZckYgHLPoAzFLw18NN6MYFGFmO+DEOn3A1O8OWP+M1TUGBX6K2/
W4HbFyVXtc1PqzJ2EEFcgmSJmObgWxdJr4EJ+7R1hzhqxAXD0TfW+/KaRw6aHT2Z
itZ/xEQyDoBwtKtDlIZMaEONAoGAco/C9WLPTcV0jqeXBNIDihjHtM+hG2r7ySkn
f7M+yRs6ceG6w8OZrri7CPBdvK7qYbheTPBhz6qlozaZR5E84CfAJOYSmEdkSJFB
VOdDZUtMnnllq5sWCWILfXGag+m61xuHqKyOwKwLg7Bjy7DJlyFM9GgSApKdKKgu
ZLGDcWkCgYBjY/5NwY32IvBEexOIsHLAAnooBWrP2e4kRypPWjT6EuNLKgZOap20
bjuQlSA8ZKeE6ahEev3e9+sF+/MHW/Na8Fnxt2/5WU79lCORrNGd0QkILtfK3fxi
G3dd/SS2uCuC4YRqwnUTODvXC724Tv4YnJ7GYasqBBQqInsJs9aStQ==
-----END RSA PRIVATE KEY-----`,
			},
		},
		{
			name: "rsa 4096",
			inputCA: CAType{
				Cert: `-----BEGIN CERTIFICATE-----
MIIFADCCAuigAwIBAgIUTDWvZgm0Me93CxG/ZFBndBOisZ0wDQYJKoZIhvcNAQEN
BQAwGDEWMBQGA1UEAwwNZ2F0ZWtlZXBlcl9jYTAeFw0xOTEwMTgyMDQwMDBaFw0y
NDEwMTYyMDQwMDBaMBgxFjAUBgNVBAMMDWdhdGVrZWVwZXJfY2EwggIiMA0GCSqG
SIb3DQEBAQUAA4ICDwAwggIKAoICAQCs9H9AtEn9gTkVPr9ae+G0DjjeWEnsjQwZ
VCKYxIXRYl3Z8pLJlguBx3zd7m0JSAXfwm6sArHA8KVr8bUKYwo0u8aWyqMCdzQM
ZWGntVJzJV2Kntn9+AuqcMKByw9YPXOnrZd2c63Zbk54WveuUBnzEzI5Hf0V+ECr
lAc1JWlnBJbs248ZrCLADcPRCSjhK9e5P7OpNI7hfPo0CcEjTmEDJv0Qdp85Vv16
5b3dJC0MICvKOdPt1e0JQ6Yc2a1IcTVOgmGnceTuKmicqlh7WEvNeSODLhuld5cZ
7MWh+pm4GWCi5J/tkiJda3zKJ/aJk4n1FKdcwLdxd1kcdTnof/yJt4pHqkBVdWYI
RtzTnHTZTUGWd7MrJeLfxdJjZyOiRQGxpzSmBO1R0Oow0iOuEYqb6Yfq18aeJhq2
tJS4jyNAnEPmGofZT/1zL+LW2t2l+hHpBV4GP0mAkObBkNZFWaUwoZmvwRCLHmpu
U3NgmCGPXPbHeb2OByBD2xinnl//ojs1QfObtagSHpMpuvSpjeQRE3Q0vDp49wSx
lEXEicIsaXZNfHjrSfsTm2rtDMURHAmtWbe4kVQAJWe+48GOv8G7eTQ72TUAg7iW
6RFTpUcEpithF5LBFKeAUd8BOxT6NH4FDbPtIiRASHbFMDmrs1AZQgB0N3c5k3Ob
GeqbFaCR+wIDAQABo0IwQDAOBgNVHQ8BAf8EBAMCAQYwDwYDVR0TAQH/BAUwAwEB
/zAdBgNVHQ4EFgQUKjOhMbElTYABle+VETDbHacX6zEwDQYJKoZIhvcNAQENBQAD
ggIBADPY3UNtEex9msSEeQ9lY1QpM+JXnZ0NrodmRYE2l7ofOBb0cD4JYp3DcCdy
hcLdQozi9+1Aj/AllzXEKvgMEqxwK3HTwBnf+W3utXRhioZD5GhqEPSzdTZ1hlkT
IgbXls+UGpJjE4CVYq4JOdhCJlqPQGzHVrQeg90XSKQuOc9r/uig0MrNDazA/oZA
wLSuD7iQC3Fj0j0PjvrzRJT0NpdwoDQSaJzbOp+ZzjzAR8c3TQW2c2CCGkbtA4VK
VXX1zvyTxgAvq5wh/70iMwIVnmvagy3olQwhi77V3j107etQ4j4eg/ZsqqU4CG3W
IrkDqzP/TaRkk3FdtTynolnMqcyDtBN3u0IYudXNR6cTHkLjvghnVud1UQ7RGqFQ
oirCUnnUVwGtuQayy4CpSj16S9KrISeqXQhYQWgsWwQPUGRYH4xVNKJuh87pEpo9
DHZOUte0qLosJMIgCuDRyR4R6BafnSQbU+CZgra1kIVXnGj9Bfi9BhIOxrch3n0o
NB7xNeV/X92araJ7UCa2YETK0wVZ2FXpQ9yl97jE4ji0U1W1IIiPkDyeQbJOlwqC
mV23fPzbnvo6h0DhiB63tIS8Ey9kBZGPJyL5IJA1lWVy2lJNNiKhB9MnQvi2L4xc
XzWZlYz11N/OUc8Q5bmGGKZqm+OiusNQ0in1BZInVcjsyFWq
-----END CERTIFICATE-----`,
				Key: `-----BEGIN RSA PRIVATE KEY-----
MIIJJwIBAAKCAgEArPR/QLRJ/YE5FT6/WnvhtA443lhJ7I0MGVQimMSF0WJd2fKS
yZYLgcd83e5tCUgF38JurAKxwPCla/G1CmMKNLvGlsqjAnc0DGVhp7VScyVdip7Z
/fgLqnDCgcsPWD1zp62XdnOt2W5OeFr3rlAZ8xMyOR39FfhAq5QHNSVpZwSW7NuP
GawiwA3D0Qko4SvXuT+zqTSO4Xz6NAnBI05hAyb9EHafOVb9euW93SQtDCAryjnT
7dXtCUOmHNmtSHE1ToJhp3Hk7iponKpYe1hLzXkjgy4bpXeXGezFofqZuBlgouSf
7ZIiXWt8yif2iZOJ9RSnXMC3cXdZHHU56H/8ibeKR6pAVXVmCEbc05x02U1Blnez
KyXi38XSY2cjokUBsac0pgTtUdDqMNIjrhGKm+mH6tfGniYatrSUuI8jQJxD5hqH
2U/9cy/i1trdpfoR6QVeBj9JgJDmwZDWRVmlMKGZr8EQix5qblNzYJghj1z2x3m9
jgcgQ9sYp55f/6I7NUHzm7WoEh6TKbr0qY3kERN0NLw6ePcEsZRFxInCLGl2TXx4
60n7E5tq7QzFERwJrVm3uJFUACVnvuPBjr/Bu3k0O9k1AIO4lukRU6VHBKYrYReS
wRSngFHfATsU+jR+BQ2z7SIkQEh2xTA5q7NQGUIAdDd3OZNzmxnqmxWgkfsCAwEA
AQKCAgAbNAmf37uTh/O2h7wJO1rwuxvuvOxDrJuukDEw3hg+Kr6gPSshUdxVeU8G
iS3VO+LQowBNRc83jaI3LDlRfOpqCO7fYNfq11z0Zi3J9xcUzVe9KecXryAGmt29
FHdBZcj/IqqkEuXRQSxOeeBjJm4ucWKA4VqhTf69/fZ0QYImle43KwGDBDQjCQc3
pb0sTX0Mwhw8DOw8QzAHZ1FdgEJ6AHPlVwMMPcZ4whHu6nW7ZoP8tsPCsNcrkdxa
xVIgBs5fntpFQADGBR2XJqPsIqMpmlgflez7Ragah8c+BvCOqE8uz87nywhksTdb
hJWeZfpY9fqs+BLiYec+NqH5E8hgjqlzw9hFcKORwHVQx4LjPHATbVk40ctIPpwT
nSlTEKtIvQJcjxJwrm/HDbgRqtgNODJaKh2zMDQqbNyEwgH88ncLVTTNz5W7XntE
XFWBfqte+KVscbvKDdMFs32237hhEz8uZQqhLkXBYyRv6as6VGb3T/6If8QNgC9o
TNmB9eNCKmuLdhSZpO1LdckrLm2R0XWFSeAyvT0Xy7ajKq16gH0PwdTXc7iZKq7U
biSCly9GO62zUDQ5ad5t1Sc6SbZPQh4cFhEyytIGsN9IdBoq9ySEuaI6pgp2A0Mv
if/2k2PzJJ5MJVyuMHE0GMqN8VzSPT8ic/2/ZgO2DfKv2sVyYQKCAQEA2IvYyHIN
plM4PUiizZyJOWU2/qrPOYzuW1nuD1gP6CX17LtG5xQRXTOlFIIkEnXNDfYLOA9y
LEbD5n0GOxa8lhEXx4LI4RaKcIMWZ77EtT/0e5/1ZeiqXcMoYXv705swmBSVCPz7
XyT9oc+Y1vpZ6VkFmzKhRGBnXWTTuhSx1io8lXHjEwZVyKoATfluVztG0tRb08E8
DqRiHBzgF300padbjj0T50mcyxqVyiTwbqvN3ihipWDLh+7kQ+Gv/XWvdyauBgXd
U+fgOLh7s0WGIPCWLliQuD93fux+UacdlQnot3geuP/ANgFfIevvqMQi7hODOBcL
QM2yo/As5P6HlQKCAQEAzHd3/TkKGI9s5STmFWKGu0Qv2ngplQNXMVeFgTJiIhcB
KPaAXCjtx3IQOTtutG8bV/RcSMvNW0v8UPMyz9uqjpfKWC6c5AkbtC601jdFspHX
CJcY6qhd9TkqwWXAf2bdFCSaqfHUd986O61wlKuHtu/RyiHPf2INdbuQbu2n0IGf
OdNVDlCuSmnlss1P4uWUiJAuwMzEkicp/4AKqOP7fa9ZRaNr4qRtsThr/4wkXEI4
ne7tEgJD3I6d8B15m3Ct7BCULaa4yvbsD90IVt7Y/KZIOZ6dbg+4N1nBd3+Hs/RL
zrKZLDn4DB2q8tHjezq2M7SGB52wETR1rnwGV9cPTwKCAQB7VqwS/2Nm6N+PiF+y
XQaL+mpog0GktfDNd1twwefNglGglMq9s2BwhYnxNG73VMGGwi2BsMqHDYdnMK7r
2PdxQisZKBTin8QacY/BZ5cC5XqLL4DGms7uuMm3PLciv7Hd7Vs102IZvyf3khar
28x6bIoU67GPEJnPSC6QPllMcqIvPL7phyI1OR8TSo7egJTGYM4svlNGw7pd6NR6
jIYAFGLBkWhUxEjaJjpK+N85KgIIF1iYeZlzw02gnFtxMibO5ukX5R87O0crB2jt
oxvShzYDD87eIsgdMvZ/63+d9Bbo6TIWjRUdrYpR9+B5b721fMewmu996atmVNY9
V/xBAoIBAHZWJqHt40P3roSoaGm0DlpPyopcxWQy/MHX77KooFcujUNR91Rfc87c
2zrkhNv0+hRbnxWaro3KWovXVW8rqXjBrSCASdlI1DniVlMsxi/lbFjSal9Vdpu4
rGAmLdUOiaFg1grJpbiC/8cOSHwjEnb0Ma0VCGynKTcciSlKbrekba0f/Lg+RcFX
rNNhNH0TdnXbTNPVL2ePNyViy8iXujQxyi8duBECLWJGT2slht3GjdIKODcWDISY
HhycUod+HYrkxX3uYkFFy7YarPrqGxeOfXqrrF3Ix0txrSEmNDoYh89nWnNYUZFh
klDa3RezEUS3lGLQBtjOTdXgfiNUms0CggEANiDErC4EYALXTA0LtUl5IQwu8FYi
02WIIzObbuyORPtdvhhO6W6KxxEGIsyxleNRpbOMje+6+diWb1o6NCmJLcU1Zv9N
lbpDHyJj5c7ShrN8Ug0igDS9ZFED8YT63FbVQnkNb/3CXuO3FeIzDeQ4iaECo1i0
0ztD0gY9jS6OrZRZFyRel1GowcvHPdkvV1wQWLZGvqUmhC802JUmq3NmeqfRqIDZ
SOjE3BVb+saqb5jSP0rEgXienjJF0syS9Csft14+7tibQCOfg82OrNfz5mi7iDxJ
V4Lx0AIJKYQcRbMrf92OHZniggpMKWEbyrvc0XxN7qNR7xPv0r7k8FcPCw==
-----END RSA PRIVATE KEY-----`,
			},
		},
		{
			name: "P256",
			inputCA: CAType{
				Cert: `-----BEGIN CERTIFICATE-----
MIIBczCCARqgAwIBAgIUdP612Ajdfl8Q9zXQqs+KozAH4rswCgYIKoZIzj0EAwIw
GDEWMBQGA1UEAwwNZ2F0ZWtlZXBlcl9jYTAeFw0xOTEwMTgyMDQwMDBaFw0yNDEw
MTYyMDQwMDBaMBgxFjAUBgNVBAMMDWdhdGVrZWVwZXJfY2EwWTATBgcqhkjOPQIB
BggqhkjOPQMBBwNCAATLgGvYeIw9RrCSw6DSK0u79h2H/59vtlb3T10elfdqrZpn
M8YU3DJ2Ug0Vn+BIJm0T4PHIVHlCeZj6AeKLNOV5o0IwQDAOBgNVHQ8BAf8EBAMC
AQYwDwYDVR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQUzrZPAi7iJiaqjoVmnVKRE3J/
4vcwCgYIKoZIzj0EAwIDRwAwRAIgOKWLjXsubS20wFg93hEFBz1VReYXByjpYBiZ
2izMjA0CIBU0hUoNCXbXZvQLPjbv0HItKLjz7ZpKlNg5UGTgHGLN
-----END CERTIFICATE-----`,
				Key: `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIH0Hkd/Fc01eMaG5L/fvguiIP6EKLAViTCYT+YwebkNqoAoGCCqGSM49
AwEHoUQDQgAEy4Br2HiMPUawksOg0itLu/Ydh/+fb7ZW909dHpX3aq2aZzPGFNwy
dlINFZ/gSCZtE+DxyFR5QnmY+gHiizTleQ==
-----END EC PRIVATE KEY-----`,
			},
		},
		{
			name: "P384",
			inputCA: CAType{
				Cert: `-----BEGIN CERTIFICATE-----
MIIBsTCCATegAwIBAgIUd6KnDhvuz/efmwQjtEkxtttl/okwCgYIKoZIzj0EAwMw
GDEWMBQGA1UEAwwNZ2F0ZWtlZXBlcl9jYTAeFw0xOTEwMTgyMDQwMDBaFw0yNDEw
MTYyMDQwMDBaMBgxFjAUBgNVBAMMDWdhdGVrZWVwZXJfY2EwdjAQBgcqhkjOPQIB
BgUrgQQAIgNiAASe4A76K/PvwQGuVe6sFVJZdMvflZGVvfX0GEvbL3UMWXOvOxR6
NZRfVFdeX2Vt7mCf+6WMqLxtV15cCrBolXJstE+/L5xn0QCKOmAKnJ+D0TDa4mBh
3qrDPFhQuYO976GjQjBAMA4GA1UdDwEB/wQEAwIBBjAPBgNVHRMBAf8EBTADAQH/
MB0GA1UdDgQWBBRlm/7t8qnnWF86v2BqKtylEexa3DAKBggqhkjOPQQDAwNoADBl
AjEAgGfZa18Wb6fgbTwEAPkfFTA932RwgpUh6a6wwh82KMbYHTMq2tWp9oP4bT2p
No/qAjAYWgZK2M9YyjFNyn7tccV9T2uK5R+PtlVQjncYd8aOQGvcto7ATvJJCP1Q
XfAGt80=
-----END CERTIFICATE-----`,
				Key: `-----BEGIN EC PRIVATE KEY-----
MIGkAgEBBDDMn8BhcufhEp4uCH8M7EksDUMnXg3TGQVXDUhDa89c8IGKneaR90Q0
uKBwm3lhxi6gBwYFK4EEACKhZANiAASe4A76K/PvwQGuVe6sFVJZdMvflZGVvfX0
GEvbL3UMWXOvOxR6NZRfVFdeX2Vt7mCf+6WMqLxtV15cCrBolXJstE+/L5xn0QCK
OmAKnJ+D0TDa4mBh3qrDPFhQuYO976E=
-----END EC PRIVATE KEY-----`,
			},
		},
		{
			name: "P521",
			inputCA: CAType{
				Cert: `-----BEGIN CERTIFICATE-----
MIIB+zCCAV2gAwIBAgIURSKnoMtOKXQwAH6TTssIVC1WEb8wCgYIKoZIzj0EAwQw
GDEWMBQGA1UEAwwNZ2F0ZWtlZXBlcl9jYTAeFw0xOTEwMTgyMDQwMDBaFw0yNDEw
MTYyMDQwMDBaMBgxFjAUBgNVBAMMDWdhdGVrZWVwZXJfY2EwgZswEAYHKoZIzj0C
AQYFK4EEACMDgYYABAHIz+N4TtGLUy+ihSZx2TvoG7b1ASHAOMEH8JnMrr2IdMje
hVQNqOdnmJ6C/tdbi7NLBvxzDDYSI9BHturD/4p/fgCKjzFixj90JnzlYGtsncRS
DrR3Nx6R0i7RPwh3tK1RlFA9boac9LV60kJLcEcACS9E+hPNu4nfkb3ueK6nOODQ
d6NCMEAwDgYDVR0PAQH/BAQDAgEGMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYE
FBGPR7uAFAYIyDvkfCVIzNieIjxvMAoGCCqGSM49BAMEA4GLADCBhwJBXxJg9+J6
26nxTy8JolxJjPCVQ2WE2jos3MwvL795Ho/srpp3rJbg5YZWqeYtedh8AIgqXs2j
i0CIzyx4PoJgM+8CQgE1XRx4hZNZzZm3MrYQ7EN6P3wp79ILweUf+i+rIsxMgAnM
CRSsrWg3vzGpLYyWQhPs/tecpeinjCbj39hcBmn4Cw==
-----END CERTIFICATE-----`,
				Key: `-----BEGIN EC PRIVATE KEY-----
MIHcAgEBBEIB8MTQXMNcKsFWnJyRNNWMKMD4ATNIqP56fSMcDWkOCYJucQbTa/gc
Wy/AKjsRKVXO6xJSenLlX4z26tl2rvnt2eGgBwYFK4EEACOhgYkDgYYABAHIz+N4
TtGLUy+ihSZx2TvoG7b1ASHAOMEH8JnMrr2IdMjehVQNqOdnmJ6C/tdbi7NLBvxz
DDYSI9BHturD/4p/fgCKjzFixj90JnzlYGtsncRSDrR3Nx6R0i7RPwh3tK1RlFA9
boac9LV60kJLcEcACS9E+hPNu4nfkb3ueK6nOODQdw==
-----END EC PRIVATE KEY-----`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			block, _ := pem.Decode([]byte(tt.inputCA.Cert))
			req.NotEqual(nil, block)
			parsedCA, err := x509.ParseCertificate(block.Bytes)
			req.NoError(err, "parse CA certificate")

			req.True(parsedCA.IsCA, "generated CA must be a CA")

			// test CA regeneration
			renewCA, err := RenewCA(tt.inputCA)
			req.NoError(err)

			// validate regenerated CA is still valid
			_, err = tls.X509KeyPair([]byte(renewCA.Cert), []byte(renewCA.Key))
			req.NoError(err)
			renewBlock, _ := pem.Decode([]byte(renewCA.Cert))
			req.NotEqual(nil, renewBlock)
			renewParsedCA, err := x509.ParseCertificate(renewBlock.Bytes)
			req.NoError(err, "parse renewed CA certificate")

			req.True(renewParsedCA.IsCA, "renewed CA must be a CA")

			// new cert should not be the same as old cert
			req.NotEqual(tt.inputCA.Cert, renewCA.Cert)
			// new key should be the same as old key
			req.Equal(tt.inputCA.Key, renewCA.Key)

			// new CA must expire after old CA
			req.True(renewParsedCA.NotAfter.After(parsedCA.NotAfter), "renewed CA must expire after original")
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

			// validate that the cert is valid for two more years
			req.True(parsedCert.NotAfter.Add(-time.Hour * (17520 - 1)).After(time.Now()))
			// and that it is valid now
			req.True(parsedCert.NotBefore.Before(time.Now()))
		})
	}
}
