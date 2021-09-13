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

const redisFollowerDeploymentName = "redis-follower"

func redisFollowerLabels() map[string]string {
	return labels("redis", "follower", "backend")
}

func (r *GuestbookReconciler) redisFollowerDeployment(i *appv1alpha1.Guestbook) *appsv1.Deployment {
	labels := redisFollowerLabels()
	size := i.Spec.RedisFollowerSize
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: i.Namespace,
			Name:      redisFollowerDeploymentName,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &size,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            "follower",
						Image:           "gcr.io/google_samples/gb-redis-follower:v2",
						ImagePullPolicy: corev1.PullIfNotPresent,
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

func (r *GuestbookReconciler) redisFollowerService(i *appv1alpha1.Guestbook) *corev1.Service {
	labels := redisFollowerLabels()
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: i.Namespace,
			Name:      "redis-follower",
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

func (r *GuestbookReconciler) handleRedisFollowerChanges(i *appv1alpha1.Guestbook) (*ctrl.Result, error) {
	found := &appsv1.Deployment{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: i.Namespace,
		Name:      redisFollowerDeploymentName,
	}, found)
	if err != nil {
		return &ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}
	size := i.Spec.RedisFollowerSize
	if size != *found.Spec.Replicas {
		*found.Spec.Replicas = size
		err = r.Client.Update(context.TODO(), found)
		if err != nil {
			return &ctrl.Result{}, err
		}
	}
	return nil, nil
}
