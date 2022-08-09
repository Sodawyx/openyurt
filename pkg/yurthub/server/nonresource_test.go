/**
 * @Author: Yuxuan WU
 * @Description:
 * @File: nonresource_test
 * @Version: 1.0.0
 * @Date: 2022/8/8 17:58
 */

package server

import (
	"bytes"
	"github.com/openyurtio/openyurt/cmd/yurthub/app/config"
	"github.com/openyurtio/openyurt/pkg/yurthub/certificate/hubself"
	"github.com/openyurtio/openyurt/pkg/yurthub/healthchecker"
	"github.com/openyurtio/openyurt/pkg/yurthub/kubernetes/rest"
	"github.com/openyurtio/openyurt/pkg/yurthub/storage/disk"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

var (
	certificatePEM = []byte(`-----BEGIN CERTIFICATE-----
MIICRzCCAfGgAwIBAgIJALMb7ecMIk3MMA0GCSqGSIb3DQEBCwUAMH4xCzAJBgNV
BAYTAkdCMQ8wDQYDVQQIDAZMb25kb24xDzANBgNVBAcMBkxvbmRvbjEYMBYGA1UE
CgwPR2xvYmFsIFNlY3VyaXR5MRYwFAYDVQQLDA1JVCBEZXBhcnRtZW50MRswGQYD
VQQDDBJ0ZXN0LWNlcnRpZmljYXRlLTAwIBcNMTcwNDI2MjMyNjUyWhgPMjExNzA0
MDIyMzI2NTJaMH4xCzAJBgNVBAYTAkdCMQ8wDQYDVQQIDAZMb25kb24xDzANBgNV
BAcMBkxvbmRvbjEYMBYGA1UECgwPR2xvYmFsIFNlY3VyaXR5MRYwFAYDVQQLDA1J
VCBEZXBhcnRtZW50MRswGQYDVQQDDBJ0ZXN0LWNlcnRpZmljYXRlLTAwXDANBgkq
hkiG9w0BAQEFAANLADBIAkEAtBMa7NWpv3BVlKTCPGO/LEsguKqWHBtKzweMY2CV
tAL1rQm913huhxF9w+ai76KQ3MHK5IVnLJjYYA5MzP2H5QIDAQABo1AwTjAdBgNV
HQ4EFgQU22iy8aWkNSxv0nBxFxerfsvnZVMwHwYDVR0jBBgwFoAU22iy8aWkNSxv
0nBxFxerfsvnZVMwDAYDVR0TBAUwAwEB/zANBgkqhkiG9w0BAQsFAANBAEOefGbV
NcHxklaW06w6OBYJPwpIhCVozC1qdxGX1dg8VkEKzjOzjgqVD30m59OFmSlBmHsl
nkVA6wyOSDYBf3o=
-----END CERTIFICATE-----`)
	keyPEM = []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIBUwIBADANBgkqhkiG9w0BAQEFAASCAT0wggE5AgEAAkEAtBMa7NWpv3BVlKTC
PGO/LEsguKqWHBtKzweMY2CVtAL1rQm913huhxF9w+ai76KQ3MHK5IVnLJjYYA5M
zP2H5QIDAQABAkAS9BfXab3OKpK3bIgNNyp+DQJKrZnTJ4Q+OjsqkpXvNltPJosf
G8GsiKu/vAt4HGqI3eU77NvRI+mL4MnHRmXBAiEA3qM4FAtKSRBbcJzPxxLEUSwg
XSCcosCktbkXvpYrS30CIQDPDxgqlwDEJQ0uKuHkZI38/SPWWqfUmkecwlbpXABK
iQIgZX08DA8VfvcA5/Xj1Zjdey9FVY6POLXen6RPiabE97UCICp6eUW7ht+2jjar
e35EltCRCjoejRHTuN9TC0uCoVipAiAXaJIx/Q47vGwiw6Y8KXsNU6y54gTbOSxX
54LzHNk/+Q==
-----END RSA PRIVATE KEY-----`)
	yurthubCon = `apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: temp
    server: https://10.10.10.113:6443
  name: default-cluster
contexts:
- context:
    cluster: default-cluster
    namespace: default
    user: default-auth
  name: default-context
current-context: default-context
kind: Config
preferences: {}
users:
- name: default-auth
  user:
    client-certificate: /tmp/pki/yurthub-current.pem
    client-key: /tmp/pki/yurthub-current.pem
`
	testDir = "/tmp/pki/"
)

func TestNonResourceInfoCache(t *testing.T) {
	remoteServers := map[string]int{"https://10.10.10.113:6443": 2}
	u, _ := url.Parse("https://10.10.10.113:6443")
	fakeHealthchecker := healthchecker.NewFakeChecker(true, remoteServers)
	dStorage, err := disk.NewDiskStorage(testDir)
	defer func() {
		if err := os.RemoveAll(testDir); err != nil {
			t.Errorf("Unable to clean up test directory %q: %v", testDir, err)
		}
	}()

	// store the kubelet ca file
	caFile := filepath.Join(testDir, "ca.crt")
	if err := dStorage.Create("ca.crt", certificatePEM); err != nil {
		t.Fatalf("Unable to create the file %q: %v", caFile, err)
	}

	// store the kubelet-pair.pem file
	pairFile := filepath.Join(testDir, "kubelet-pair.pem")
	pd := bytes.Join([][]byte{certificatePEM, keyPEM}, []byte("\n"))
	if err := dStorage.Create("kubelet-pair.pem", pd); err != nil {
		t.Fatalf("Unable to create the file %q: %v", pairFile, err)
	}

	// store the yurthub-current.pem
	yurthubCurrent := filepath.Join(testDir, "yurthub-current.pem")
	if err := dStorage.Create("yurthub-current.pem", pd); err != nil {
		t.Fatalf("Unable to create the file %q: %v", yurthubCurrent, err)
	}

	// set the YurtHubConfiguration
	cfgNew := &config.YurtHubConfiguration{
		RootDir:               testDir,
		RemoteServers:         []*url.URL{u},
		KubeletRootCAFilePath: caFile,
		KubeletPairFilePath:   pairFile,
	}

	cfgNew.CertMgrMode = "hubself"
	certMgr, err := hubself.NewFakeYurtHubCertManager(testDir, yurthubCon, string(certificatePEM), string(keyPEM))
	if err != nil {
		t.Fatalf("error in cfg %v", err)
	}
	certMgr.Start()
	rcm, err := rest.NewRestConfigManager(cfgNew, certMgr, fakeHealthchecker)
	if err != nil {
		t.Errorf("failed to create RestConfigManager: %v", err)
	}

	rc := rcm.GetRestConfig(true)
	clientSet, err := kubernetes.NewForConfig(rc)

	testcases := map[string]struct {
		Accept      string
		Verb        string
		Path        string
		Code        int
		ContentType string
	}{
		"version info": {
			Accept:      "application/json",
			Verb:        "GET",
			Path:        "/version",
			Code:        http.StatusOK,
			ContentType: "application/json",
		},

		"discovery v1": {
			Accept:      "application/json",
			Verb:        "GET",
			Path:        "/apis/discovery.k8s.io/v1",
			Code:        http.StatusOK,
			ContentType: "application/json",
		},
		"discovery v1beta1": {
			Accept:      "application/json",
			Path:        "/apis/discovery.k8s.io/v1beta1",
			Code:        http.StatusOK,
			ContentType: "application/json",
		},
	}

	for k, tt := range testcases {
		t.Run(k, func(t *testing.T) {
			req, _ := http.NewRequest(tt.Verb, tt.Path, nil)
			if len(tt.Accept) != 0 {
				req.Header.Set("Accept", tt.Accept)
			}

			req.RemoteAddr = "127.0.0.1"

			var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

			})
			wrapNonResourceHandler(handler, cfgNew, clientSet)

		})
	}
}
