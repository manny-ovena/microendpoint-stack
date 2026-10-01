package v1alpha1

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestRegistersListOptions(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.NewParameterCodec(scheme).EncodeParameters(&metav1.ListOptions{}, GroupVersion); err != nil {
		t.Fatalf("watch/list parameters cannot be encoded: %v", err)
	}
}
