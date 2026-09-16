# T CLOUD PUBLIC ELB Certificate Sync Controller

This Kubernetes controller dynamically monitors TLS certificates (Kubernetes Secrets) within the cluster and automatically synchronizes them with **T CLOUD PUBLIC**. It uploads renewed certificates and binds them to the configured Elastic Load Balancer (ELB v3) listener with zero downtime.

## how it's worikng

The controller uses Kubernetes Informers to detect changes to `Secret` resources across the cluster in real time (event-driven).
For the controller to process a certificate, the Secret must be annotated with a specific annotation:

`t-cloud.telekom.com/listener-id: "<deine-elb-listener-uuid>"`

1. **Detection:** As soon as a secret is created or updated with this annotation (e.g., by `cert-manager`), the controller is triggered.
2. **Caching:** It calculates a SHA256 hash of the certificate in combination with the listener ID. The T CLOUD PUBLIC API is only called if the certificate has actually been renewed or the listener ID has changed.
3. **Upload & Bind:** The new certificate is uploaded to T CLOUD PUBLIC and bound to the specified listener as the default TLS certificate.

*Note on T CLOUD PUBLIC authentication: The application uses pure AK/SK signatures and communicates directly with the ELB endpoint. This eliminates IAM token issues (especially in sub-projects) due to system constraints.*

## Prerequisites

- Go 1.23+ (for local development)
- Docker
- Open Telekom Cloud Access Key (AK) and Secret Key (SK)

## Local Development & Build

1. Download dependencies:
   ```bash
   go mod tidy

2. Run the application locally (with environment variables for T CLOUD PUBLIC access):
   ```bash
   docker build -t your-repo/t-cloud-cert-sync:latest .

## Environment Variables

| Variable | Description | Example |
|---|---|---|
| `OS_AUTH_URL` | T CLOUD PUBLIC Identity Endpoint | `https://iam.eu-de.t cloud public.t-systems.com/v3` |
| `OS_REGION_NAME` | T CLOUD PUBLIC Region | `eu-de` |
| `OS_PROJECT_ID` | id of the (sub-)Projekts | `29f93e92810d...` |
| `OS_ACCESS_KEY` | T CLOUD PUBLIC Access Key (AK) | `ABCDEF...` |
| `OS_SECRET_KEY` | T CLOUD PUBLIC Secret Key (SK) | `a1b2c3d4...` |
| `KEEP_CERT_REGEX` | Regex matched against the certificate name. Matching certificates are kept during cleanup, even if no longer used. Default `dummy` | `^k8s-prod-.*` |
