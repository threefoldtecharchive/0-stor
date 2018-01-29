/*
 * Copyright (C) 2017-2018 GIG Technology NV and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package grpc

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseNetworkProtocol(t *testing.T) {
	testCases := []struct {
		address, protocol string
	}{
		{"foo", "unix"},
		{"foo/bar", "unix"},
		{"/foo", "unix"},
		{"/foo/bar", "unix"},
		{"foo:8000", "tcp"},
		{"foo.bar:8000", "tcp"},
		{"a.b.c.d:8000", "tcp"},
		{"foo.bar", "unix"},
		{"127.0.0.1:3030", "tcp"},
	}
	for _, testCase := range testCases {
		protocol := parseNetworkProtocol(testCase.address)
		assert.Equal(t, testCase.protocol, protocol)
	}
}

func genTestCertificate() (key, cert string, err error) {
	const (
		host    = "localhost"
		rsaBits = 4096
	)

	priv, err := rsa.GenerateKey(rand.Reader, rsaBits)

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		err = fmt.Errorf("failed to generate serial number: %s", err)
		return
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"GIG Technology NV"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage: x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
			x509.ExtKeyUsageServerAuth,
		},
		BasicConstraintsValid: true,
	}

	template.DNSNames = append(template.DNSNames, host)
	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign

	derBytes, err := x509.CreateCertificate(
		rand.Reader, &template, &template, publicKey(priv), priv)
	if err != nil {
		err = fmt.Errorf("Failed to create certificate: %s", err)
		return
	}

	var certOut bytes.Buffer
	pem.Encode(&certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	cert = string(certOut.Bytes())

	var keyOut bytes.Buffer
	pem.Encode(&keyOut, pemBlockForKey(priv))
	key = string(keyOut.Bytes())

	return
}

// taken from pkg/crypto/tls/generate_cert.go
func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

// taken from pkg/crypto/tls/generate_cert.go
func pemBlockForKey(priv interface{}) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to marshal ECDSA private key: %v", err)
			os.Exit(2)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
	default:
		return nil
	}
}
