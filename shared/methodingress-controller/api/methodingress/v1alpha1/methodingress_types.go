package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type MethodIngress struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MethodIngressSpec   `json:"spec,omitempty"`
	Status MethodIngressStatus `json:"status,omitempty"`
}

// +kubebuilder:object:generate=true
type MethodIngressSpec struct {
	// +kubebuilder:validation:MinLength=1
	IngressRef string `json:"ingressRef"`
	// +kubebuilder:validation:MinItems=1
	Rules []MethodRule `json:"rules"`
}

// +kubebuilder:object:generate=true
type MethodRule struct {
	// +kubebuilder:validation:Pattern=`^/[A-Za-z0-9/_~.%+-]*$`
	Path string `json:"path"`
	// +kubebuilder:validation:Enum=GET;POST;PUT;DELETE;PATCH;HEAD;OPTIONS;CONNECT;TRACE
	Method  string     `json:"method"`
	Backend BackendRef `json:"backend"`
}

// +kubebuilder:object:generate=true
type BackendRef struct {
	// +kubebuilder:validation:MinLength=1
	ServiceName string `json:"serviceName"`
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	ServicePort int32 `json:"servicePort"`
}

// +kubebuilder:object:generate=true
type MethodIngressStatus struct {
	ObservedGeneration int64  `json:"observedGeneration,omitempty"`
	LastReconcileTime  string `json:"lastReconcileTime,omitempty"`
}

// +kubebuilder:object:root=true
type MethodIngressList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MethodIngress `json:"items"`
}
