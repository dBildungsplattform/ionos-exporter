# ionos-exporter

![Version: 0.0.11](https://img.shields.io/badge/Version-0.0.11-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.0.3](https://img.shields.io/badge/AppVersion-0.0.3-informational?style=flat-square)

A Helm chart for Kubernetes

## How to install this chart

```console
helm install chart_name ./ionos-exporter
```

To install the chart with the release name `my-release`:

```console
helm install chart_name ./ionos-exporter
```

To install with some set values:

```console
helm install chart_name ./ionos-exporter --set values_key1=value1 --set values_key2=value2
```

To install with custom values file:

```console
helm install chart_name ./ionos-exporter -f values.yaml
```

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| image.repository | string | ghcr.io/dbildungsplattform/ionos-exporter | registry to pull image from |
| image.pullPolicy | string | IfNotPresent | overwrite image pull policy |
| image.tag | string | Chart.AppVersion | set image tag |
| nameOverride | string | "" | Overrides Chart.name as the name of the Kubernetes deployment as defined in the ionos-exporter.name template | 
| fullnameOverride | string | "" | Overrides building a fully qualified name from Release.name and ionos-exporter.name as defined in the ionos-exporter.fullname template | 
| containerPort | int | 9100 | port to be used for exposing the metrics |
| ionos_credentials_secret_name | string | ionos-exporter-credentials | name of kubernetes secret that entails ionos credentials |
| ionos_credentials_username_key | string | username | key of secret to reference to username |
| ionos_credentials_password_key | string | password | key of secret to reference to password |
| serviceAccount.create | bool | true | device whether to create a service acccount |
| serviceAccount.name | string | "" | if not set and create is true name is generated using the fullname template |
| replicaCount | int | 1 | number of replicas |
| ionosApiCycle | int | 900 | cycle time in seconds to query the IONOS API for changes |
