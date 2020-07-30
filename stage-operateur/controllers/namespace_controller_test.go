package controllers

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("resourceQuotaShouldBePresent", func() {

	It("should return true if namespace has label project set and no deletion timestamp", func() {
		// Given
		namespace := corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "test1", Labels: map[string]string{"project": "patate"}}}
		// When
		result := resourceQuotaShouldBePresent(&namespace)

		// Then
		Expect(result).To(BeTrue())
	})

	It("should return false if namespace has deletion timestamp", func() {
		// Given
		instant := v1.Now()

		namespace := corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "test1", DeletionTimestamp: &instant, Labels: map[string]string{"project": "patate"}}}
		// When
		result := resourceQuotaShouldBePresent(&namespace)

		// Then
		Expect(result).To(BeFalse())
	})

	It("should return false if namespace has no project label set", func() {
		// Given
		namespace := corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: "test1", Labels: map[string]string{"patate": "saute"}}}
		// When
		result := resourceQuotaShouldBePresent(&namespace)

		// Then
		Expect(result).To(BeFalse())
	})

})

var _ = Describe("newDefaultResourceQuota", func() {
	It("should create ResourceQuota with correct specs", func() {
		// Given
		namespaceName := "namespace-1"
		projectName := "project-1"

		var expectedMemory int64 = 0
		var expectedCpu int64 = 0
		// When
		quota := newDefaultResourceQuota(namespaceName, projectName)

		// Then
		Expect(quota.ObjectMeta.Namespace).To(Equal(namespaceName))
		Expect(quota.ObjectMeta.Labels["project"]).To(Equal(projectName))
		Expect(quota.Spec.Hard.Memory().Value()).To(Equal(expectedMemory))
		Expect(quota.Spec.Hard.Cpu().Value()).To(Equal(expectedCpu))
		Expect(quota.ObjectMeta.Name).To(Equal("project-quota"))
	})

})
