# IONOS Exporter 
This application polls the [Ionos API](https://pkg.go.dev/github.com/ionos-cloud/sdk-go/v6@v6.1.9#section-readme) for [Datacenter](https://github.com/ionos-cloud/sdk-go/blob/v6.1.9/model_datacenters.go#L18) objects. These objects contain information on the amount of servers, RAM, and CPU cores currently in use. The data is logged and exposed as Prometheus metrics at the _/metrics_ endpoint. 

## Deployment
The application is packaged as a Helm chart for deployment. The compilation of container image and Helm chart are automated via Github Actions. The workflows are triggered once a semver git tag (of type 'v[0-9]+.[0-9]+.[0-9]+') is pushed. Container images are compiled and pushed to [ghcr](https://github.com/dbildungsplattform/ionos-exporter/pkgs/container/ionos-exporter). The helm charts are pushed to this [registry](https://github.com/dBildungsplattform/helm-charts-registry). 
For more details on application configuration and Helm chart packaging see [/charts/ionos-exporter/README.md](/charts/ionos-exporter/README.md).
The rollout of the provided Helm chart is automated with Ansible and the [kubernetes.core.helm](https://docs.ansible.com/ansible/latest/collections/kubernetes/core/helm_module.html) Module. 
A release follows these steps:
1. Increment _version_ and _AppVersion_ in `charts/ionos-exporter/Chart.yaml`
2. Create a tag with `git tag <AppVersion>`. Make sure the tags versions matches with _AppVersion_. This is the default for the container images tag as defined in `charts/ionos-exporter/templates/deployment.yaml`.
3. Push tags (and other changes) upstream with `git push origin --tags` 
