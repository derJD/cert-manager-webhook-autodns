# ACME webhook for AutoDNS API

Solver enabling cert-manager to interact with [AutoDNS API](https://help.internetx.com/display/APIXMLEN/JSON+API+Basics).

> This Solver took heavy inspiration from [cert-manager-webhook-hetzner](https://github.com/vadimkim/cert-manager-webhook-hetzner)

## Requirements

* [go](https://golang.org/) >= 1.13.0
* [helm](https://helm.sh/) >= v3.0.0
* [kubernetes](https://kubernetes.io/) >= v1.14.0
* [cert-manager](https://cert-manager.io/) >= 0.12.0

## Installation

### cert-manager

Follow the [instructions](https://cert-manager.io/docs/installation/) using the cert-manager documentation to install it within your cluster.

### Webhook

**To install the webhook run:**

```bash
# Clone this repository and ...
helm install --namespace cert-manager cert-manager-webhook-autodns deploy/cert-manager-webhook-autodns
```

**Note**: The kubernetes resources used to install the Webhook should be deployed within the same namespace as the **cert-manager**.

**To uninstall the webhook run:**

```bash
helm uninstall --namespace cert-manager cert-manager-webhook-autodns
```

Values for customization via *values.yaml* or *--set* can be seen [here](deploy/cert-manager-webhook-autodns/values.yaml)

## Issuer

Create a `ClusterIssuer` or `Issuer` resource as following:

```yaml
apiVersion: cert-manager.io/v1alpha2
kind: ClusterIssuer
metadata:
  name: letsencrypt-staging
spec:
  acme:
    # The ACME server URL
    server: https://acme-staging-v02.api.letsencrypt.org/directory

    # Email address used for ACME registration
    email: mail@example.com # REPLACE THIS WITH YOUR EMAIL!!!

    # Name of a secret used to store the ACME account private key
    privateKeySecretRef:
      name: letsencrypt-staging

    solvers:
      - dns01:
          webhook:
            # This group needs to be configured when installing the helm package, otherwise the webhook won't have permission to create an ACME challenge for this API group.
            groupName: acme.yourdomain.tld
            solverName: autodns
            config:
              url: https://api.autodns.com/v1
              zone: example.com # (Optional): When not provided the Zone will obtained by cert-manager's ResolvedZone
              nameserver: ns1.pns.de # (Mandatory): Nameserver used for RR updates
              context: 1234567 # (Mandatory): PersonalAutoDNS Context number used for authentification
              username: example_username # (Mandatory): Username for basic auth.
              password: example_password # (Mandatory): Password for basic auth.
```

### Create a certificate

* Create an A-Record pointing to `example-fqdn.example.com` (of course you have to replace `example-fqdn.example.com`)
* Finally you can create certificates, for example:

```yaml
apiVersion: cert-manager.io/v1alpha2
kind: Certificate
metadata:
  name: example-cert
  namespace: cert-manager
spec:
  dnsNames:
    - example-fqdn.example.com
  issuerRef:
    name: letsencrypt-staging
    kind: ClusterIssuer
  secretName: example-cert
```

## Development

### Running the test suite

All DNS providers **must** run the DNS01 provider conformance testing suite,
else they will have undetermined behavior when used with cert-manager.

**It is essential that you configure and run the test suite when creating a DNS01 webhook.**

Copy [config.json.sample](testdata/autoDNS/config.json.sample) to `testdata/autoDNS/config.json`
and fill it with your actual AutoDNS authentification data and a valid zone as well as nameserver.

You can then run the test suite with:

```bash
# then run the tests
TEST_ZONE_NAME=example.com. make test
```
