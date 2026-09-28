# Helm Chart: T Cloud ELB Certificate Sync

The controller watches TLS Secrets in all namespaces and patches their sync-state annotations. Review the cluster-wide Secret permissions before installing it.

Create a Kubernetes Secret named `t-cloud-credentials` in the release namespace with keys `OS_ACCESS_KEY` and `OS_SECRET_KEY`. Do not commit actual credentials to Helm values or Git.

Install the latest GitHub release of the chart with its matching `latest` image:

```bash
helm install cert-sync https://github.com/telekom-mms/t-cloud-cert-sync/releases/latest/download/t-cloud-cert-sync.tgz \
  --namespace cert-sync --create-namespace \
  --set config.tCloudProjectId=YOUR_PROJECT_ID \
  --set credentials.existingSecret.name=t-cloud-credentials
```

The chart defaults to `ghcr.io/telekom-mms/t-cloud-cert-sync:latest`. The URL works after the first GitHub release and follows its latest non-prerelease version. For a fixed deployment, use a versioned chart asset and set `image.tag` to the corresponding `v` tag. Replace the project ID and override `image.repository` if using a fork or another registry. Set `config.tCloudRegion` if the project is not in `eu-de`. `config.keepCertRegex` defaults to `dummy`: certificates whose names match this regex will not be removed during cleanup. If your image is private, supply `imagePullSecrets`.
