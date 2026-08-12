// Command grpccerts creates an ephemeral local mTLS trust bundle for Platform.
// Production deployments must supply certificates issued by their workload CA.
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

func main() {
	output := flag.String("out", "./.run/grpc-certs", "certificate output directory")
	force := flag.Bool("force", false, "replace an existing local trust bundle")
	owner := flag.Int("owner", -1, "set the generated directory and files to this Unix uid")
	flag.Parse()
	if err := generate(*output, *force, *owner); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate(output string, force bool, owner int) error {
	output, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	if info, err := os.Stat(output); err == nil && info.IsDir() {
		entries, readErr := os.ReadDir(output)
		if readErr != nil {
			return readErr
		}
		if len(entries) > 0 && !force {
			return fmt.Errorf("local gRPC certificate directory is not empty: %s", output)
		}
	}
	if err := os.MkdirAll(output, 0o700); err != nil {
		return err
	}
	now := time.Now()
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          serial(),
		Subject:               pkix.Name{CommonName: "LiveShop local workload CA"},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return err
	}
	caCertificate, err := x509.ParseCertificate(caDER)
	if err != nil {
		return err
	}
	if err := writePEM(filepath.Join(output, "ca.pem"), "CERTIFICATE", caDER); err != nil {
		return err
	}
	serverTemplate := &x509.Certificate{
		SerialNumber: serial(),
		Subject:      pkix.Name{CommonName: "platform"},
		NotBefore:    now.Add(-time.Minute),
		NotAfter:     now.Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"platform", "localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	if err := issue(output, "server", serverTemplate, caCertificate, caKey); err != nil {
		return err
	}
	for _, workload := range []struct {
		name     string
		spiffeID string
	}{
		{name: "gateway", spiffeID: "spiffe://liveshop.local/gateway"},
		{name: "release", spiffeID: "spiffe://liveshop.local/module-release-ci"},
	} {
		identity, parseErr := url.Parse(workload.spiffeID)
		if parseErr != nil {
			return parseErr
		}
		template := &x509.Certificate{
			SerialNumber: serial(),
			Subject:      pkix.Name{CommonName: workload.name},
			NotBefore:    now.Add(-time.Minute),
			NotAfter:     now.Add(24 * time.Hour),
			KeyUsage:     x509.KeyUsageDigitalSignature,
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			URIs:         []*url.URL{identity},
		}
		if err := issue(output, workload.name, template, caCertificate, caKey); err != nil {
			return err
		}
	}
	if owner >= 0 {
		if err := os.Chown(output, owner, owner); err != nil {
			return fmt.Errorf("set local gRPC certificate directory owner: %w", err)
		}
		entries, err := os.ReadDir(output)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := os.Chown(filepath.Join(output, entry.Name()), owner, owner); err != nil {
				return fmt.Errorf("set local gRPC certificate owner: %w", err)
			}
		}
	}
	fmt.Printf("local Platform gRPC certificates written to %s\n", output)
	return nil
}

func issue(output, name string, template *x509.Certificate, ca *x509.Certificate, caKey *ecdsa.PrivateKey) error {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	certificateDER, err := x509.CreateCertificate(rand.Reader, template, ca, &key.PublicKey, caKey)
	if err != nil {
		return err
	}
	privateKeyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return err
	}
	if err := writePEM(filepath.Join(output, name+".pem"), "CERTIFICATE", certificateDER); err != nil {
		return err
	}
	return writePEM(filepath.Join(output, name+"-key.pem"), "EC PRIVATE KEY", privateKeyDER)
}

func serial() *big.Int {
	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	value, err := rand.Int(rand.Reader, limit)
	if err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return value
}

func writePEM(path, blockType string, der []byte) error {
	return os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der}), 0o600)
}
