package controllers

import (
	projectv1 "project/api/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("updateProjectStatus", func() {
	It("should let project.Status.Namespaces empty", func() {
		// Given
		namespaces := corev1.NamespaceList{
			Items: []corev1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test1",
					},
				},
			},
		}
		project := projectv1.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name: "project-test1",
			},
		}

		// When
		updateProjectStatus(namespaces, &project)

		// Then
		Expect(project.Status.Namespaces).To(BeEmpty())
	})

	It("should populate project.Status.Namespaces with 1 namespace", func() {
		// Given
		namespaces := corev1.NamespaceList{
			Items: []corev1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test1",
						Labels: map[string]string{
							"project": "project-test1",
						},
					},
				},
			},
		}
		project := projectv1.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name: "project-test1",
			},
		}

		// When
		updateProjectStatus(namespaces, &project)

		// Then
		Expect(project.Status.Namespaces).To(ConsistOf("test1"))
	})

	It("should not keep namespaces in project.Status.Namespace if namespace does not have the correct label anymore", func() {
		// Given
		namespaces := corev1.NamespaceList{
			Items: []corev1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test1",
						Labels: map[string]string{
							"project": "not-project-test1",
						},
					},
				},
			},
		}

		project := projectv1.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name: "project-test1",
			},
			Status: projectv1.ProjectStatus{Namespaces: []string{"test1", "test2"}},
		}

		// When
		updateProjectStatus(namespaces, &project)

		// Then
		Expect(project.Status.Namespaces).To(BeEmpty())
	})

	It("should populate project.Status.Namespaces with 2 namespace and delete one", func() {
		// Given
		namespaces := corev1.NamespaceList{
			Items: []corev1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test1",
						Labels: map[string]string{
							"project": "project-test1",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test2",
						Labels: map[string]string{
							"project": "project-test1",
						},
					},
				},
			},
		}
		project := projectv1.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name: "project-test1",
			},
			Status: projectv1.ProjectStatus{Namespaces: []string{"test3"}},
		}

		// When
		updateProjectStatus(namespaces, &project)

		// Then
		Expect(project.Status.Namespaces).To(ConsistOf("test1", "test2"))
	})

	It("should populate project.Status.Namespaces of project-test-2 with namespace test-2 project.Status.Namespaces of project-test-1 with namespaces test-1 and test-3", func() {
		// Given
		namespaces := corev1.NamespaceList{
			Items: []corev1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test1",
						Labels: map[string]string{
							"project": "project-test1",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test2",
						Labels: map[string]string{
							"project": "project-test2",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test3",
						Labels: map[string]string{
							"project": "project-test1",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test4",
						Labels: map[string]string{
							"project": "project-test4",
						},
					},
				},
			},
		}
		project1 := projectv1.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name: "project-test1",
			},
			Status: projectv1.ProjectStatus{Namespaces: []string{"test-2", "test3"}},
		}

		project2 := projectv1.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name: "project-test2",
			},
			Status: projectv1.ProjectStatus{Namespaces: []string{}},
		}

		// When
		updateProjectStatus(namespaces, &project1)
		updateProjectStatus(namespaces, &project2)

		// Then
		Expect(project1.Status.Namespaces).To(ConsistOf("test1", "test3"))
		Expect(project2.Status.Namespaces).To(ConsistOf("test2"))
	})
})
