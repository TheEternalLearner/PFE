package integration_tests

import (
	"context"
	projectv1 "project/api/v1"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestNamespaceReconcile(t *testing.T) {

	var _ = Describe("Namespace reconciler tests", func() {

		const interval = 1 * time.Second
		const timeout = 30 * time.Second

		Context("When labeling a namespace", func() {
			It("Should update the status of the project", func() {
				ctx := context.Background()

				By("Creating project test-project-1 and namespace test-1")
				projectTest1 := projectv1.Project{ObjectMeta: v1.ObjectMeta{Name: "project-test-1"}}

				namespaceTest1 := corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "test-1", Labels: map[string]string{"project": "project-test-1"}}}

				err := k8sClient.Create(ctx, &projectTest1)
				Expect(err).NotTo(HaveOccurred())

				err = k8sClient.Create(ctx, &namespaceTest1)
				Expect(err).NotTo(HaveOccurred())

				key := client.ObjectKey{Name: "project-test-1"}

				By("Checking if namespace name was added to project status")
				Eventually(func() []string {
					Expect(k8sClient.Get(ctx, key, &projectTest1)).Should(Succeed())
					return projectTest1.Status.Namespaces
				}, timeout).Should(ConsistOf(namespaceTest1.Name), "timed out waiting for test-project-1 status.namespaces field to be correctly filled")

				By("Delete namespace and project")
				Expect(k8sClient.Delete(ctx, &namespaceTest1)).Should(Succeed())
				Expect(k8sClient.Delete(ctx, &projectTest1)).Should(Succeed())
			})
		})

		Context("namespace resourceQuota tests", func() {
			It("Test 2", func() {
				ctx := context.Background()

				By("Creating project test-project-1")
				project := projectv1.Project{}
				project.Name = "project-test-2"

				err := k8sClient.Create(ctx, &project)
				Expect(err).NotTo(HaveOccurred())

				By("Creating namespace test-2 labeled project-test-2")
				namespace := corev1.Namespace{}
				namespace.Name = "test-2"
				namespace.Labels = make(map[string]string)
				namespace.Labels["project"] = "project-test-2"

				err = k8sClient.Create(ctx, &namespace)
				Expect(err).NotTo(HaveOccurred())

				By("Getting resourceQuota")
				quota := corev1.ResourceQuota{}
				key := client.ObjectKey{Name: "project-quota", Namespace: "test-2"}
				Eventually(Expect(k8sClient.Get(ctx, key, &quota)).Should(Succeed()), timeout, interval)

				cpu := resource.Quantity{Format: "DecimalSI"}
				cpu.Set(0)
				Expect(quota.Spec.Hard.Cpu().Value()).To(Equal(cpu.Value()))

				memory := resource.Quantity{}
				memory.Set(0)

				Expect(quota.Spec.Hard.Memory().Value()).To(Equal(memory.Value()))

				By("Delete namespace and project")
				Expect(k8sClient.Delete(ctx, &namespace)).Should(Succeed())
				Expect(k8sClient.Delete(ctx, &project)).Should(Succeed())
			})
		})
	})
}
