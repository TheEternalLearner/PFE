/*
Copyright 2018 The Kubernetes Authors.
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

package webhook

import (
	"context"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"net/http"
	projectv1 "project/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/validate-v1-resourcequota,mutating=false,failurePolicy=fail,groups="",resources=resourcequotas,verbs=update;delete,versions=v1,name=vresourcequota.kb.io

// resourceQuotaValidator validates ResourceQuotas
type ResourceQuotaValidator struct {
	Client  client.Client
	decoder *admission.Decoder
}

// resourceQuota validator
func (v *ResourceQuotaValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	switch req.Operation {
		case v1beta1.Create:
			return v.validateCreate(ctx, req)
		case v1beta1.Update:
			return v.validateUpdate(ctx, req)
		case v1beta1.Delete:
			return v.validateDelete(ctx, req)
		default:
			return admission.Allowed("No specific logic for" + string(req.Operation) + " operations")
	}
}

func (v *ResourceQuotaValidator) validateCreate(ctx context.Context, req admission.Request) admission.Response {
	namespace := corev1.Namespace{}
	err := v.Client.Get(ctx, client.ObjectKey{Name: req.Namespace}, &namespace)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	project := projectv1.Project{}
	err = v.Client.Get(ctx, client.ObjectKey{Name: namespace.Labels["project"]}, &project)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	quota := corev1.ResourceQuota{}
	oldQuota := &corev1.ResourceQuota{}

	err = v.decoder.Decode(req, &quota)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	err = v.decoder.DecodeRaw(req.OldObject, oldQuota)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	namespaceList := corev1.NamespaceList{}
	err = v.Client.List(ctx, &namespaceList)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	resourceQuotaList, err := v.allResourceQuotasInProject(ctx, project)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return allowOrDenyUpdateOrCreate(project, quota, oldQuota, resourceQuotaList)
}
//TODO resourcequota du projet seulement
func (v *ResourceQuotaValidator) validateUpdate(ctx context.Context, req admission.Request) admission.Response {
	namespace := corev1.Namespace{}
	err := v.Client.Get(ctx, client.ObjectKey{Name: req.Namespace}, &namespace)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	project := projectv1.Project{}
	err = v.Client.Get(ctx, client.ObjectKey{Name: namespace.Labels["project"]}, &project)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}


	quota := corev1.ResourceQuota{}
	oldQuota := &corev1.ResourceQuota{}
	err = v.decoder.Decode(req, &quota)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	err = v.decoder.DecodeRaw(req.OldObject, oldQuota)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	resourceQuotaList, err := v.allResourceQuotasInProject(ctx, project)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return allowOrDenyUpdateOrCreate(project, quota, oldQuota, resourceQuotaList)
}

func (v *ResourceQuotaValidator) validateDelete(ctx context.Context, req admission.Request) admission.Response {
	namespace := corev1.Namespace{}
	err := v.Client.Get(ctx, client.ObjectKey{Name: req.Namespace}, &namespace)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	project := projectv1.Project{}
	err = v.Client.Get(ctx, client.ObjectKey{Name: namespace.Labels["project"]}, &project)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	quota := corev1.ResourceQuota{}
	err = v.decoder.Decode(req, &quota)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	if quota.Name == "project-quota" && namespace.Labels["project"] == project.Name {
		return allowOrDenyDelete(namespace)
	}
	return admission.Allowed("resourceQuota not related to project")
}

func (v *ResourceQuotaValidator) allResourceQuotasInProject( ctx context.Context, project projectv1.Project) (ResourceQuotaList corev1.ResourceQuotaList, err error) {
	namespaceList := corev1.NamespaceList{}
	err = v.Client.List(ctx, &namespaceList)

	resourceQuotaList := corev1.ResourceQuotaList{}
	var resourceQuota corev1.ResourceQuota

	for _, namespace := range namespaceList.Items {
		if namespace.Labels["project"] == project.Name {

			err = v.Client.Get(ctx, client.ObjectKey{Name:"project-quota", Namespace: namespace.Name}, &resourceQuota)

			resourceQuotaList.Items = append(resourceQuotaList.Items, resourceQuota)

		}
	}
	return resourceQuotaList, err
}

func allowOrDenyUpdateOrCreate(project projectv1.Project, quota corev1.ResourceQuota, oldQuota *corev1.ResourceQuota, allResourceQuotas corev1.ResourceQuotaList) admission.Response {


	projectCpuLimit := project.Spec.ProjectLimits[corev1.ResourceLimitsCPU]
	projectMemoryLimit := project.Spec.ProjectLimits[corev1.ResourceLimitsMemory]

	if oldQuota == nil  {
		return admission.Allowed("allow creation of resourceQuota, by default it does not increase cpu or memory usage in the project")
	} else if oldQuota.Spec.Hard.Cpu().Value() >= quota.Spec.Hard.Cpu().Value() &&
		oldQuota.Spec.Hard.Memory().Value() >= quota.Spec.Hard.Memory().Value(){
		return admission.Allowed("resourceQuota cpu and memory can be decreased no matter the limits")
	} else {
		var SumRQCpu int64 = 0
		var SumRQMemory int64 = 0

		for _, resourceQuota := range allResourceQuotas.Items  {

			SumRQCpu += resourceQuota.Spec.Hard.Cpu().Value()
			SumRQMemory += resourceQuota.Spec.Hard.Memory().Value()
		}

		if SumRQCpu <= projectCpuLimit.Value() &&
			SumRQMemory <= projectMemoryLimit.Value() {
			return admission.Allowed("sum of resourceQuotas memory and cpu limits below project's limits, allow resourceQuota update")
		} else {
			return admission.Denied("resourceQuota cpu or memory increase is forbidden when project limits have been exceeded")
		}

	}
}

func allowOrDenyDelete(namespace corev1.Namespace) admission.Response {

	if namespace.DeletionTimestamp != nil {
		return admission.Allowed("Namespace terminating")
	}
	return admission.Denied("Namespace not terminating")
}

// resourceQuotaValidator implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (v *ResourceQuotaValidator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}
