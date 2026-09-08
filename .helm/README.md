# Helm Chart: T Cloud ELB Cert Sync

Dieses Helm-Chart installiert den **T Cloud ELB Cert Sync "Controller"** in deinem Kubernetes-Cluster. 
Der Controller benötigt clusterweite Lese-Rechte (`ClusterRole`) für Secrets, um Zertifikatsänderungen in allen Namespaces erkennen zu können.

## Usage

# On commandline:
```bash
helm repo add --username gitlab+deploy-token-XXX  --password <access_token> <project-name> https://git.mms-support.de/api/v4/projects/<projectID>/packages/helm/stable

helm install <my-release> <project-name>/mychart
or
helm pull <my-release> <project-name>/mychart

```

# Within TerrAnsible
```yaml
...
    helm_charts:

      - name: "certificate"
        ref: "certificate/certificate"
        url: "https://git.mms-support.de/api/v4/projects/2758/packages/helm/stable"
        username: "{{ k8s_secrets.registries['aoc-gitlab-7009-helm'].username }}"
        password: "{{ k8s_secrets.registries['aoc-gitlab-7009-helm'].password }}"
        release_values:
          secretTemplate:
            annotations:
              t-cloud.telekom.com/listener-id: "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXX"
          duration: 8760h
          renewBefore: 645h
          fullnameOverride: "[certificate-name]"
          secretName: "[secret-name]"
          commonName: "www.domain.tld"
          dnsNames:
            - domain.tld
            - *.domain.tld
          privateKey: {size: 4096, rotationPolicy: Always, algorithm: RSA}
        deploy_tags: "certs certificates"

      - name: "t-cloud-cert-sync"
        ref: "t-cloud-cert-sync/t-cloud-cert-sync"
        url: "https://git.mms-support.de/api/v4/projects/2984/packages/helm/stable"
        username: "{{ k8s_secrets.registries['aoc-gitlab-7009-helm'].username }}"
        password: "{{ k8s_secrets.registries['aoc-gitlab-7009-helm'].password }}"
        release_values:
          imagePullSecrets: [{name: "aoc-gitlab-7009-03"}]
          config:
            tCloudRegion: "{{ (tf_providers.opentelekomcloud.tenant_name).split('_')[0] }}"
            tCloudDomainName: "{{ tf_providers.opentelekomcloud.domain_name }}"
            tCloudProjectId: "{{ terraform_project_id }}"
          credentials:
            existingSecret: {name: "t-cloud-credentials"}
        deploy_tags: "certs certificates t-cloud-cert-sync"
...
```

Also a secret is needed, containing the T Cloud credentials with the following keys: `OS_ACCESS_KEY` and `OS_SECRET_KEY`. You may reuse the AK/SK Credentials from your Terraform provider configuration. The secret must be created in the same namespace where the controller is deployed, and the controller will read the credentials from there.<br>
Example:

```yaml
...
[clustername]:
...
  [namespace]_secrets:
    - name: "t-cloud-credentials"
      deploy_tags: "appstack frontend portal nrw-portal certs certificates"
      entries_list:
        - {name: "OS_ACCESS_KEY", value: "{{ tf_providers.opentelekomcloud.access_key }}"}
        - {name: "OS_SECRET_KEY", value: "{{ tf_providers.opentelekomcloud.secret_key }}"}
...
```
