// Package security handles all the TLS stuff.
package security

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// CAKeyPair holds a CA certificate and private key.
type CAKeyPair struct {
	ca   *x509.Certificate
	priv *rsa.PrivateKey
}

// LoadCAKeyPair loads ca.rt and ca.key from $dir.
func LoadCAKeyPair(dir string) (*CAKeyPair, error) {
	p := &CAKeyPair{}
	var err error
	p.ca, err = loadCertificate(filepath.Join(dir, "ca.crt"))
	if err != nil {
		return nil, err
	}
	priv, err := pemFromFile(filepath.Join(dir, "ca.key"), "RSA PRIVATE KEY")
	if err != nil {
		return nil, err
	}
	p.priv, err = x509.ParsePKCS1PrivateKey(priv)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// Name returns the CommonName of the CA. (The circle name.)
func (p *CAKeyPair) Name() string {
	return p.ca.Subject.CommonName
}

// Sign a given public key with this CA and create a certificate for $name.
func (p *CAKeyPair) Sign(pubKey []byte, name string) ([]byte, error) {
	pk, err := x509.ParsePKIXPublicKey(pubKey)
	if err != nil {
		return nil, err
	}

	t := createCertTemplate(false, name)
	cert, err := x509.CreateCertificate(rand.Reader, t, p.ca, pk, p.priv)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert}), nil
}

// CreateToken creates a registration token for $user.
func (p *CAKeyPair) CreateToken(user string) string {
	h := sha1.New()
	h.Write([]byte(user))
	h.Write([]byte{0})
	h.Write(x509.MarshalPKCS1PrivateKey(p.priv))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (p *CAKeyPair) TLSConfigForDiscovery() *tls.Config {
	return getTlsConfig(tlsConfigMaster, p.ca, p.certificate(), "rufs-master")
}

// certificate converts this pair to a *tls.Certificate.
func (p *CAKeyPair) certificate() *tls.Certificate {
	return &tls.Certificate{
		Certificate: [][]byte{p.ca.Raw},
		PrivateKey:  p.priv,
		Leaf:        p.ca,
	}
}

func createCertTemplate(isCA bool, name string) *x509.Certificate {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		panic(fmt.Errorf("failed to generate serial number: %s", err))
	}

	cert := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   name,
			Organization: []string{"RUFS"},
		},
		DNSNames:              []string{name},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	if isCA {
		cert.IsCA = true
		cert.KeyUsage |= x509.KeyUsageCertSign
	}
	return cert
}

// NewCA creates a new CA key pair and writes it to disk.
func NewCA(dir string, name string) error {
	keyfn := filepath.Join(dir, "ca.key")
	t := createCertTemplate(true, name)
	var err error
	priv, err := createKeyPair(keyfn)
	if err != nil {
		return err
	}
	pub := &priv.PublicKey
	ca, err := x509.CreateCertificate(rand.Reader, t, t, pub, priv)
	if err != nil {
		os.Remove(keyfn)
		return err
	}

	if err := pemToFile(filepath.Join(dir, "ca.crt"), "CERTIFICATE", ca, 0644); err != nil {
		os.Remove(keyfn)
		return err
	}
	return nil
}

func loadCertificate(fn string) (*x509.Certificate, error) {
	ca, err := pemFromFile(fn, "CERTIFICATE")
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(ca)
}

func createKeyPair(fn string) (*rsa.PrivateKey, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	mp := x509.MarshalPKCS1PrivateKey(priv)

	if err := pemToFile(fn, "RSA PRIVATE KEY", mp, 0600); err != nil {
		return nil, err
	}

	return priv, nil
}

func serializePubKey(priv *rsa.PrivateKey) ([]byte, error) {
	return x509.MarshalPKIXPublicKey(&priv.PublicKey)
}

// StoreNewKeyPair writes the private key to $privFile and returns the pubkey.
func StoreNewKeyPair(privFile string) ([]byte, error) {
	priv, err := createKeyPair(privFile)
	if err != nil {
		return nil, err
	}
	return serializePubKey(priv)
}

func pemToFile(fn, pemType string, data []byte, mode os.FileMode) error {
	fh, err := os.OpenFile(fn, os.O_WRONLY|os.O_CREATE|os.O_EXCL|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if err := pem.Encode(fh, &pem.Block{Type: pemType, Bytes: data}); err != nil {
		fh.Close()
		os.Remove(fn)
		return err
	}
	if err := fh.Close(); err != nil {
		os.Remove(fn)
		return err
	}
	return nil
}

func pemFromFile(fn, pemType string) ([]byte, error) {
	data, err := ioutil.ReadFile(fn)
	if err != nil {
		return nil, fmt.Errorf("error when reading %q: %v", fn, err)
	}
	pem, _ := pem.Decode(data)
	if pem == nil {
		return nil, fmt.Errorf("error when reading %q: no PEM block found", fn)
	}
	if pem.Type != pemType {
		return nil, fmt.Errorf("error when reading %q: expected PEM type %q, found %q", fn, pemType, pem.Type)
	}
	return pem.Bytes, nil
}

type tlsConfigType int

const (
	tlsConfigMaster tlsConfigType = iota
	tlsConfigMasterClient
	tlsConfigServer
	tlsConfigServerClient
)

func getTlsConfig(mode tlsConfigType, ca *x509.Certificate, cert *tls.Certificate, serverName string) *tls.Config {
	CAs := x509.NewCertPool()
	CAs.AddCert(ca)
	cfg := &tls.Config{
		RootCAs:    CAs,
		ClientCAs:  CAs,
		ServerName: serverName,
	}
	if cert != nil {
		cfg.Certificates = []tls.Certificate{*cert}
	}
	switch mode {
	case tlsConfigMaster, tlsConfigMasterClient:
		cfg.ClientAuth = tls.VerifyClientCertIfGiven
		cfg.PreferServerCipherSuites = true
	case tlsConfigServer, tlsConfigServerClient:
		cfg.ClientAuth = tls.RequireAndVerifyClientCert
	}
	return cfg
}

func TLSConfigForRegistration(caFile string) (*tls.Config, error) {
	ca, err := loadCertificate(caFile)
	if err != nil {
		return nil, err
	}
	return getTlsConfig(tlsConfigMasterClient, ca, nil, ca.Subject.CommonName), nil
}

func LoadKeyPair(caFile, crtFile, keyFile string) (*KeyPair, error) {
	ca, err := loadCertificate(caFile)
	if err != nil {
		return nil, err
	}
	crt, err := tls.LoadX509KeyPair(crtFile, keyFile)
	if err != nil {
		return nil, err
	}
	x509Cert, err := x509.ParseCertificate(crt.Certificate[0])
	if err != nil {
		return nil, err
	}
	crt.Leaf = x509Cert
	return &KeyPair{
		ca: ca,
		crt: crt,
	}, nil
}

type KeyPair struct {
	ca *x509.Certificate
	crt tls.Certificate
}

func (p *KeyPair) TLSConfigForMasterClient() *tls.Config {
	return getTlsConfig(tlsConfigMasterClient, p.ca, &p.crt, p.ca.Subject.CommonName)
}

func (p *KeyPair) TLSConfigForServer() *tls.Config {
	return getTlsConfig(tlsConfigServer, p.ca, &p.crt, p.crt.Leaf.Subject.CommonName)
}

func (p *KeyPair) TLSConfigForServerClient(name string) *tls.Config {
	return getTlsConfig(tlsConfigServer, p.ca, &p.crt, name)
}

// PeerFromContext can be called from inside an RPC handler to get the remote peer and circle name.
func PeerFromContext(ctx context.Context) (name string, circle string, err error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		// This should never happen.
		return "", "", status.Error(codes.Unauthenticated, "no Peer attached to context; TLS issue?")
	}
	ti, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return "", "", status.Error(codes.Unauthenticated, "couldn't get TLSInfo; TLS issue?")
	}
	if len(ti.State.PeerCertificates) == 0 {
		return "", "", status.Error(codes.Unauthenticated, "no client certificate given")
	}
	c := ti.State.PeerCertificates[0]
	return c.Subject.CommonName, c.Issuer.CommonName, nil
}
