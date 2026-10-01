package controllers

import (
	"context"
	"strings"
	"testing"

	v1alpha1 "github.com/manny-ovena/microendpoint-stack/shared/methodingress-controller/api/methodingress/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func fixture(t *testing.T) (*MethodIngressReconciler, *v1alpha1.MethodIngress, *networkingv1.Ingress) {
	t.Helper()
	scheme := runtime.NewScheme()
	for _, register := range []func(*runtime.Scheme) error{v1alpha1.AddToScheme, corev1.AddToScheme, networkingv1.AddToScheme} {
		if err := register(scheme); err != nil {
			t.Fatal(err)
		}
	}
	mi := &v1alpha1.MethodIngress{
		ObjectMeta: metav1.ObjectMeta{Name: "hello", Namespace: "default", UID: "hello-uid", Generation: 1},
		Spec: v1alpha1.MethodIngressSpec{
			IngressRef: "hello-ingress",
			Rules: []v1alpha1.MethodRule{
				{Path: "/hello", Method: "POST", Backend: v1alpha1.BackendRef{ServiceName: "hello-writer", ServicePort: 8080}},
				{Path: "/hello", Method: "GET", Backend: v1alpha1.BackendRef{ServiceName: "hello-service", ServicePort: 80}},
			},
		},
	}
	ing := &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: "hello-ingress", Namespace: "default"}}
	reader := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "hello-service", Namespace: "default"},
		Spec:       corev1.ServiceSpec{ClusterIP: "10.96.0.10", Ports: []corev1.ServicePort{{Port: 80}}},
	}
	writer := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "hello-writer", Namespace: "default"},
		Spec:       corev1.ServiceSpec{ClusterIP: "10.96.0.11", Ports: []corev1.ServicePort{{Port: 8080}}},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&v1alpha1.MethodIngress{}).
		WithObjects(mi, ing, reader, writer).Build()
	return &MethodIngressReconciler{Client: c, Scheme: scheme}, mi, ing
}

func TestGenerateSnippet(t *testing.T) {
	r, mi, ing := fixture(t)
	snippet, err := r.generateSnippet(context.Background(), mi, ing)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"location = /hello {",
		"if ($request_method = GET)",
		"set $method_backend http://10.96.0.10:80;",
		"if ($request_method = POST)",
		"set $method_backend http://10.96.0.11:8080;",
		"return 405;",
		"proxy_pass $method_backend;",
	} {
		if !strings.Contains(snippet, want) {
			t.Errorf("snippet missing %q:\n%s", want, snippet)
		}
	}
	if strings.Count(snippet, "location = /hello") != 1 {
		t.Error("colliding paths must share a single location")
	}
	mi.Spec.Rules[0], mi.Spec.Rules[1] = mi.Spec.Rules[1], mi.Spec.Rules[0]
	reordered, err := r.generateSnippet(context.Background(), mi, ing)
	if err != nil || reordered != snippet {
		t.Errorf("generation must be deterministic: %v", err)
	}
}

func TestGenerateSnippetRejectsInvalidRules(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*v1alpha1.MethodIngress, *networkingv1.Ingress)
	}{
		{"empty rules", func(mi *v1alpha1.MethodIngress, _ *networkingv1.Ingress) { mi.Spec.Rules = nil }},
		{"path injection", func(mi *v1alpha1.MethodIngress, _ *networkingv1.Ingress) {
			mi.Spec.Rules[0].Path = "/hello { return 200; }"
		}},
		{"variable path", func(mi *v1alpha1.MethodIngress, _ *networkingv1.Ingress) { mi.Spec.Rules[0].Path = "/$host" }},
		{"method injection", func(mi *v1alpha1.MethodIngress, _ *networkingv1.Ingress) { mi.Spec.Rules[0].Method = "POST) {}" }},
		{"invalid name", func(mi *v1alpha1.MethodIngress, _ *networkingv1.Ingress) {
			mi.Spec.Rules[0].Backend.ServiceName = "host;return"
		}},
		{"invalid port", func(mi *v1alpha1.MethodIngress, _ *networkingv1.Ingress) {
			mi.Spec.Rules[0].Backend.ServicePort = 65536
		}},
		{"missing service", func(mi *v1alpha1.MethodIngress, _ *networkingv1.Ingress) {
			mi.Spec.Rules[0].Backend.ServiceName = "missing"
		}},
		{"missing service port", func(mi *v1alpha1.MethodIngress, _ *networkingv1.Ingress) { mi.Spec.Rules[0].Backend.ServicePort = 1234 }},
		{"duplicate pair", func(mi *v1alpha1.MethodIngress, _ *networkingv1.Ingress) {
			mi.Spec.Rules = append(mi.Spec.Rules, mi.Spec.Rules[0])
		}},
		{"ingress path collision", func(_ *v1alpha1.MethodIngress, ing *networkingv1.Ingress) {
			ing.Spec.Rules = []networkingv1.IngressRule{{IngressRuleValue: networkingv1.IngressRuleValue{
				HTTP: &networkingv1.HTTPIngressRuleValue{Paths: []networkingv1.HTTPIngressPath{{Path: "/hello"}}},
			}}}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r, mi, ing := fixture(t)
			test.mutate(mi, ing)
			if _, err := r.generateSnippet(context.Background(), mi, ing); err == nil {
				t.Fatal("invalid rule accepted")
			}
		})
	}
}

func TestReconcileUpdatesRepairsAndCleansRouting(t *testing.T) {
	ctx := context.Background()
	r, mi, ing := fixture(t)
	req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(mi)}
	reconcile := func() {
		t.Helper()
		if _, err := r.Reconcile(ctx, req); err != nil {
			t.Fatal(err)
		}
		if err := r.Get(ctx, client.ObjectKeyFromObject(mi), mi); err != nil {
			t.Fatal(err)
		}
		if err := r.Get(ctx, client.ObjectKeyFromObject(ing), ing); err != nil {
			t.Fatal(err)
		}
	}
	reconcile()
	if ing.Annotations[ownerAnnotation] != string(mi.UID) || ing.Annotations[snippetAnnotation] == "" {
		t.Fatal("snippet not applied")
	}
	if mi.Status.ObservedGeneration != 1 || mi.Status.LastReconcileTime == "" {
		t.Fatal("status not recorded")
	}
	version := ing.ResourceVersion
	reconcile()
	if ing.ResourceVersion != version {
		t.Fatal("unchanged reconciliation patched the Ingress")
	}

	mi.Spec.Rules = mi.Spec.Rules[1:]
	mi.Generation++
	if err := r.Update(ctx, mi); err != nil {
		t.Fatal(err)
	}
	reconcile()
	if strings.Contains(ing.Annotations[snippetAnnotation], "POST") {
		t.Fatal("removed rule remains in snippet")
	}

	var svc corev1.Service
	if err := r.Get(ctx, client.ObjectKey{Namespace: "default", Name: "hello-service"}, &svc); err != nil {
		t.Fatal(err)
	}
	svc.Spec.ClusterIP = "10.96.0.12"
	if err := r.Update(ctx, &svc); err != nil {
		t.Fatal(err)
	}
	reconcile()
	if !strings.Contains(ing.Annotations[snippetAnnotation], "10.96.0.12") {
		t.Fatal("Service IP change was not reflected")
	}
	delete(ing.Annotations, snippetAnnotation)
	if err := r.Update(ctx, ing); err != nil {
		t.Fatal(err)
	}
	reconcile()
	if ing.Annotations[snippetAnnotation] == "" {
		t.Fatal("snippet drift was not repaired")
	}
	if err := r.Delete(ctx, mi); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Reconcile(ctx, req); err != nil {
		t.Fatal(err)
	}
	if err := r.Get(ctx, client.ObjectKeyFromObject(ing), ing); err != nil {
		t.Fatal(err)
	}
	if _, exists := ing.Annotations[snippetAnnotation]; exists {
		t.Fatal("snippet remains after deletion")
	}
}

func TestReconcilePreservesUnmanagedSnippet(t *testing.T) {
	r, mi, ing := fixture(t)
	ing.Annotations = map[string]string{snippetAnnotation: "# user snippet"}
	if err := r.Update(context.Background(), ing); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(mi)}); err == nil {
		t.Fatal("unmanaged snippet overwritten")
	}
	if err := r.Get(context.Background(), client.ObjectKeyFromObject(ing), ing); err != nil {
		t.Fatal(err)
	}
	if ing.Annotations[snippetAnnotation] != "# user snippet" {
		t.Fatal("unmanaged snippet changed")
	}
}

func TestReconcileRetargetsIngress(t *testing.T) {
	r, mi, old := fixture(t)
	ctx := context.Background()
	req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(mi)}
	if _, err := r.Reconcile(ctx, req); err != nil {
		t.Fatal(err)
	}
	next := &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: "next", Namespace: "default"}}
	if err := r.Create(ctx, next); err != nil {
		t.Fatal(err)
	}
	if err := r.Get(ctx, client.ObjectKeyFromObject(mi), mi); err != nil {
		t.Fatal(err)
	}
	mi.Spec.IngressRef = next.Name
	if err := r.Update(ctx, mi); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Reconcile(ctx, req); err != nil {
		t.Fatal(err)
	}
	for _, ing := range []*networkingv1.Ingress{old, next} {
		if err := r.Get(ctx, client.ObjectKeyFromObject(ing), ing); err != nil {
			t.Fatal(err)
		}
	}
	if old.Annotations[snippetAnnotation] != "" || next.Annotations[snippetAnnotation] == "" {
		t.Fatal("retargeting did not move the snippet")
	}
}
