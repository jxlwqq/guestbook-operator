# guestbook-operator

「[Kubernetes 官方示例：使用 Redis 部署 PHP 留言板应用程序](https://kubernetes.io/docs/tutorials/stateless-application/guestbook/)」Operator 化。

### 前置条件

* 安装 Docker Desktop，并启动内置的 Kubernetes 集群
* 注册一个 [hub.docker.com](https://hub.docker.com/) 账户，需要将本地构建好的镜像推送至公开仓库中
* 安装 operator SDK CLI: `brew install operator-sdk`
* 安装 Go: `brew install go`

本示例推荐的依赖版本：

* Docker Desktop: >= 4.0.0
* Kubernetes: >= 1.21.4
* Operator-SDK: >= 1.11.0
* Go: >= 1.17

> jxlwqq 为笔者的 ID，命令行和代码中涉及的个人 ID，均需要替换为读者自己的，包括
> * `--domain=`
> * `--repo=`
> * `//+kubebuilder:rbac:groups=`
> * `IMAGE_TAG_BASE ?=`

### 创建项目

使用 Operator SDK CLI 创建名为 guestbook-operator 的项目。

```shell
mkdir -p $HOME/projects/guestbook-operator
cd $HOME/projects/guestbook-operator
go env -w GOPROXY=https://goproxy.cn,direct
```shell

operator-sdk init \
--domain=jxlwqq.github.io \
--repo=github.com/jxlwqq/guestbook-operator \
--skip-go-version-check
```


### 创建 API 和控制器

使用 Operator SDK CLI 创建自定义资源定义（CRD）API 和控制器。

运行以下命令创建带有组 app、版本 v1alpha1 和种类 Guestbook 的 API：

```shell
operator-sdk create api \
--resource=true \
--controller=true \
--group=app \
--version=v1alpha1 \
--kind=Guestbook
```


定义 Guestbook 自定义资源（CR）的 API。

修改 api/v1alpha1/guestbook_types.go 中的 Go 类型定义，使其具有以下 spec 和 status

```go
type GuestbookSpec struct {
	FrontendSize int32 `json:"frontendSize"`
	RedisFollowerSize int32 `json:"redisFollowerSize"`
}
```



为资源类型更新生成的代码：
```shell
make generate
```


运行以下命令以生成和更新 CRD 清单：
```shell
make manifests
```


### 实现控制器

> 由于逻辑较为复杂，代码较为庞大，所以无法在此全部展示，完整的操作器代码请参见 controllers 目录。
在本例中，将生成的控制器文件 controllers/wordpress_controller.go 替换为以下示例实现：
```go
/*
Copyright 2021.

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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appv1alpha1 "github.com/jxlwqq/guestbook-operator/api/v1alpha1"
)

// GuestbookReconciler reconciles a Guestbook object
type GuestbookReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=app.jxlwqq.github.io,resources=guestbooks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.jxlwqq.github.io,resources=guestbooks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.jxlwqq.github.io,resources=guestbooks/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=service,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Guestbook object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *GuestbookReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)
	reqLogger.Info("Reconciling Guestbook")

	guestbook := &appv1alpha1.Guestbook{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, guestbook)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	var result = &ctrl.Result{}

	result, err = r.ensureDeployment(r.redisLeaderDeployment(guestbook))
	if result != nil {
		return *result, err
	}
	result, err = r.ensureService(r.redisLeaderService(guestbook))
	if result != nil {
		return *result, err
	}

	result, err = r.ensureDeployment(r.redisFollowerDeployment(guestbook))
	if result != nil {
		return *result, err
	}
	result, err = r.ensureService(r.redisFollowerService(guestbook))
	if result != nil {
		return *result, err
	}
	result, err = r.handleRedisFollowerChanges(guestbook)
	if result != nil {
		return *result, err
	}

	result, err = r.ensureDeployment(r.frontendDeployment(guestbook))
	if result != nil {
		return *result, err
	}
	result, err = r.ensureService(r.frontendService(guestbook))
	if result != nil {
		return *result, err
	}
	result, err = r.handleFrontendChanges(guestbook)
	if result != nil {
		return *result, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GuestbookReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1alpha1.Guestbook{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
```

运行以下命令以生成和更新 CRD 清单：
```shell
make manifests
```
