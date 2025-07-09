# Bitwarden CLI Webhook

This container image runs the Bitwarden CLI (`bw`) and exposes it as a simple REST API webhook on port 8087. It is
designed to be used as a webhook provider for the [External Secrets Operator](https://external-secrets.io/latest/) (ESO)
in Kubernetes, allowing you to sync secrets from a self-hosted Vaultwarden instance.

The entrypoint is a small Go program that handles the login and unlock flow before starting the bw serve process.

## ✨ Security Highlights

This image is built with security as a top priority, adhering to modern best practices for containerization.

**Distroless**: The final image is built on a Google distroless base. It contains only the application and its runtime
dependencies. It does not include a shell, package manager, or other standard utilities, which dramatically reduces the
attack surface and eliminates a huge number of potential CVEs.

**Rootless**: The container process runs as a non-privileged nonroot user by default. This prevents a whole class of
container-breakout vulnerabilities and ensures that even if the application were compromised, the attacker would not
have root access within the container.

## 🚀 Usage

This image is intended to be run as a service inside a Kubernetes cluster.

Kubernetes Manifest
Here is an example Deployment that uses this image and exposes it as a service for ESO.

```YAML
apiVersion: apps/v1
kind: Deployment
metadata:
    name: bitwarden-cli
    namespace: external-secrets # Or your preferred namespace
    labels:
        app: bitwarden-cli
spec:
    replicas: 1
    selector:
        matchLabels:
            app: bitwarden-cli
    template:
        metadata:
            labels:
                app: bitwarden-cli
        spec:
            containers:
                - name: bitwarden-cli
                  image: ghcr.io/vehkiya/bw-cli:latest # 👈 Image here
                  ports:
                      - containerPort: 8087
                  env:
                      - name: BW_HOST
                        value: "https://vaultwarden.your.domain"
                      - name: BW_CLIENTID
                        valueFrom:
                            secretKeyRef:
                                name: vaultwarden-credentials
                                key: BW_CLIENTID
                      - name: BW_CLIENTSECRET
                        valueFrom:
                            secretKeyRef:
                                name: vaultwarden-credentials
                                key: BW_CLIENTSECRET
                      - name: BW_PASSWORD
                        valueFrom:
                            secretKeyRef:
                                name: vaultwarden-credentials
                                key: BW_PASSWORD
                    # - name: DEBUG_ENABLED
                    #   value: "true" # Optional: for verbose logging
---
apiVersion: v1
kind: Service
metadata:
    name: bitwarden-cli-service
    namespace: external-secrets
spec:
    selector:
        app: bitwarden-cli
    ports:
        - protocol: TCP
          port: 8087
          targetPort: 8087
```

## 🔧 Environment Variables

The container is configured using the following environment variables.

| Variable        | Description                                                | Required |
|-----------------|------------------------------------------------------------|----------|
| BW_HOST         | The full URL of your Vaultwarden instance.                 | Yes      |
| BW_CLIENTID     | The API Key Client ID from your Vaultwarden account.       | Yes      |
| BW_CLIENTSECRET | The API Key Client Secret from your Vaultwarden account.   | Yes      |
| BW_PASSWORD     | Your master password, used to unlock the vault.            | Yes      |
| DEBUG_ENABLED   | If set to any value (e.g., "true"), enables debug logging. | No       |

## 🛠️ Building the Image

To build the image locally, use the provided Dockerfile.

```Bash
# Build and tag the image
docker build -t ghcr.io/<your_github_user>/bw-cli:latest .

# Push the image to the GitHub Container Registry
docker push ghcr.io/<your_github_user>/bw-cli:latest
```