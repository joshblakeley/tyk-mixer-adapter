# tyk-mixer-adapter
Custom Istio Mixer Authorization Adapter For Policy Enforcement Using Tyk API Gateway

+-----------------------------+
|                             |
|                             |            +----------------+
|                             |            |                |
|                             |            |                |
|                             |            |   Mixer        |  Istio coducts its
|       Istio                 +------------+                |  policy enforcement
|       Service               |            |                |  via Mixer component
|       Mesh                  |            |                |
|                             |            +--------+-------+
|                             |                     |
|                             |            +--------+-------+
|                             |            |                |  Mixer calls Tyk
|                             |            |  Tyk Adapter   |  adapter
|                             |            |                |
|                             |            +--------+-------+
|                             |                     |
|                             |                     |
|                             |                     |
|                             |            +--------+-------+
|                             |            |                |
+-----------------------------+            |                |  Tyk can enforce access control
                                           |    Tyk         |  quotas and rate limiting on a
                                           |                |  per service and per method
                                           |                |  basis.
                                           |                |
                                           +----------------+



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
