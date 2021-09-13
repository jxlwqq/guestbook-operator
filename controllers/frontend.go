package controllers

import (
	"context"
	appv1alpha1 "github.com/jxlwqq/guestbook-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)

func frontendLabels() map[string]string {
	return labels("guestbook", "", "frontend")
}

const frontendDeploymentName = "frontend"

func (r *GuestbookReconciler) frontendDeployment(i *appv1alpha1.Guestbook) *appsv1.Deployment {
	size := i.Spec.FrontendSize
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: i.Namespace,
			Name:      frontendDeploymentName,
			Labels:    frontendLabels(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &size,
			Selector: &metav1.LabelSelector{
				MatchLabels: frontendLabels(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: frontendLabels(),
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            "php-redis",
						Image:           "gcr.io/google_samples/gb-frontend:v5",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Env: []corev1.EnvVar{
							{
								Name:  "GET_HOSTS_FROM",
								Value: "dns",
							},
						},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 80,
						}},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("100m"),
								corev1.ResourceMemory: resource.MustParse("100Mi"),
							},
						},
					}},
				},
			},
		},
	}

	_ = controllerutil.SetControllerReference(i, dep, r.Scheme)

	return dep
}

func (r *GuestbookReconciler) frontendService(i *appv1alpha1.Guestbook) *corev1.Service {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: i.Namespace,
			Name:      "frontend",
			Labels:    frontendLabels(),
		},
		Spec: corev1.ServiceSpec{
			Selector: frontendLabels(),
			Ports: []corev1.ServicePort{{
				Port:       80,
				TargetPort: intstr.FromInt(80),
				NodePort:   30693,
			}},
			Type: corev1.ServiceTypeNodePort,
		},
	}

	_ = controllerutil.SetControllerReference(i, svc, r.Scheme)

	return svc
}

func (r *GuestbookReconciler) handleFrontendChanges(i *appv1alpha1.Guestbook) (*ctrl.Result, error) {
	found := &appsv1.Deployment{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: i.Namespace,
		Name:      frontendDeploymentName,
	}, found)
	if err != nil {
		return &ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}

	size := i.Spec.FrontendSize
	if size != *found.Spec.Replicas {
		*found.Spec.Replicas = size
		err = r.Client.Update(context.TODO(), found)
		if err != nil {
			return &ctrl.Result{}, err
		}
	}

	return nil, nil
}
