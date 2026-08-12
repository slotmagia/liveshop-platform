package grpcserver

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	platformv1 "github.com/liveshop-platform/contracts/gen/go/platform/v1"
	"github.com/liveshop-platform/contracts/modulemanifest"
	"github.com/liveshop-platform/module-platform/internal/platform/application/platform/logic"
	"github.com/liveshop-platform/module-platform/internal/platform/common/grpcauth"
	platformregistry "github.com/liveshop-platform/module-platform/internal/platform/registry"
	moduleregistry "github.com/liveshop-platform/module-platform/internal/platform/registry/module"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

func TestPlatformGRPCServerPublishesAuthorizedSnapshots(t *testing.T) {
	certificateFiles, clientCredentials := testCertificates(t, "spiffe://liveshop.test/gateway")
	modules := moduleregistry.NewStore()
	manifest := grpcManifest()
	if _, err := modules.Register(context.Background(), manifest); err != nil {
		t.Fatal(err)
	}
	if err := modules.Activate(context.Background(), manifest.Metadata.ID, manifest.Metadata.Version); err != nil {
		t.Fatal(err)
	}
	dependencies := platformregistry.Dependencies{Modules: modules}
	server, err := New(Config{
		Address: "127.0.0.1:0",
		TLS:     certificateFiles,
		Workloads: []grpcauth.Workload{
			{
				SPIFFEID:    "spiffe://liveshop.test/gateway",
				Subject:     "gateway",
				Permissions: []string{"platform.registry.routes.read", "platform.registry.capabilities.read"},
			},
		},
	}, logic.New(dependencies))
	if err != nil {
		t.Fatal(err)
	}
	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- server.Serve()
	}()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Stop(ctx); err != nil {
			t.Errorf("stop gRPC server: %v", err)
		}
		if err := <-serveErrors; err != nil {
			t.Errorf("serve gRPC: %v", err)
		}
	})

	connection, err := grpc.NewClient(server.Address(), grpc.WithTransportCredentials(clientCredentials))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	healthResponse, err := grpc_health_v1.NewHealthClient(connection).Check(ctx, &grpc_health_v1.HealthCheckRequest{
		Service: platformv1.PlatformRegistryService_ServiceDesc.ServiceName,
	})
	if err != nil || healthResponse.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Fatalf("health response=%v err=%v", healthResponse, err)
	}

	client := platformv1.NewPlatformRegistryServiceClient(connection)
	routes, err := client.GetRouteSnapshot(ctx, &platformv1.GetRouteSnapshotRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if routes.GetRevision() != 2 || len(routes.GetRoutes()) != 1 || routes.GetRoutes()[0].GetModuleId() != "catalog" {
		t.Fatalf("unexpected route snapshot: %#v", routes)
	}
	catalog, err := client.GetCapabilityCatalog(ctx, &platformv1.GetCapabilityCatalogRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if catalog.GetRevision() != 2 || len(catalog.GetItems()) != 1 || len(catalog.GetItems()[0].GetReleases()) != 1 {
		t.Fatalf("unexpected capability catalog: %#v", catalog)
	}
	decoded, err := modulemanifest.Decode(catalog.GetItems()[0].GetReleases()[0].GetManifestJson())
	if err != nil || decoded.Metadata.ID != manifest.Metadata.ID {
		t.Fatalf("capability manifest=%#v err=%v", decoded, err)
	}
}

func TestPlatformGRPCServerRejectsUntrustedSPIFFEIdentity(t *testing.T) {
	certificateFiles, clientCredentials := testCertificates(t, "spiffe://liveshop.test/untrusted")
	dependencies := platformregistry.Dependencies{Modules: moduleregistry.NewStore()}
	server, err := New(Config{
		Address: "127.0.0.1:0",
		TLS:     certificateFiles,
		Workloads: []grpcauth.Workload{
			{
				SPIFFEID:    "spiffe://liveshop.test/gateway",
				Subject:     "gateway",
				Permissions: []string{"platform.registry.routes.read"},
			},
		},
	}, logic.New(dependencies))
	if err != nil {
		t.Fatal(err)
	}
	serveErrors := make(chan error, 1)
	go func() { serveErrors <- server.Serve() }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Stop(ctx)
		<-serveErrors
	})
	connection, err := grpc.NewClient(server.Address(), grpc.WithTransportCredentials(clientCredentials))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = platformv1.NewPlatformRegistryServiceClient(connection).GetRouteSnapshot(ctx, &platformv1.GetRouteSnapshotRequest{})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("untrusted SPIFFE error=%v, want PermissionDenied", err)
	}
}

func grpcManifest() modulemanifest.Manifest {
	permission := "catalog.item.read"
	return modulemanifest.Manifest{
		APIVersion: modulemanifest.APIVersion,
		Kind:       "ModuleRelease",
		Metadata: modulemanifest.Metadata{
			ID:      "catalog",
			Name:    "Catalog",
			Version: "1.0.0",
		},
		Spec: modulemanifest.Spec{
			Backend: modulemanifest.Backend{
				Service: "catalog",
				Origin:  "http://catalog:8090",
				HTTPRoutes: []modulemanifest.HTTPRoute{
					{
						Surface: "admin",
						Prefix:  "/admin/catalog",
						Operations: []modulemanifest.HTTPOperation{
							{
								ID:                  "catalog.item.list",
								Method:              "GET",
								Path:                "/admin/catalog/items",
								Summary:             "List items",
								Description:         "Lists catalog items.",
								Authentication:      "module-session",
								Idempotency:         "safe",
								RequiredPermissions: []string{permission},
								Responses: []modulemanifest.CapabilityResponse{
									{
										Status:      200,
										Description: "Item list",
									},
								},
							},
						},
					},
				},
			},
			Permissions: []modulemanifest.PermissionDefinition{
				{
					Code:     permission,
					Name:     "Read items",
					Resource: "catalog.item",
					Action:   "read",
				},
			},
		},
	}
}

func testCertificates(t *testing.T, clientSPIFFEID string) (TLSConfig, credentials.TransportCredentials) {
	t.Helper()
	now := time.Now()
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "LiveShop test CA"},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	caCertificate, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatal(err)
	}
	serverCertificate, serverKey := signedCertificate(t, caCertificate, caKey, &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    now.Add(-time.Minute),
		NotAfter:     now.Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	})
	spiffeURI, err := url.Parse(clientSPIFFEID)
	if err != nil {
		t.Fatal(err)
	}
	clientCertificate, clientKey := signedCertificate(t, caCertificate, caKey, &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject:      pkix.Name{CommonName: "test workload"},
		NotBefore:    now.Add(-time.Minute),
		NotAfter:     now.Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		URIs:         []*url.URL{spiffeURI},
	})
	directory := t.TempDir()
	caFile := filepath.Join(directory, "ca.pem")
	serverCertificateFile := filepath.Join(directory, "server.pem")
	serverKeyFile := filepath.Join(directory, "server-key.pem")
	writePEM(t, caFile, "CERTIFICATE", caDER)
	writePEM(t, serverCertificateFile, "CERTIFICATE", serverCertificate.Certificate[0])
	writeECKey(t, serverKeyFile, serverKey)
	roots := x509.NewCertPool()
	roots.AddCert(caCertificate)
	return TLSConfig{
			CertificateFile: serverCertificateFile,
			PrivateKeyFile:  serverKeyFile,
			ClientCAFile:    caFile,
		}, credentials.NewTLS(&tls.Config{
			MinVersion:   tls.VersionTLS13,
			RootCAs:      roots,
			Certificates: []tls.Certificate{clientCertificateWithKey(t, clientCertificate, clientKey)},
			ServerName:   "localhost",
		})
}

func signedCertificate(t *testing.T, ca *x509.Certificate, caKey *ecdsa.PrivateKey, template *x509.Certificate) (tls.Certificate, *ecdsa.PrivateKey) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.CreateCertificate(rand.Reader, template, ca, &key.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}}, key
}

func clientCertificateWithKey(t *testing.T, certificate tls.Certificate, key *ecdsa.PrivateKey) tls.Certificate {
	t.Helper()
	certificate.PrivateKey = key
	return certificate
}

func writePEM(t *testing.T, path, blockType string, der []byte) {
	t.Helper()
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeECKey(t *testing.T, path string, key *ecdsa.PrivateKey) {
	t.Helper()
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	writePEM(t, path, "EC PRIVATE KEY", der)
}
