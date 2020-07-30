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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	projectv1 "project/api/v1"
)

// ProjectReconciler reconciles a Project object
type ProjectReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=project.my.domain,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=project.my.domain,resources=projects/status,verbs=get;update;patch

func (r *ProjectReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	logger := r.Log.WithValues("project", req.NamespacedName)
	project := &projectv1.Project{}
	namespaces := corev1.NamespaceList{}

	if err := r.Client.Get(ctx, req.NamespacedName, project); err != nil {
		logger.Error(err, "unable to fetch Project")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.List(ctx, &namespaces); err != nil {
		logger.Error(err, "unable to list namespaces")
		return ctrl.Result{}, err
	}

	updateProjectStatus(namespaces, project)

	if err := r.Client.Status().Update(ctx, project); err != nil {
		logger.Error(err, "unable to update Project Status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func updateProjectStatus(namespaces corev1.NamespaceList, project *projectv1.Project) {
	tmp := make([]string, 0, len(namespaces.Items))
	for _, namespace := range namespaces.Items {
		labels := namespace.Labels
		// log.Printf("labels: %v", labels)
		if labels["project"] == project.Name {
			tmp = append(tmp, namespace.Name)
		}
	}
	project.Status.Namespaces = tmp
}

func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	eventHandler := &handler.EnqueueRequestsFromMapFunc{ToRequests: handler.ToRequestsFunc(r.namespaceMapFn)}
	return ctrl.NewControllerManagedBy(mgr).
		For(&projectv1.Project{}).
		Watches(&source.Kind{Type: &corev1.Namespace{}}, eventHandler).Named("Project").
		Complete(r)
}

func (r *ProjectReconciler) namespaceMapFn(handler.MapObject) []reconcile.Request {
	ctx := context.Background()

	ProjectList := &projectv1.ProjectList{}
	if err := r.List(ctx, ProjectList); err != nil {
		return []reconcile.Request{}
	}
	requests := make([]reconcile.Request, 0, len(ProjectList.Items))
	for _, project := range ProjectList.Items {
		requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: project.GetNamespace(), Name: project.GetName()}})
	}
	return requests
}
