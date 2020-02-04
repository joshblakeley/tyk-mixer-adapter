# tyk-mixer-adapter
Custom Istio Mixer Authorization Adapter For Policy Enforcement Using Tyk API Gateway


## How it works

This is an adapter for the [Istio Mixer component](https://istio.io/docs/reference/config/policy-and-telemetry/mixer-overview/) which invokes the Tyk Istio Mixer Adapter.

Tyk API Gateways can then define access control, rate limiting and quotas for several different authentication scenarios based on receiving user defined headers, and other mesh information passed to the adapter by Mixer.

## Istio Prerequsites

* k8s cluster running any Istio (1.1+) sample app i.e.
  - [Helloworld](https://github.com/istio/istio/tree/master/samples/helloworld)
  - [Bookinfo](https://istio.io/docs/examples/bookinfo/)

>> Note, in `istio-1.1`, [policy checks are disabled](https://istio.io/docs/reference/config/installation-options/).

While setting up the cluster using the instructions above, set the value 
```
--set global.disablePolicyChecks=false
```

## Tyk Prerequisites

* Install the Tyk Deployment you need into k8s using our [Official Helm Charts](https://github.com/TykTechnologies/tyk-helm-chart)

* In your Tyk Dashboard import functionality or via the [Rest API](https://www.tyk.io/docs/tyk-dashboard-api/api-definitions/#create-api-definition) define APIs in Tyk that will map to the service names in your istio cluster. 
For example, when deploying the Istio helloworld app the servicename is `helloworld`. Therefore, there must be an API loaded into Tyk with that listenpath i.e. http(s)://{GATEWAY_SERVICE}:8080/helloworld/

There are two example definitions in the `samples` folder of this repository that will set up an externally facing API listening on `helloworld` that routes internally to a second API that will return a [mock response](https://tyk.io/docs/advanced-configuration/transform-traffic/endpoint-designer/#mock-response) when it is successfully called via the external API - we dont use a mock response int he first API as it will prevent collecting analytics data for that API.

If the public facing API is accessed with a key that is unauthorized/rate limited or quota limited then the relevant response code will be returned. If the auth/rl/q step is successful then the internal API returns a 200 code (this is configurable on the mock response middleware).

 * The auth mode for the API in Tyk can be any that utilises Auth headers incoming to the gateway - [Bearer Token](https://www.tyk.io/docs/basic-config-and-security/security/your-apis/bearer-tokens/), [JSON Web Tokens](https://www.tyk.io/docs/basic-config-and-security/security/your-apis/json-web-tokens/), [Open ID Connect](https://www.tyk.io/docs/basic-config-and-security/security/your-apis/openid-connect/), [Oauth2](https://www.tyk.io/docs/basic-config-and-security/security/your-apis/oauth-2-0/) and [Basic Authentication](https://www.tyk.io/docs/basic-config-and-security/security/your-apis/basic-auth/).

* Define a [security policy](https://tyk.io/docs/try-out-tyk/tutorials/create-security-policy/#a-namewithdashboardatutorial-create-a-security-policy-with-the-dashboard) in Tyk that will apply to your exposed API in Tyk.



## Running the Adapter

Apply the adapter service config:

```
apiVersion: v1
kind: Service
metadata:
  name: tykgrpcadapterservice
  namespace: istio-system
  labels:
    app: tykgrpcadapter
spec:
  type: ClusterIP
  ports:
    - name: grpc
      protocol: TCP
      port: 5000
      targetPort: 5000
  selector:
    app: tykgrpcadapter
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: tykgrpcadapter
  namespace: istio-system
  labels:
    app: tykgrpcadapter
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: tykgrpcadapter
      annotations:
        sidecar.istio.io/inject: "false"
    spec:
      containers:
        - name: tykgrpcadapter
          image: joshtyk/tyk-istio-adapter:latest
          imagePullPolicy: Always
          ports:
            - containerPort: 5000
```

`kubectl apply -f adapter_service.yaml`

## Adapter configuration

TODO


## Deploy configuration state for the adapter to Istio

Please make sure you have Istio cluster setup running the sample helloworld or BookInfo

First setup the attributes maps and deploy them from the cloned repo:

`kubectl apply -f testdata/attributes.yaml -f testdata/template.yaml`

Deploy the state for the adapter

`kubectl apply -f testdata/tykgrpcadapter.yaml`


Deploy the config:
`kubectl apply -f testdata/sample_operator_cfg.yaml`


you should now see a connection established on the mixer logs:
```
$ kubectl -n istio-system logs $(kubectl -n istio-system get pods -lapp=mixer -o jsonpath='{.items[0].metadata.name}') -c mixer
2020-01-28T17:59:49.249312Z	info	grpcAdapter	Connected to: tykgrpcadapterservice:5000
2020-01-28T17:59:49.249312Z	info	ccResolverWrapper: sending new addresses to cc: [{tykgrpcadapterservice:5000 0  <nil>}]
2020-01-28T17:59:49.249312Z	info	ClientConn switching balancer to "pick_first"
2020-01-28T17:59:49.249312Z	info	pickfirstBalancer: HandleSubConnStateChange: 0xc4211e2cb0, CONNECTING
2020-01-28T17:59:49.249312Z	info	pickfirstBalancer: HandleSubConnStateChange: 0xc4211e2cb0, READY
```

## Validate things are working

1. Check tyk dashboard for Analytics data relating to the calls to your setup APIs
2. Check adapter logs for returned codes from tyk and details about what endpoints the adapter is trying to call in Tyk.




# References

https://istio.io/docs/concepts/policies-and-telemetry/#adapters
https://github.com/salrashid123/istio_custom_auth_adapter 
https://github.com/istio/istio/wiki/Mixer-Out-Of-Process-Adapter-Walkthrough
https://venilnoronha.io/set-sail-a-production-ready-istio-adapter
https://istio.io/help/ops/setup/validation/


