apiVersion: cloudflare.local.dev/v1alpha1
kind: DNSRecord
metadata:
  name: camel-dnsrecord
  namespace: cloudflare-operator-system
spec:
  zone: "sprezzatura.consulting"
  name: "camel"
  type: "CNAME"
  content: "google.com"
  ttl: 3600
  proxied: true
---
apiVersion: cloudflare.local.dev/v1alpha1
kind: DNSRecord
metadata:
  name: test-dnsrecord
  namespace: cloudflare-operator-system
spec:
  zone: "sprezzatura.consulting"
  name: "test"
  type: "CNAME"
  content: "google.com"
  ttl: 3600
  proxied: false