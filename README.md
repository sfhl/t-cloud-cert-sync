# T Cloud ELB Certificate Sync Controller

This Kubernetes controller watches TLS Secrets annotated with `t-cloud.telekom.com/listener-id: "<listener-uuid>"`. When the certificate changes, it uploads the new certificate to T Cloud ELB v3 and binds it to that listener. It remembers successful syncs in a Secret annotation.

The previous certificate is deleted after a successful bind unless its name matches `KEEP_CERT_REGEX`. If the certificate name cannot be checked, cleanup is skipped. The controller needs cluster-wide access to Secrets; deploy it only in a trusted cluster.

## Install

Use a Kubernetes cluster with cert-manager or another source of TLS Secrets, a T Cloud project, and a published container image. Create a Kubernetes Secret in the controller's namespace containing `OS_ACCESS_KEY` and `OS_SECRET_KEY`. The [Helm chart](.helm/README.md) documents installation with an existing Secret and an explicit image tag.

## Development

Go 1.23.12 or newer is required. The controller itself uses in-cluster Kubernetes credentials.

```bash
go test -mod=readonly ./...
docker build -t t-cloud-cert-sync:dev .
```

## Configuration

| Variable | Description |
|---|---|
| `OS_AUTH_URL` | Identity endpoint, e.g. `https://iam.eu-de.otc.t-systems.com/v3` |
| `OS_REGION_NAME` | T Cloud region, e.g. `eu-de` |
| `OS_PROJECT_ID` | T Cloud project ID |
| `OS_ACCESS_KEY` | T Cloud access key (from a Kubernetes Secret) |
| `OS_SECRET_KEY` | T Cloud secret key (from a Kubernetes Secret) |
| `KEEP_CERT_REGEX` | Regex matched against old certificate names before cleanup; default `dummy`. An invalid regex stops startup. |

## Releases

GitHub Actions runs tests, lints the chart, and builds the image on pull requests and pushes. A `v*` tag publishes the image to GHCR and attaches the Helm chart to a GitHub release. Set the GHCR package visibility to public before using the chart without registry credentials; the repository owner must authorize publication of the code and images.

Licensed under [GPL-3.0](LICENSE).
