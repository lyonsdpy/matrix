# deploy/k8s/

Kubernetes 部署清单。用于生产/预发环境。

## 目录结构

```
k8s/
├── base/                   # 基础配置（Kustomize base）
│   ├── api-deployment.yaml
│   ├── web-deployment.yaml
│   ├── api-service.yaml
│   ├── web-service.yaml
│   └── kustomization.yaml
└── overlays/               # 环境覆写（Kustomize overlays）
    ├── staging/
    │   └── kustomization.yaml
    └── production/
        └── kustomization.yaml
```

## 常见资源文件

| 文件 | 说明 |
|---|---|
| `*-deployment.yaml` | Deployment 定义（副本数、镜像、资源限制）|
| `*-service.yaml` | Service（ClusterIP / LoadBalancer）|
| `ingress.yaml` | Ingress 路由规则（域名、TLS）|
| `configmap.yaml` | 非敏感配置（环境变量）|
| `*-secret.yaml` | 敏感配置模板（实际值通过 CI 注入，不提交）|

## 约定

- 使用 Kustomize 管理多环境差异，不维护多份重复的 YAML
- Secret 文件（`*-secret.yaml`）不提交到 git，通过 CI/CD pipeline 或外部 secret 管理工具注入
- 所有容器设置 `resources.requests` 和 `resources.limits`
