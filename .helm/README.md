# Helm Chart: T Cloud ELB Certificate Sync

The controller watches TLS Secrets in all namespaces and patches their sync-state annotations. Review the cluster-wide Secret permissions before installing it.

Create a Kubernetes Secret named `t-cloud-credentials` in the release namespace with keys `OS_ACCESS_KEY` and `OS_SECRET_KEY`. Do not commit actual credentials to Helm values or Git.

Download the Helm chart archive from a GitHub release, then install it with your published image and project ID:

```bash
helm install cert-sync ./t-cloud-cert-sync-0.1.0.tgz \
  --namespace cert-sync --create-namespace \
  --set image.tag=v0.1.0 \
  --set config.tCloudProjectId=YOUR_PROJECT_ID \
  --set credentials.existingSecret.name=t-cloud-credentials
```

The chart defaults to `ghcr.io/telekom-mms/t-cloud-cert-sync`. Replace the image tag, project ID, and chart archive name with the published release values. Override `image.repository` if using a fork or another registry. Set `config.tCloudRegion` if the project is not in `eu-de`. `config.keepCertRegex` defaults to `dummy`: certificates whose names match this regex will not be removed during cleanup. If your image is private, supply `imagePullSecrets`.
