# Shell Service

The *shell-service* is a Keptn core component and used for triggering Shell tests.

The *shell-service* listens to Keptn events of type:
- `sh.keptn.events.deployment-finished`

In case the tests succeeed, this service sends a `sh.keptn.events.test-finished` event with `pass` as `result`. In case the tests do not succeed (e.g., the error rate is too high), this service sends an `sh.keptn.events.test-finished` event with `fail` as `result`.

## Installation

The *shell-service* is installed as a part of [Keptn](https://keptn.sh).

## Deploy in your Kubernetes cluster

To deploy the current version of the *shell-service* in your Keptn Kubernetes cluster, use the file `deploy/service.yaml` from this repository and apply it:

```console
kubectl apply -f deploy/service.yaml
```

## Delete in your Kubernetes cluster

To delete a deployed *shell-service*, use the file `deploy/service.yaml` from this repository and delete the Kubernetes resources:

```console
kubectl delete -f deploy/service.yaml
```
