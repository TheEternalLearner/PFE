package webhook

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	projectv1 "project/api/v1"
	"testing"
)

func setResourceQuota(cpu int64, memory int64 ) corev1.ResourceQuota {
	quota := corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:"project-quota",
			Labels: map[string]string{"project":"project-1"},
		},
		Spec:       corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{
				corev1.ResourceCPU: *resource.NewQuantity( cpu, resource.DecimalSI),
				corev1.ResourceMemory: *resource.NewQuantity(memory, resource.BinarySI),
			},
		},
	}
	return quota
}

func setProject(cpuLimit int64, memoryLimit int64) projectv1.Project {
	project := projectv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "project-1",
		},
		Spec:       projectv1.ProjectSpec{
			ProjectLimits: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceLimitsCPU: *resource.NewQuantity(cpuLimit, resource.DecimalSI),
				corev1.ResourceLimitsMemory: *resource.NewQuantity(memoryLimit, resource.BinarySI),
			},
		},
	}
	return project
}

func fillResourcequotaList(quotas... corev1.ResourceQuota) corev1.ResourceQuotaList {
	list := corev1.ResourceQuotaList{
		Items: quotas,
	}
	return list
}

var _ = Describe("Testing allowOrDenyUpdateOrCreate function", func() {

	quotaDefault := setResourceQuota(0,0)
	quotaJustBelowLimit := setResourceQuota(100, 10000)
	quota1 := setResourceQuota(10, 1000)
	quota2 := setResourceQuota(80, 8000)
	quotaExceedCpu := setResourceQuota(101, 0)
	quotaExceedMemory := setResourceQuota(0, 10001)
	quotaMoreCpu := setResourceQuota(11, 0)
	quotaMoreMemory := setResourceQuota(0, 1001)
	quotaEvenMoreCpu := setResourceQuota(12, 0)
	quotaEvenMoreMemory := setResourceQuota(0, 1002)

	project1 := setProject(100, 10000)

	Context("Only one namespace in the project", func() {
		It("Should allow to increase resourceQuota's when cpu and memory are below or equal to the project's limits", func() {
			//Given
			project := project1

			quota := quotaJustBelowLimit

			oldQuota := &quotaDefault

			resourceQuotaList := corev1.ResourceQuotaList{
				Items: []corev1.ResourceQuota{
					quota,
				},
			}

			reason := metav1.StatusReason("sum of resourceQuotas memory and cpu limits below project's limits, allow resourceQuota update")

			//When
			result := allowOrDenyUpdateOrCreate(project, quota, oldQuota, resourceQuotaList)

			//Then
			Expect(result.Allowed).To(BeTrue())
			Expect(result.Result.Reason).To(Equal(reason))
		})

		It("Should deny to increase resourceQuota's cpu over the project's limits", func() {
			//Given
			project := project1

			quota := quotaExceedCpu

			oldQuota := &quotaDefault

			resourceQuotaList := fillResourcequotaList(quota)

			reason := metav1.StatusReason("resourceQuota cpu or memory increase is forbidden when project limits have been exceeded")

			//When
			result := allowOrDenyUpdateOrCreate(project, quota, oldQuota, resourceQuotaList)

			//Then
			Expect(result.Allowed).To(BeFalse())
			Expect(result.Result.Reason).To(Equal(reason))
		})

		It("Should deny to increase resourceQuota's memory over the project's limits", func() {
			//Given
			project := project1

			quota := quotaExceedMemory

			oldQuota := &quotaDefault

			resourceQuotaList := fillResourcequotaList(quota)

			reason := metav1.StatusReason("resourceQuota cpu or memory increase is forbidden when project limits have been exceeded")

			//When
			result := allowOrDenyUpdateOrCreate(project, quota, oldQuota, resourceQuotaList)

			//Then
			Expect(result.Allowed).To(BeFalse())
			Expect(result.Result.Reason).To(Equal(reason))
		})
	})

	Context("More than one namespace in project", func() {
		It("Should allow to update resourceQuota when sum of resourceQuotas cpu and sum of resourceQuotas memory are below the project's limit for memory and cpu", func() {
			//Given
			project := project1

			currentQuota := quota1

			oldQuota := &quotaDefault

			otherResourceQuotas := fillResourcequotaList(currentQuota, quota1, quota2)

			reason := metav1.StatusReason("sum of resourceQuotas memory and cpu limits below project's limits, allow resourceQuota update")

			//When
			result := allowOrDenyUpdateOrCreate(project, currentQuota, oldQuota, otherResourceQuotas)

			//Then
			Expect(result.Allowed).To(BeTrue())
			Expect(result.Result.Reason).To(Equal(reason))
		})

		It("Should deny the increase of resourceQuota cpu if sum of resourceQuotas cpu exceeds the project limit for cpu", func() {
			//Given
			project := project1

			currentQuota := quotaMoreCpu

			oldQuota := &quota1

			otherResourceQuotas := fillResourcequotaList(currentQuota, quota1, quota2)

			reason := metav1.StatusReason("resourceQuota cpu or memory increase is forbidden when project limits have been exceeded")

			//When
			result := allowOrDenyUpdateOrCreate(project, currentQuota, oldQuota, otherResourceQuotas)

			//Then
			Expect(result.Allowed).To(BeFalse())
			Expect(result.Result.Reason).To(Equal(reason))
		})

		It("Should deny the increase of resourceQuota memory if sum of resourceQuotas memory exceeds the project limit for memory", func() {
			//Given
			project := project1
			currentQuota := quotaMoreMemory

			oldQuota := &quota1

			otherResourceQuotas := fillResourcequotaList(currentQuota, quota1, quota2)

			reason := metav1.StatusReason("resourceQuota cpu or memory increase is forbidden when project limits have been exceeded")

			//When
			result := allowOrDenyUpdateOrCreate(project, currentQuota, oldQuota, otherResourceQuotas)

			//Then
			Expect(result.Allowed).To(BeFalse())
			Expect(result.Result.Reason).To(Equal(reason))
		})

		It("Should allow the decrease of resourceQuota cpu even if sum of resourceQuotas cpu exceeds the project limit for cpu", func() {
			//Given
			project := project1

			currentQuota := quotaMoreCpu
			oldQuota := &quotaEvenMoreCpu

			otherResourceQuotas := fillResourcequotaList(quotaMoreCpu, quota1, quota2)

			reason := metav1.StatusReason("resourceQuota cpu and memory can be decreased no matter the limits")

			//When
			result := allowOrDenyUpdateOrCreate(project, currentQuota, oldQuota, otherResourceQuotas)

			//Then
			Expect(result.Allowed).To(BeTrue())
			Expect(result.Result.Reason).To(Equal(reason))
		})

		It("Should allow the decrease of resourceQuota memory even if sum of resourceQuotas memory exceeds the project limit for memory", func() {
			//Given
			project := project1
			currentQuota := quotaMoreMemory

			oldQuota := &quotaEvenMoreMemory
			otherResourceQuotas := fillResourcequotaList(currentQuota, quota1, quota2)

			reason := metav1.StatusReason("resourceQuota cpu and memory can be decreased no matter the limits")

			//When
			result := allowOrDenyUpdateOrCreate(project, currentQuota, oldQuota, otherResourceQuotas)

			//Then
			Expect(result.Allowed).To(BeTrue())
			Expect(result.Result.Reason).To(Equal(reason))
		})
	})
})


var _ = Describe("Testing function allowOrDenyDelete", func() {

	It("Should deny resourceQuota belonging to a project deletion if namespace is not terminating", func() {
		//Given
		namespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test1", DeletionTimestamp: nil}}

		reason := metav1.StatusReason("Namespace not terminating")

		//when
		result := allowOrDenyDelete(namespace)

		//Then
		Expect(result.Allowed).To(BeFalse())
		Expect(result.Result.Reason).To(Equal(reason))

	})

	It("Should allow resourceQuota deletion if namespace is terminating", func() {
		//Given
		instant := metav1.Now()
		namespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test1", Labels: map[string]string{"project": "project-1", "other-tag": "whatever"}, DeletionTimestamp: &instant}}

		reason := metav1.StatusReason("Namespace terminating")

		//when
		result := allowOrDenyDelete(namespace)

		//Then
		Expect(result.Allowed).To(BeTrue())
		Expect(result.Result.Reason).To(Equal(reason))
	})
})

func  Test_allowOrDenyUpdateOrCreate(t *testing.T) {

	quotaDefault := setResourceQuota(0,0)
	quota1 := setResourceQuota(10, 1000)
	quota2 := setResourceQuota(80, 8000)
	quotaMoreCpu := setResourceQuota(11, 0)
	quotaMoreMemory := setResourceQuota(0, 1001)
	quotaEvenMoreCpu := setResourceQuota(12, 0)
	quotaEvenMoreMemory := setResourceQuota(0, 1002)

	project1 := setProject(100, 10000)

	type args struct {
		project           projectv1.Project
		quota             corev1.ResourceQuota
		oldQuota          *corev1.ResourceQuota
		allResourceQuotas corev1.ResourceQuotaList
	}
	type want struct {
		Allowed bool
		Reason metav1.StatusReason
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "testing resourceQuota creation when project's limit's have not been exceeded",
			args:args{
				project: project1,
				quota: quotaDefault,
				oldQuota: nil,
				allResourceQuotas: fillResourcequotaList(quotaDefault),
			},
			want:want {
			Allowed: true,
			Reason: metav1.StatusReason("allow creation of resourceQuota, by default it does not increase cpu or memory usage in the project"),
			},
		},

		{
			name: "testing resourceQuota creation when project's limit's have been exceeded",
			args:args{
				project: project1,
				quota: quotaDefault,
				oldQuota: nil,
				allResourceQuotas: fillResourcequotaList(quotaDefault, quota2, quotaMoreMemory, quotaEvenMoreCpu, quota1),
			},
			want:want {
				Allowed: true,
				Reason: metav1.StatusReason("allow creation of resourceQuota, by default it does not increase cpu or memory usage in the project"),
			},
		},

		{
			name: "testing resourceQuota increase of Cpu and Memory below project's limits",
			args:args{
				project: project1,
				quota: quota1,
				oldQuota: &quotaDefault,
				allResourceQuotas: fillResourcequotaList(quota1, quota1, quota2),
			},
			want:want {
				Allowed: true,
				Reason: metav1.StatusReason("sum of resourceQuotas memory and cpu limits below project's limits, allow resourceQuota update"),
			},
		},

		{
			name: "testing resourceQuota increase of Cpu above project's Cpu limit",
			args:args{
				project: project1,
				quota: quotaMoreCpu,
				oldQuota: &quotaDefault,
				allResourceQuotas: fillResourcequotaList(quota1, quota2, quotaMoreCpu),
			},
			want:want {
				Allowed: false,
				Reason: metav1.StatusReason("resourceQuota cpu or memory increase is forbidden when project limits have been exceeded"),
			},
		},

		{
			name: "testing resourceQuota increase of Cpu above project's Cpu limit",
			args:args{
				project: project1,
				quota: quotaMoreMemory,
				oldQuota: &quotaDefault,
				allResourceQuotas: fillResourcequotaList(quota1, quota2, quotaMoreMemory),
			},
			want:want {
				Allowed: false,
				Reason: metav1.StatusReason("resourceQuota cpu or memory increase is forbidden when project limits have been exceeded"),
			},
		},

		{
			name: "testing resourceQuota cpu decrease when project's limits have been exceeded",
			args:args{
				project: project1,
				quota: quotaMoreCpu,
				oldQuota: &quotaEvenMoreCpu,
				allResourceQuotas: fillResourcequotaList(quota1, quota2, quotaMoreCpu),
			},
			want:want {
				Allowed: true,
				Reason: metav1.StatusReason("resourceQuota cpu and memory can be decreased no matter the limits"),
			},
		},

		{
			name: "testing resourceQuota memory decrease when project limit's have been exceeded",
			args:args{
				project: project1,
				quota: quotaMoreMemory,
				oldQuota: &quotaEvenMoreMemory,
				allResourceQuotas: fillResourcequotaList(quota1, quota2, quotaMoreMemory),
			},
			want:want {
				Allowed: true,
				Reason: metav1.StatusReason("resourceQuota cpu and memory can be decreased no matter the limits"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := allowOrDenyUpdateOrCreate(tt.args.project, tt.args.quota, tt.args.oldQuota, tt.args.allResourceQuotas)
			if got.Allowed != tt.want.Allowed {

				t.Errorf("Allow = %v, want %v", got.Allowed, tt.want.Allowed)

				if got.Result.Reason != tt.want.Reason {

					t.Errorf("Reason = %v, want %v", got.Result.Reason, tt.want.Reason)

				}
			}

		})
	}
}