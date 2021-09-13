package controllers

import (
	appv1alpha1 "github.com/jxlwqq/guestbook-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func redisLeaderLabels() map[string]string {
	return labels("redis", "follower", "backend")
}

func (r *GuestbookReconciler) redisLeaderDeployment(i *appv1alpha1.Guestbook) *appsv1.Deployment {
	labels := redisLeaderLabels()
	replicas := int32(1)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: i.Namespace,
			Name:      "redis-leader",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            "leader",
						Image:           "docker.io/redis:6.0.5",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Ports: []corev1.ContainerPort{{
							Protocol:      corev1.ProtocolTCP,
							ContainerPort: 6379,
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

func (r *GuestbookReconciler) redisLeaderService(i *appv1alpha1.Guestbook) *corev1.Service {
	labels := redisLeaderLabels()
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: i.Namespace,
			Name:      "redis-leader",
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Port:       6379,
				TargetPort: intstr.FromInt(6379),
			}},
		},
	}

	_ = controllerutil.SetControllerReference(i, svc, r.Scheme)

	return svc
}
