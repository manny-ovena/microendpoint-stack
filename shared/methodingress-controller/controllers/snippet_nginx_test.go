package controllers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Run explicitly with NGINX_INTEGRATION=1 to exercise the generated snippet in
// NGINX without requiring a Kubernetes cluster or changing an existing ingress.
func TestSnippetInNGINX(t *testing.T) {
	if os.Getenv("NGINX_INTEGRATION") != "1" {
		t.Skip("set NGINX_INTEGRATION=1 to run the Docker-based NGINX routing test")
	}
	r, mi, ing := fixture(t)
	ctx := context.Background()
	for i := range mi.Spec.Rules {
		rule := &mi.Spec.Rules[i]
		var svc corev1.Service
		if err := r.Get(ctx, client.ObjectKey{Namespace: mi.Namespace, Name: rule.Backend.ServiceName}, &svc); err != nil {
			t.Fatal(err)
		}
		rule.Backend.ServicePort = int32(8081 + i)
		svc.Spec.ClusterIP = "127.0.0.1"
		svc.Spec.Ports[0].Port = rule.Backend.ServicePort
		if err := r.Update(ctx, &svc); err != nil {
			t.Fatal(err)
		}
	}
	snippet, err := r.generateSnippet(ctx, mi, ing)
	if err != nil {
		t.Fatal(err)
	}
	config := fmt.Sprintf(`events {}
http {
  access_log /dev/stdout;
  error_log /dev/stderr;
  server { listen 8081; location / { return 200 "writer $request_method $request_uri\n"; } }
  server { listen 8082; location / { return 200 "reader $request_method $request_uri\n"; } }
  server {
    listen 8080;
    %s
    location / { return 404; }
  }
}
`, snippet)
	configPath := filepath.Join(t.TempDir(), "nginx.conf")
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatal(err)
	}
	docker := func(args ...string) string {
		t.Helper()
		cmdCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		output, err := exec.CommandContext(cmdCtx, "docker", args...).CombinedOutput()
		if err != nil {
			t.Fatalf("docker %v: %v\n%s", args, err, output)
		}
		return strings.TrimSpace(string(output))
	}
	mount := configPath + ":/etc/nginx/nginx.conf:ro"
	docker("run", "--rm", "-v", mount, "nginx:stable", "nginx", "-t")
	id := docker("run", "-d", "--rm", "-p", "127.0.0.1::8080", "-v", mount, "nginx:stable")
	t.Cleanup(func() { docker("stop", id) })
	address := docker("port", id, "8080/tcp")
	httpClient := &http.Client{Timeout: 5 * time.Second}
	url := "http://" + address
	deadline := time.Now().Add(15 * time.Second)
	for {
		resp, err := httpClient.Get(url + "/hello")
		if err == nil {
			resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("NGINX did not become responsive: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	for _, test := range []struct {
		method string
		path   string
		status int
		body   string
	}{
		{"GET", "/hello?user=42", 200, "reader GET /hello?user=42\n"},
		{"POST", "/hello?user=42", 200, "writer POST /hello?user=42\n"},
		{"DELETE", "/hello", 405, ""},
		{"HEAD", "/hello", 405, ""},
		{"GET", "/hello/child", 404, ""},
	} {
		req, err := http.NewRequest(test.method, url+test.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != test.status || (test.body != "" && string(body) != test.body) {
			t.Errorf("%s %s: status=%d body=%q, want status=%d body=%q",
				test.method, test.path, resp.StatusCode, body, test.status, test.body)
		}
	}
}
