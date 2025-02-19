package tlsconfig

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"os"
	"reflect"
	"runtime"
	"testing"
)

// This is the currently active Let’s Encrypt R3 (RSA 2048, O = Let's Encrypt, CN = R3)
// cross-signed CA Intermediate cert, downloaded from: https://letsencrypt.org/certs/lets-encrypt-r3-cross-signed.pem
// It expires Sep 29 19:21:40 2021 GMT
// download updated versions from https://letsencrypt.org/certificates/
const (
	systemRootTrustedCert = `
-----BEGIN CERTIFICATE-----
MIIEZTCCA02gAwIBAgIQQAF1BIMUpMghjISpDBbN3zANBgkqhkiG9w0BAQsFADA/
MSQwIgYDVQQKExtEaWdpdGFsIFNpZ25hdHVyZSBUcnVzdCBDby4xFzAVBgNVBAMT
DkRTVCBSb290IENBIFgzMB4XDTIwMTAwNzE5MjE0MFoXDTIxMDkyOTE5MjE0MFow
MjELMAkGA1UEBhMCVVMxFjAUBgNVBAoTDUxldCdzIEVuY3J5cHQxCzAJBgNVBAMT
AlIzMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAuwIVKMz2oJTTDxLs
jVWSw/iC8ZmmekKIp10mqrUrucVMsa+Oa/l1yKPXD0eUFFU1V4yeqKI5GfWCPEKp
Tm71O8Mu243AsFzzWTjn7c9p8FoLG77AlCQlh/o3cbMT5xys4Zvv2+Q7RVJFlqnB
U840yFLuta7tj95gcOKlVKu2bQ6XpUA0ayvTvGbrZjR8+muLj1cpmfgwF126cm/7
gcWt0oZYPRfH5wm78Sv3htzB2nFd1EbjzK0lwYi8YGd1ZrPxGPeiXOZT/zqItkel
/xMY6pgJdz+dU/nPAeX1pnAXFK9jpP+Zs5Od3FOnBv5IhR2haa4ldbsTzFID9e1R
oYvbFQIDAQABo4IBaDCCAWQwEgYDVR0TAQH/BAgwBgEB/wIBADAOBgNVHQ8BAf8E
BAMCAYYwSwYIKwYBBQUHAQEEPzA9MDsGCCsGAQUFBzAChi9odHRwOi8vYXBwcy5p
ZGVudHJ1c3QuY29tL3Jvb3RzL2RzdHJvb3RjYXgzLnA3YzAfBgNVHSMEGDAWgBTE
p7Gkeyxx+tvhS5B1/8QVYIWJEDBUBgNVHSAETTBLMAgGBmeBDAECATA/BgsrBgEE
AYLfEwEBATAwMC4GCCsGAQUFBwIBFiJodHRwOi8vY3BzLnJvb3QteDEubGV0c2Vu
Y3J5cHQub3JnMDwGA1UdHwQ1MDMwMaAvoC2GK2h0dHA6Ly9jcmwuaWRlbnRydXN0
LmNvbS9EU1RST09UQ0FYM0NSTC5jcmwwHQYDVR0OBBYEFBQusxe3WFbLrlAJQOYf
r52LFMLGMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjANBgkqhkiG9w0B
AQsFAAOCAQEA2UzgyfWEiDcx27sT4rP8i2tiEmxYt0l+PAK3qB8oYevO4C5z70kH
ejWEHx2taPDY/laBL21/WKZuNTYQHHPD5b1tXgHXbnL7KqC401dk5VvCadTQsvd8
S8MXjohyc9z9/G2948kLjmE6Flh9dDYrVYA9x2O+hEPGOaEOa1eePynBgPayvUfL
qjBstzLhWVQLGAkXXmNs+5ZnPBxzDJOLxhF2JIbeQAcH5H0tZrUlo5ZYyOqA7s9p
O5b85o3AM/OJ+CktFBQtfvBhcJVd9wvlwPsk+uyOy2HI7mNxKKgsBTt375teA2Tw
UdHkhVNcsAKX1H7GNNLOEADksd86wuoXvg==
-----END CERTIFICATE-----
`
	rsaPrivateKeyFile             = "fixtures/key.pem"
	certificateFile               = "fixtures/cert.pem"
	multiCertificateFile          = "fixtures/multi.pem"
	rsaEncryptedPrivateKeyFile    = "fixtures/encrypted_key.pem"
	certificateOfEncryptedKeyFile = "fixtures/cert_of_encrypted_key.pem"
)

// returns the name of a pre-generated, multiple-certificate CA file
// with both RSA and ECDSA certs.
func getMultiCert() string {
	return multiCertificateFile
}

// returns the names of pre-generated key and certificate files.
func getCertAndKey() (string, string) {
	return rsaPrivateKeyFile, certificateFile
}

// returns the names of pre-generated, encrypted private key and
// corresponding certificate file
func getCertAndEncryptedKey() (string, string) {
	return rsaEncryptedPrivateKeyFile, certificateOfEncryptedKeyFile
}

// If the cert files and directory are provided but are invalid, an error is
// returned.
func TestConfigServerTLSFailsIfUnableToLoadCerts(t *testing.T) {
	key, cert := getCertAndKey()
	ca := getMultiCert()

	tempFile, err := ioutil.TempFile("", "cert-test")
	if err != nil {
		t.Fatal("Unable to create temporary empty file")
	}
	defer os.RemoveAll(tempFile.Name())
	tempFile.Close()

	for _, badFile := range []string{"not-a-file", tempFile.Name()} {
		for i := 0; i < 3; i++ {
			files := []string{cert, key, ca}
			files[i] = badFile

			result, err := Server(Options{
				CertFile:   files[0],
				KeyFile:    files[1],
				CAFile:     files[2],
				ClientAuth: tls.VerifyClientCertIfGiven,
			})
			if err == nil || result != nil {
				t.Fatal("Expected a non-real file to error and return a nil TLS config")
			}
		}
	}
}

// If server cert and key are provided and client auth and client CA are not
// set, a tls config with only the server certs will be returned.
func TestConfigServerTLSServerCertsOnly(t *testing.T) {
	key, cert := getCertAndKey()

	keypair, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		t.Fatal("Unable to load the generated cert and key")
	}

	tlsConfig, err := Server(Options{
		CertFile: cert,
		KeyFile:  key,
	})
	if err != nil || tlsConfig == nil {
		t.Fatal("Unable to configure server TLS", err)
	}

	if len(tlsConfig.Certificates) != 1 {
		t.Fatal("Unexpected server certificates")
	}
	if len(tlsConfig.Certificates[0].Certificate) != len(keypair.Certificate) {
		t.Fatal("Unexpected server certificates")
	}
	for i, cert := range tlsConfig.Certificates[0].Certificate {
		if !bytes.Equal(cert, keypair.Certificate[i]) {
			t.Fatal("Unexpected server certificates")
		}
	}

	if !reflect.DeepEqual(tlsConfig.CipherSuites, DefaultServerAcceptedCiphers) {
		t.Fatal("Unexpected server cipher suites")
	}
	if !tlsConfig.PreferServerCipherSuites {
		t.Fatal("Expected server to prefer cipher suites")
	}
	if tlsConfig.MinVersion != tls.VersionTLS12 {
		t.Fatal("Unexpected server TLS version")
	}
}

// If client CA is provided, it will only be used if the client auth is >=
// VerifyClientCertIfGiven
func TestConfigServerTLSClientCANotSetIfClientAuthTooLow(t *testing.T) {
	key, cert := getCertAndKey()
	ca := getMultiCert()

	tlsConfig, err := Server(Options{
		CertFile:   cert,
		KeyFile:    key,
		ClientAuth: tls.RequestClientCert,
		CAFile:     ca,
	})

	if err != nil || tlsConfig == nil {
		t.Fatal("Unable to configure server TLS", err)
	}

	if len(tlsConfig.Certificates) != 1 {
		t.Fatal("Unexpected server certificates")
	}
	if tlsConfig.ClientAuth != tls.RequestClientCert {
		t.Fatal("ClientAuth was not set to what was in the options")
	}
	if tlsConfig.ClientCAs != nil {
		t.Fatalf("Client CAs should never have been set")
	}
}

// If client CA is provided, it will only be used if the client auth is >=
// VerifyClientCertIfGiven
func TestConfigServerTLSClientCASet(t *testing.T) {
	key, cert := getCertAndKey()
	ca := getMultiCert()

	tlsConfig, err := Server(Options{
		CertFile:   cert,
		KeyFile:    key,
		ClientAuth: tls.VerifyClientCertIfGiven,
		CAFile:     ca,
	})

	if err != nil || tlsConfig == nil {
		t.Fatal("Unable to configure server TLS", err)
	}

	if len(tlsConfig.Certificates) != 1 {
		t.Fatal("Unexpected server certificates")
	}
	if tlsConfig.ClientAuth != tls.VerifyClientCertIfGiven {
		t.Fatal("ClientAuth was not set to what was in the options")
	}
	basePool, err := SystemCertPool()
	if err != nil {
		basePool = x509.NewCertPool()
	}
	// because we are not enabling `ExclusiveRootPools`, any root pool will also contain the system roots
	if tlsConfig.ClientCAs == nil || len(tlsConfig.ClientCAs.Subjects()) != len(basePool.Subjects())+2 {
		t.Fatalf("Client CAs were never set correctly")
	}
}

// Exclusive root pools determines whether the CA pool will be a union of the system
// certificate pool and custom certs, or an exclusive or of the custom certs and system pool
func TestConfigServerExclusiveRootPools(t *testing.T) {
	if runtime.GOOS == "windows" {
		// FIXME TestConfigServerExclusiveRootPools is failing on windows:
		// config_test.go:244: Unable to verify certificate 1: x509: certificate signed by unknown authority
		t.Skip("FIXME: failing on Windows")
	}
	key, cert := getCertAndKey()
	ca := getMultiCert()

	caBytes, err := ioutil.ReadFile(ca)
	if err != nil {
		t.Fatal("Unable to read CA certs", err)
	}

	var testCerts []*x509.Certificate
	for _, pemBytes := range [][]byte{caBytes, []byte(systemRootTrustedCert)} {
		pemBlock, _ := pem.Decode(pemBytes)
		if pemBlock == nil {
			t.Fatal("Malformed certificate")
		}
		cert, err := x509.ParseCertificate(pemBlock.Bytes)
		if err != nil {
			t.Fatal("Unable to parse certificate")
		}
		testCerts = append(testCerts, cert)
	}

	// ExclusiveRootPools not set, so should be able to verify both system-signed certs
	// and custom CA-signed certs
	tlsConfig, err := Server(Options{
		CertFile:   cert,
		KeyFile:    key,
		ClientAuth: tls.VerifyClientCertIfGiven,
		CAFile:     ca,
	})

	if err != nil || tlsConfig == nil {
		t.Fatal("Unable to configure server TLS", err)
	}

	for i, cert := range testCerts {
		if _, err := cert.Verify(x509.VerifyOptions{Roots: tlsConfig.ClientCAs}); err != nil {
			t.Fatalf("Unable to verify certificate %d: %v", i, err)
		}
	}

	// ExclusiveRootPools set and custom CA provided, so system certs should not be verifiable
	// and custom CA-signed certs should be verifiable
	tlsConfig, err = Server(Options{
		CertFile:           cert,
		KeyFile:            key,
		ClientAuth:         tls.VerifyClientCertIfGiven,
		CAFile:             ca,
		ExclusiveRootPools: true,
	})

	if err != nil || tlsConfig == nil {
		t.Fatal("Unable to configure server TLS", err)
	}

	for i, cert := range testCerts {
		_, err := cert.Verify(x509.VerifyOptions{Roots: tlsConfig.ClientCAs})
		switch {
		case i == 0 && err != nil:
			t.Fatal("Unable to verify custom certificate, even though the root pool should have only the custom CA", err)
		case i == 1 && err == nil:
			t.Fatal("Successfully verified system root-signed certificate though the root pool should have only the cusotm CA", err)
		}
	}

	// No CA file provided, system cert should be verifiable only
	tlsConfig, err = Server(Options{
		CertFile: cert,
		KeyFile:  key,
	})

	if err != nil || tlsConfig == nil {
		t.Fatal("Unable to configure server TLS", err)
	}

	for i, cert := range testCerts {
		_, err := cert.Verify(x509.VerifyOptions{Roots: tlsConfig.ClientCAs})
		switch {
		case i == 1 && err != nil:
			t.Fatal("Unable to verify system root-signed certificate, even though the root pool should be the system pool only", err)
		case i == 0 && err == nil:
			t.Fatal("Successfully verified custom certificate though the root pool should be the system pool only", err)
		}
	}
}

// If we provide a modifier to the server's default TLS configuration generator, it
// should be applied accordingly
func TestConfigServerDefaultWithTLSMinimumModifier(t *testing.T) {
	tlsVersions := []uint16{
		tls.VersionTLS11,
		tls.VersionTLS12,
	}

	for _, tlsVersion := range tlsVersions {
		servDefault := ServerDefault(func(c *tls.Config) {
			c.MinVersion = tlsVersion
		})

		if servDefault.MinVersion != tlsVersion {
			t.Fatalf("Unexpected min TLS version for default server TLS config: %d", servDefault.MinVersion)
		}
	}
}

// If we provide a modifier to the client's default TLS configuration generator, it
// should be applied accordingly
func TestConfigClientDefaultWithTLSMinimumModifier(t *testing.T) {
	tlsVersions := []uint16{
		tls.VersionTLS11,
		tls.VersionTLS12,
	}

	for _, tlsVersion := range tlsVersions {
		clientDefault := ClientDefault(func(c *tls.Config) {
			c.MinVersion = tlsVersion
		})

		if clientDefault.MinVersion != tlsVersion {
			t.Fatalf("Unexpected min TLS version for default client TLS config: %d", clientDefault.MinVersion)
		}
	}
}

// If a valid minimum version is specified in the options, the server's
// minimum version should be set accordingly
func TestConfigServerTLSMinVersionIsSetBasedOnOptions(t *testing.T) {
	versions := []uint16{
		tls.VersionTLS12,
	}
	key, cert := getCertAndKey()

	for _, v := range versions {
		tlsConfig, err := Server(Options{
			MinVersion: v,
			CertFile:   cert,
			KeyFile:    key,
		})

		if err != nil || tlsConfig == nil {
			t.Fatal("Unable to configure server TLS", err)
		}

		if tlsConfig.MinVersion != v {
			t.Fatal("Unexpected minimum TLS version: ", tlsConfig.MinVersion)
		}
	}
}

// An error should be returned if the specified minimum version for the server
// is too low, i.e. less than VersionTLS10
func TestConfigServerTLSMinVersionNotSetIfMinVersionIsTooLow(t *testing.T) {
	key, cert := getCertAndKey()

	_, err := Server(Options{
		MinVersion: tls.VersionTLS10,
		CertFile:   cert,
		KeyFile:    key,
	})

	if err == nil {
		t.Fatal("Should have returned an error for minimum version below TLS10")
	}
}

// An error should be returned if an invalid minimum version for the server is
// in the options struct
func TestConfigServerTLSMinVersionNotSetIfMinVersionIsInvalid(t *testing.T) {
	key, cert := getCertAndKey()

	_, err := Server(Options{
		MinVersion: 1,
		CertFile:   cert,
		KeyFile:    key,
	})

	if err == nil {
		t.Fatal("Should have returned error on invalid minimum version option")
	}
}

// The root CA is never set if InsecureSkipBoolean is set to true, but the
// default client options are set
func TestConfigClientTLSNoVerify(t *testing.T) {
	ca := getMultiCert()

	tlsConfig, err := Client(Options{CAFile: ca, InsecureSkipVerify: true})

	if err != nil || tlsConfig == nil {
		t.Fatal("Unable to configure client TLS", err)
	}

	if tlsConfig.RootCAs != nil {
		t.Fatal("Should not have set Root CAs", err)
	}

	if !reflect.DeepEqual(tlsConfig.CipherSuites, clientCipherSuites) {
		t.Fatal("Unexpected client cipher suites")
	}
	if tlsConfig.MinVersion != tls.VersionTLS12 {
		t.Fatal("Unexpected client TLS version")
	}

	if tlsConfig.Certificates != nil {
		t.Fatal("Somehow client certificates were set")
	}
}

// The root CA is never set if InsecureSkipBoolean is set to false and root CA
// is not provided.
func TestConfigClientTLSNoRoot(t *testing.T) {
	tlsConfig, err := Client(Options{})

	if err != nil || tlsConfig == nil {
		t.Fatal("Unable to configure client TLS", err)
	}

	if tlsConfig.RootCAs != nil {
		t.Fatal("Should not have set Root CAs", err)
	}

	if !reflect.DeepEqual(tlsConfig.CipherSuites, clientCipherSuites) {
		t.Fatal("Unexpected client cipher suites")
	}
	if tlsConfig.MinVersion != tls.VersionTLS12 {
		t.Fatal("Unexpected client TLS version")
	}

	if tlsConfig.Certificates != nil {
		t.Fatal("Somehow client certificates were set")
	}
}

// The RootCA is set if the file is provided and InsecureSkipVerify is false
func TestConfigClientTLSRootCAFileWithOneCert(t *testing.T) {
	ca := getMultiCert()

	tlsConfig, err := Client(Options{CAFile: ca})

	if err != nil || tlsConfig == nil {
		t.Fatal("Unable to configure client TLS", err)
	}
	basePool, err := SystemCertPool()
	if err != nil {
		basePool = x509.NewCertPool()
	}
	// because we are not enabling `ExclusiveRootPools`, any root pool will also contain the system roots
	if tlsConfig.RootCAs == nil || len(tlsConfig.RootCAs.Subjects()) != len(basePool.Subjects())+2 {
		t.Fatal("Root CAs not set properly", err)
	}
	if tlsConfig.Certificates != nil {
		t.Fatal("Somehow client certificates were set")
	}
}

// An error is returned if a root CA is provided but the file doesn't exist.
func TestConfigClientTLSNonexistentRootCAFile(t *testing.T) {
	tlsConfig, err := Client(Options{CAFile: "nonexistent"})

	if err == nil || tlsConfig != nil {
		t.Fatal("Should not have been able to configure client TLS", err)
	}
}

// An error is returned if either the client cert or the key are provided
// but invalid or blank.
func TestConfigClientTLSClientCertOrKeyInvalid(t *testing.T) {
	key, cert := getCertAndKey()

	tempFile, err := ioutil.TempFile("", "cert-test")
	if err != nil {
		t.Fatal("Unable to create temporary empty file")
	}
	defer os.Remove(tempFile.Name())
	_ = tempFile.Close()

	for i := 0; i < 2; i++ {
		for _, invalid := range []string{"not-a-file", "", tempFile.Name()} {
			files := []string{cert, key}
			files[i] = invalid

			tlsConfig, err := Client(Options{CertFile: files[0], KeyFile: files[1]})
			if err == nil || tlsConfig != nil {
				t.Fatal("Should not have been able to configure client TLS", err)
			}
		}
	}
}

// The certificate is set if the client cert and client key are provided and
// valid.
func TestConfigClientTLSValidClientCertAndKey(t *testing.T) {
	key, cert := getCertAndKey()

	keypair, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		t.Fatal("Unable to load the generated cert and key")
	}

	tlsConfig, err := Client(Options{CertFile: cert, KeyFile: key})

	if err != nil || tlsConfig == nil {
		t.Fatal("Unable to configure client TLS", err)
	}

	if len(tlsConfig.Certificates) != 1 {
		t.Fatal("Unexpected client certificates")
	}
	if len(tlsConfig.Certificates[0].Certificate) != len(keypair.Certificate) {
		t.Fatal("Unexpected client certificates")
	}
	for i, cert := range tlsConfig.Certificates[0].Certificate {
		if !bytes.Equal(cert, keypair.Certificate[i]) {
			t.Fatal("Unexpected client certificates")
		}
	}

	if tlsConfig.RootCAs != nil {
		t.Fatal("Root CAs should not have been set", err)
	}
}

// The certificate is set if the client cert and encrypted client key are
// provided and valid and passphrase can decrypt the key
func TestConfigClientTLSValidClientCertAndEncryptedKey(t *testing.T) {
	key, cert := getCertAndEncryptedKey()

	tlsConfig, err := Client(Options{
		CertFile:   cert,
		KeyFile:    key,
		Passphrase: "FooBar123",
	})

	if err != nil || tlsConfig == nil {
		t.Fatal("Unable to configure client TLS", err)
	}

	if len(tlsConfig.Certificates) != 1 {
		t.Fatal("Unexpected client certificates")
	}
}

// The certificate is not set if the provided passphrase cannot decrypt
// the encrypted key.
func TestConfigClientTLSNotSetWithInvalidPassphrase(t *testing.T) {
	key, cert := getCertAndEncryptedKey()

	tlsConfig, err := Client(Options{
		CertFile:   cert,
		KeyFile:    key,
		Passphrase: "InvalidPassphrase",
	})

	if !IsErrEncryptedKey(err) || tlsConfig != nil {
		t.Fatal("Expected failure due to incorrect passphrase.")
	}
}

// Exclusive root pools determines whether the CA pool will be a union of the system
// certificate pool and custom certs, or an exclusive or of the custom certs and system pool
func TestConfigClientExclusiveRootPools(t *testing.T) {
	if runtime.GOOS == "windows" {
		// FIXME TestConfigClientExclusiveRootPools is failing on windows:
		// config_test.go:597: Unable to verify certificate 1: x509: certificate signed by unknown authority
		t.Skip("FIXME: failing on Windows")
	}
	ca := getMultiCert()

	caBytes, err := ioutil.ReadFile(ca)
	if err != nil {
		t.Fatal("Unable to read CA certs", err)
	}

	var testCerts []*x509.Certificate
	for _, pemBytes := range [][]byte{caBytes, []byte(systemRootTrustedCert)} {
		pemBlock, _ := pem.Decode(pemBytes)
		if pemBlock == nil {
			t.Fatal("Malformed certificate")
		}
		cert, err := x509.ParseCertificate(pemBlock.Bytes)
		if err != nil {
			t.Fatal("Unable to parse certificate")
		}
		testCerts = append(testCerts, cert)
	}

	// ExclusiveRootPools not set, so should be able to verify both system-signed certs
	// and custom CA-signed certs
	tlsConfig, err := Client(Options{CAFile: ca})

	if err != nil || tlsConfig == nil {
		t.Fatal("Unable to configure client TLS", err)
	}

	for i, cert := range testCerts {
		if _, err := cert.Verify(x509.VerifyOptions{Roots: tlsConfig.RootCAs}); err != nil {
			t.Fatalf("Unable to verify certificate %d: %v", i, err)
		}
	}

	// ExclusiveRootPools set and custom CA provided, so system certs should not be verifiable
	// and custom CA-signed certs should be verifiable
	tlsConfig, err = Client(Options{
		CAFile:             ca,
		ExclusiveRootPools: true,
	})

	if err != nil || tlsConfig == nil {
		t.Fatal("Unable to configure client TLS", err)
	}

	for i, cert := range testCerts {
		_, err := cert.Verify(x509.VerifyOptions{Roots: tlsConfig.RootCAs})
		switch {
		case i == 0 && err != nil:
			t.Fatal("Unable to verify custom certificate, even though the root pool should have only the custom CA", err)
		case i == 1 && err == nil:
			t.Fatal("Successfully verified system root-signed certificate though the root pool should have only the cusotm CA", err)
		}
	}

	// No CA file provided, system cert should be verifiable only
	tlsConfig, err = Client(Options{})

	if err != nil || tlsConfig == nil {
		t.Fatal("Unable to configure client TLS", err)
	}

	for i, cert := range testCerts {
		_, err := cert.Verify(x509.VerifyOptions{Roots: tlsConfig.RootCAs})
		switch {
		case i == 1 && err != nil:
			t.Fatal("Unable to verify system root-signed certificate, even though the root pool should be the system pool only", err)
		case i == 0 && err == nil:
			t.Fatal("Successfully verified custom certificate though the root pool should be the system pool only", err)
		}
	}
}

// If a valid MinVersion is specified in the options, the client's
// minimum version should be set accordingly
func TestConfigClientTLSMinVersionIsSetBasedOnOptions(t *testing.T) {
	key, cert := getCertAndKey()

	tlsConfig, err := Client(Options{
		MinVersion: tls.VersionTLS12,
		CertFile:   cert,
		KeyFile:    key,
	})

	if err != nil || tlsConfig == nil {
		t.Fatal("Unable to configure client TLS", err)
	}

	if tlsConfig.MinVersion != tls.VersionTLS12 {
		t.Fatal("Unexpected minimum TLS version: ", tlsConfig.MinVersion)
	}
}

// An error should be returned if the specified minimum version for the client
// is too low, i.e. less than VersionTLS12
func TestConfigClientTLSMinVersionNotSetIfMinVersionIsTooLow(t *testing.T) {
	key, cert := getCertAndKey()

	_, err := Client(Options{
		MinVersion: tls.VersionTLS11,
		CertFile:   cert,
		KeyFile:    key,
	})

	if err == nil {
		t.Fatal("Should have returned an error for minimum version below TLS12")
	}
}

// An error should be returned if an invalid minimum version for the client is
// in the options struct
func TestConfigClientTLSMinVersionNotSetIfMinVersionIsInvalid(t *testing.T) {
	key, cert := getCertAndKey()

	_, err := Client(Options{
		MinVersion: 1,
		CertFile:   cert,
		KeyFile:    key,
	})

	if err == nil {
		t.Fatal("Should have returned error on invalid minimum version option")
	}
}
