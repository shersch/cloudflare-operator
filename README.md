# Cloudflare Operator

The Cloudflare Operator is a Kubernetes operator designed to manage Cloudflare DNS records directly from Kubernetes. This allows automatic management of DNS records based on declarative configurations within your Kubernetes cluster.

## Features

- **DNS Record Management**: Automatically handle the lifecycle of DNS records with Kubernetes resources.
- **Custom Resource Definitions (CRDs)**: Easily define DNS records using Kubernetes CRDs.
- **Security**: Integrates with Kubernetes RBAC to control access to DNS record management.

## Prerequisites

- Kubernetes 1.16+
- Helm 3.2.0+

## Installation

### Using Helm

This operator is packaged as a Helm chart. To install it, you first need to add the Helm repository:

```bash
helm repo add cloudflare-operator https://YOUR_GITHUB_USERNAME.github.io/YOUR_REPOSITORY/
helm repo update
```

Then, you can install the chart with the release name my-cloudflare-operator:
```
helm install my-cloudflare-operator cloudflare-operator/cloudflare-operator
```

## Configuration
Modify the values.yaml file or specify configuration parameters using --set during installation. Key parameters include:

- image.repository: The image repository of the Cloudflare operator
- image.tag: The tag of the Docker image to use
- replicaCount: Number of replicas of the operator to run

Example:
```
helm install my-cloudflare-operator cloudflare-operator/cloudflare-operator --set replicaCount=2
```

## Usage
After installation, you can create DNSRecord resources. Here is an example of a DNSRecord:
```
apiVersion: cloudflare.local.dev/v1alpha1
kind: DNSRecord
metadata:
  name: example-dnsrecord
spec:
  zone: "example.com"
  name: "test"
  type: "A"
  content: "192.0.2.1"
  ttl: 3600
  proxied: false
```

Apply this configuration using kubectl:
```
kubectl apply -f example-dnsrecord.yaml
```

## Contributing
This has been an immense learning experience for me.  I am sure that more seasoned developers may find ways to improve on what is here.  Any contributions are greatly appreciated.
1. Fork the Project
2.	Create your Feature Branch (git checkout -b feature/AmazingFeature)
3.	Commit your Changes (git commit -m 'Add some AmazingFeature')
4.	Push to the Branch (git push origin feature/AmazingFeature)
5.	Open a Pull Request

## License

Distributed under the MIT License. See LICENSE for more information.

## Contact
Steve Hersch - steve@hersch.xyz