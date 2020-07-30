/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=core,resources=namespaces;resourcequotas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=namespaces/status,verbs=get;update;patch

// NamespaceReconciler reconciles a Namespace object
type NamespaceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *NamespaceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	logger := r.Log.WithValues("namespace", req.NamespacedName)

	namespace := corev1.Namespace{}
	if err := r.Client.Get(ctx, req.NamespacedName, &namespace); err != nil {
		logger.Info("unable to get namespace", "error", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if resourceQuotaShouldBePresent(&namespace) {
		logger.Info("label 'project' was set")
	} else {
		logger.Info("label 'project' is not set, ending reconciliation")
		return ctrl.Result{}, nil
	}

	// Create resourceQuota "project-quota" and ignore err existAlready
	quotaDefault := newDefaultResourceQuota(req.Name, namespace.ObjectMeta.Labels["project"])

	if err := r.Client.Create(ctx, &quotaDefault); err != nil {
		if errors.IsAlreadyExists(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to get project")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func newDefaultResourceQuota(namespaceName string, projectName string) corev1.ResourceQuota {

	quotaDefault := corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-quota",
			Namespace: namespaceName,
			Labels: map[string]string{
				"project": projectName,
			},
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{
				corev1.ResourceCPU:    *resource.NewQuantity(0, resource.DecimalSI),
				corev1.ResourceMemory: *resource.NewQuantity(0, resource.BinarySI),
			},
		},
	}

	return quotaDefault
}

func resourceQuotaShouldBePresent(namespace *corev1.Namespace) bool {
	//	logger.Info("namespace is terminating, ending reconciliation")
	//		return ctrl.Result{}, nil

	_, ok := namespace.Labels["project"]

	return namespace.DeletionTimestamp.IsZero() && ok
}

func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	eventHandler := &handler.EnqueueRequestsFromMapFunc{ToRequests: handler.ToRequestsFunc(r.namespaceMapFn)}
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		Watches(&source.Kind{Type: &corev1.Namespace{}}, eventHandler).Named("Namespace").
		Complete(r)
}

func (r *NamespaceReconciler) namespaceMapFn(handler.MapObject) []reconcile.Request {
	ctx := context.Background()

	NamespaceList := &corev1.NamespaceList{}
	if err := r.List(ctx, NamespaceList); err != nil {
		return []reconcile.Request{}
	}
	requests := make([]reconcile.Request, 0, len(NamespaceList.Items))
	for _, namespace := range NamespaceList.Items {
		requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: namespace.GetNamespace(), Name: namespace.GetName()}})
	}
	return requests
}
