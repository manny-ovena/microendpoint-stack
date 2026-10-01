package controllers

import (
	"context"
	"fmt"
	"time"

	v1alpha1 "github.com/manny-ovena/microendpoint-stack/shared/methodingress-controller/api/methodingress/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	snippetAnnotation = "nginx.ingress.kubernetes.io/server-snippet"
	ownerAnnotation   = "networking.microendpoints.ovena.io/methodingress-owner"
	finalizer         = "networking.microendpoints.ovena.io/ingress-cleanup"
)

type MethodIngressReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *MethodIngressReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	var mi v1alpha1.MethodIngress
	if err := r.Get(ctx, req.NamespacedName, &mi); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !mi.DeletionTimestamp.IsZero() {
		if err := r.cleanup(ctx, &mi, ""); err != nil {
			return ctrl.Result{}, err
		}
		if controllerutil.RemoveFinalizer(&mi, finalizer) {
			return ctrl.Result{}, r.Update(ctx, &mi)
		}
		return ctrl.Result{}, nil
	}
	if controllerutil.AddFinalizer(&mi, finalizer) {
		if err := r.Update(ctx, &mi); err != nil {
			return ctrl.Result{}, err
		}
	}
	if mi.Spec.IngressRef == "" {
		return ctrl.Result{}, fmt.Errorf("ingressRef is required")
	}

	var ing networkingv1.Ingress
	if err := r.Get(ctx, types.NamespacedName{Name: mi.Spec.IngressRef, Namespace: mi.Namespace}, &ing); err != nil {
		return ctrl.Result{}, fmt.Errorf("get referenced Ingress: %w", err)
	}
	owner := string(mi.UID)
	if existing := ing.Annotations[ownerAnnotation]; existing != "" && existing != owner {
		return ctrl.Result{}, fmt.Errorf("Ingress %s is managed by another MethodIngress", ing.Name)
	}
	if ing.Annotations[snippetAnnotation] != "" && ing.Annotations[ownerAnnotation] != owner {
		return ctrl.Result{}, fmt.Errorf("Ingress %s already has an unmanaged server-snippet", ing.Name)
	}
	snippet, err := r.generateSnippet(ctx, &mi, &ing)
	if err != nil {
		logger.Error(err, "cannot generate method routing", "methodingress", mi.Name)
		return ctrl.Result{}, err
	}
	if err := r.cleanup(ctx, &mi, ing.Name); err != nil {
		return ctrl.Result{}, err
	}
	if ing.Annotations[snippetAnnotation] != snippet || ing.Annotations[ownerAnnotation] != owner {
		before := ing.DeepCopy()
		if ing.Annotations == nil {
			ing.Annotations = make(map[string]string)
		}
		ing.Annotations[snippetAnnotation] = snippet
		ing.Annotations[ownerAnnotation] = owner
		if err := r.Patch(ctx, &ing, client.MergeFromWithOptions(before, client.MergeFromWithOptimisticLock{})); err != nil {
			return ctrl.Result{}, fmt.Errorf("patch Ingress routing: %w", err)
		}
		logger.Info("updated method routing", "ingress", ing.Name, "rules", len(mi.Spec.Rules))
	}
	if mi.Status.ObservedGeneration != mi.Generation || mi.Status.LastReconcileTime == "" {
		mi.Status.ObservedGeneration = mi.Generation
		mi.Status.LastReconcileTime = metav1.Now().Format(time.RFC3339)
		if err := r.Status().Update(ctx, &mi); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *MethodIngressReconciler) cleanup(ctx context.Context, mi *v1alpha1.MethodIngress, keep string) error {
	var ingresses networkingv1.IngressList
	if err := r.List(ctx, &ingresses, client.InNamespace(mi.Namespace)); err != nil {
		return err
	}
	for i := range ingresses.Items {
		ing := &ingresses.Items[i]
		if ing.Name == keep || ing.Annotations[ownerAnnotation] != string(mi.UID) {
			continue
		}
		before := ing.DeepCopy()
		delete(ing.Annotations, snippetAnnotation)
		delete(ing.Annotations, ownerAnnotation)
		if err := r.Patch(ctx, ing, client.MergeFromWithOptions(before, client.MergeFromWithOptimisticLock{})); err != nil {
			return err
		}
	}
	return nil
}

func (r *MethodIngressReconciler) enqueueNamespace(ctx context.Context, obj client.Object) []reconcile.Request {
	var resources v1alpha1.MethodIngressList
	if err := r.List(ctx, &resources, client.InNamespace(obj.GetNamespace())); err != nil {
		log.FromContext(ctx).Error(err, "list MethodIngress resources for dependency change")
		return nil
	}
	requests := make([]reconcile.Request, 0, len(resources.Items))
	for _, mi := range resources.Items {
		requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&mi)})
	}
	return requests
}

func (r *MethodIngressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.MethodIngress{}).
		Watches(&networkingv1.Ingress{}, handler.EnqueueRequestsFromMapFunc(r.enqueueNamespace)).
		Watches(&corev1.Service{}, handler.EnqueueRequestsFromMapFunc(r.enqueueNamespace)).
		Complete(r)
}
