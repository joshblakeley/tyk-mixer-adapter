# tyk-mixer-adapter
Custom Istio Mixer Authorization Adapter For Policy Enforcement Using Tyk API Gateway


## How it works

Service to service policies will be offloaded to mixer which then invokes the Tyk Istio Mixer Adapter.
Tyk API Gateway can then action access control, rate limiting and quotas for several different authentication scenarios based on receiving an API Key, target service path and method (also other important data such as version).
You get a choice of bearer token or JWT to start with - in theory Oauth2 and OIDC will also be possible but need testing.
Tyk will be able to give access denial or an ok signal to Mixer to control the mesh traffic.

## Prerequsites

* On your local system:
  - go `1.13`
  - protoc ```libprotoc 3.6.1```
  - docker

* k8s cluster running any Istio (1.0+) sample app
  - [Istio "HelloWorld"](https://github.com/salrashid123/istio_helloworld)
    - I'd suggest following [these instructions](https://github.com/salrashid123/istio_helloworld#create-a-110-gke-cluster-and-bootstrap-istio)
  - [BookInfo](https://istio.io/docs/examples/bookinfo/)

>> Note, in `istio-1.1`, [policy checks are disabled](https://istio.io/docs/reference/config/installation-options/).

While setting up the cluster using the instructions above, set the value 
```
--set global.disablePolicyChecks=false
```
