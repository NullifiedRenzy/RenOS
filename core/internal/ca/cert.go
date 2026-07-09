package ca

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"time"
)

func SHA256Fingerprint(der []byte) string {
	h := sha256.Sum256(der)
	return fmt.Sprintf("%x", h)
}

type SigningParams struct {
	CommonName   string
	Organization string
	NotAfter     time.Time
}

func (c *CA) SignCSR(csrDER []byte, params SigningParams) ([]byte, error) {
	csr, err := x509.ParseCertificateRequest(csrDER)
	if err != nil {
		return nil, fmt.Errorf("parse csr: %w", err)
	}
	if err := csr.CheckSignature(); err != nil {
		return nil, fmt.Errorf("csr signature invalid: %w", err)
	}

	return c.createCert(csr.PublicKey, csr.Subject.CommonName, params.Organization, params.NotAfter)
}

func (c *CA) CreateDeviceCert(pubKey crypto.PublicKey, params SigningParams) ([]byte, error) {
	return c.createCert(pubKey, params.CommonName, params.Organization, params.NotAfter)
}

func (c *CA) createCert(pubKey crypto.PublicKey, commonName, org string, notAfter time.Time) ([]byte, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("generate serial: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{org},
		},
		NotBefore: time.Now(),
		NotAfter:  notAfter,
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
			x509.ExtKeyUsageServerAuth,
		},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, c.Cert, pubKey, c.Key)
	if err != nil {
		return nil, fmt.Errorf("create device cert: %w", err)
	}

	return certDER, nil
}

func GenerateKey() (*ed25519.PrivateKey, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &priv, nil
}

func GenerateCSR(priv crypto.Signer, commonName string) ([]byte, error) {
	template := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: commonName,
		},
	}

	return x509.CreateCertificateRequest(rand.Reader, template, priv)
}
