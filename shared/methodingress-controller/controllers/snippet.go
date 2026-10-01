package controllers

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"

	v1alpha1 "github.com/manny-ovena/microendpoint-stack/shared/methodingress-controller/api/methodingress/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
)

var safePath = regexp.MustCompile(`^/[A-Za-z0-9/_~.%+-]*$`)

func (r *MethodIngressReconciler) generateSnippet(ctx context.Context, mi *v1alpha1.MethodIngress, ing *networkingv1.Ingress) (string, error) {
	if len(mi.Spec.Rules) == 0 {
		return "", fmt.Errorf("at least one method rule is required")
	}
	grouped := make(map[string]map[string]string)
	for _, rule := range mi.Spec.Rules {
		if !safePath.MatchString(rule.Path) {
			return "", fmt.Errorf("invalid literal path %q", rule.Path)
		}
		switch rule.Method {
		case "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "CONNECT", "TRACE":
		default:
			return "", fmt.Errorf("invalid HTTP method %q", rule.Method)
		}
		if len(validation.IsDNS1035Label(rule.Backend.ServiceName)) != 0 {
			return "", fmt.Errorf("invalid backend service name %q", rule.Backend.ServiceName)
		}
		if rule.Backend.ServicePort < 1 || rule.Backend.ServicePort > 65535 {
			return "", fmt.Errorf("invalid backend service port %d", rule.Backend.ServicePort)
		}
		for _, ingressRule := range ing.Spec.Rules {
			if ingressRule.HTTP == nil {
				continue
			}
			for _, path := range ingressRule.HTTP.Paths {
				if path.Path == rule.Path {
					return "", fmt.Errorf("path %q already exists on referenced Ingress; remove it to avoid duplicate NGINX locations", rule.Path)
				}
			}
		}
		if grouped[rule.Path] == nil {
			grouped[rule.Path] = make(map[string]string)
		}
		if _, exists := grouped[rule.Path][rule.Method]; exists {
			return "", fmt.Errorf("duplicate method/path rule: %s %s", rule.Method, rule.Path)
		}
		var service corev1.Service
		if err := r.Get(ctx, types.NamespacedName{Name: rule.Backend.ServiceName, Namespace: mi.Namespace}, &service); err != nil {
			return "", fmt.Errorf("get backend Service %s: %w", rule.Backend.ServiceName, err)
		}
		if net.ParseIP(service.Spec.ClusterIP) == nil {
			return "", fmt.Errorf("backend Service %s must have a ClusterIP", service.Name)
		}
		found := false
		for _, port := range service.Spec.Ports {
			if port.Port == rule.Backend.ServicePort && (port.Protocol == "" || port.Protocol == corev1.ProtocolTCP) {
				found = true
			}
		}
		if !found {
			return "", fmt.Errorf("backend Service %s does not expose TCP port %d", service.Name, rule.Backend.ServicePort)
		}
		// Resolve through the API rather than assuming a cluster DNS suffix or
		// NGINX resolver. Service watches refresh the snippet if the IP changes.
		grouped[rule.Path][rule.Method] = "http://" + net.JoinHostPort(service.Spec.ClusterIP, strconv.Itoa(int(rule.Backend.ServicePort)))
	}
	paths := make([]string, 0, len(grouped))
	for path := range grouped {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	var snippet strings.Builder
	for _, path := range paths {
		fmt.Fprintf(&snippet, "location = %s {\n  set $method_backend \"\";\n", path)
		methods := make([]string, 0, len(grouped[path]))
		for method := range grouped[path] {
			methods = append(methods, method)
		}
		sort.Strings(methods)
		for _, method := range methods {
			fmt.Fprintf(&snippet, "  if ($request_method = %s) {\n    set $method_backend %s;\n  }\n", method, grouped[path][method])
		}
		snippet.WriteString("  if ($method_backend = \"\") { return 405; }\n")
		snippet.WriteString("  proxy_set_header Host $host;\n  proxy_set_header X-Real-IP $remote_addr;\n  proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n  proxy_set_header X-Forwarded-Proto $scheme;\n  proxy_pass $method_backend;\n}\n")
	}
	return snippet.String(), nil
}
