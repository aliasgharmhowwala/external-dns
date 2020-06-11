# Setting up ExternalDNS for Services on UltraDNS

This tutorial describes how to setup ExternalDNS for usage within a Kubernetes cluster using UltraDNS.

For this tutorial, please make sure that you are using version  **>0.7.2** (or higher) of ExternalDNS.

## Managing DNS with UltraDNS

If you would like to read-up on the UltraDNS service, you can find additional details here: [Introduction to UltraDNS](https://docs.ultradns.neustar)

Before proceeding, please create a new DNS Zone that you will create your records in for this tutorial process. For the examples in this tutorial, we will be using `example.com` as our Zone.

## Setting Up UltraDNS Credentials

The following environment variables will be needed to run ExternalDNS with UltraDNS.

`ULTRADNS_USERNAME`,`ULTRADNS_PASSWORD`, &`ULTRADNS_BASEURL`
`ULTRADNS_ACCOUNTNAME`(optional variable).

## Deploying ExternalDNS

Connect your `kubectl` client to the cluster you want to test ExternalDNS with.
Then apply one of the following manifests file to deploy ExternalDNS.

- Note: We are assuming the zone is already present at UltraDNS.
- Note: While creating CNAMES as target endpoints the `--txt-prefix` option is mandatory.
### Manifest (for clusters without RBAC enabled)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: external-dns
spec:
  strategy:
    type: Recreate
  selector:
    matchLabels:
      app: external-dns
  template:
    metadata:
      labels:
        app: external-dns
    spec:
      containers:
      - name: external-dns
        image: registry.opensource.zalan.do/teapot/external-dns:latest
        args:
        - --source=service # ingress is also possible
        - --domain-filter=example.com # (Recommended) We recommend to use this filter as it minimize the time to propagate changes, as there are less number of zones to look into..
        - --provider=ultradns
        env:
        - name: ULTRADNS_USERNAME
          value: ""
        - name: ULTRADNS_PASSWORD  # The password is required to be BASE64 encrypted.
          value: ""
        - name: ULTRADNS_BASEURL
          value: "https://api.ultradns.com/"
        - name: ULTRADNS_ACCOUNTNAME
          value: ""
```

### Manifest (for clusters with RBAC enabled)

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: external-dns
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: external-dns
rules:
- apiGroups: [""]
  resources: ["services","endpoints","pods"]
  verbs: ["get","watch","list"]
- apiGroups: ["extensions"]
  resources: ["ingresses"]
  verbs: ["get","watch","list"]
- apiGroups: [""]
  resources: ["nodes"]
  verbs: ["list"]
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: external-dns-viewer
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: external-dns
subjects:
- kind: ServiceAccount
  name: external-dns
  namespace: default
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: external-dns
spec:
  strategy:
    type: Recreate
  selector:
    matchLabels:
      app: external-dns
  template:
    metadata:
      labels:
        app: external-dns
    spec:
      serviceAccountName: external-dns
      containers:
      - name: external-dns
        image: registry.opensource.zalan.do/teapot/external-dns:latest
        args:
        - --source=service 
        - --source=ingress
        - --domain-filter=example.com #(Recommended) We recommend to use this filter as it minimize the time to propagate changes, as there are less number of zones to look into..
        - --provider=ultradns
        env:
        - name: ULTRADNS_USERNAME
          value: ""
        - name: ULTRADNS_PASSWORD # The password is required to be BASE64 encrypted.
          value: ""
        - name: ULTRADNS_BASEURL
          value: "https://api.ultradns.com/"
        - name: ULTRADNS_ACCOUNTNAME
          value: ""
```

## Deploying an Nginx Service

Create a service file called 'nginx.yaml' with the following contents:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
spec:
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - image: nginx
        name: nginx
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: nginx
  annotations:
    external-dns.alpha.kubernetes.io/hostname: my-app.example.com.
spec:
  selector:
    app: nginx
  type: LoadBalancer
  ports:
    - protocol: TCP
      port: 80
      targetPort: 80
```

Please note the annotation on the service. Use the same hostname as the UltraDNS zone created above.

ExternalDNS uses this annotation to determine what services should be registered with DNS. Removing the annotation will cause ExternalDNS to remove the corresponding DNS records.

####  Creating the deployment and service:

```console
$ kubectl create -f nginx.yaml
$ kubectl create -f external-dns.yaml
```

Depending on where you run your service, it can take a few minutes while for your cloud provider to create an external IP for the service.

Once the service has an external IP assigned, ExternalDNS will notice the new service IP address and will synchronize the UltraDNS records.

## Verifying UltraDNS records

Please verify on the [UltraDNS UI](https://portal.ultradns.neustar) that the records are created under the zone "example.com".

Select the zone that was created above (or select the appropriate zone if a different zone was used.)

This should show the external IP will be displayed as an CNAME record for your zone.

## Cleanup

Now that we have verified that ExternalDNS will automatically manage UltraDNS records, we can delete example zones that you created in this tutorial:

```
$ kubectl delete service -f nginx.yaml
$ kubectl delete service -f externaldns.yaml
```

## Creating Multiple A Records Target
- First of all create service file called 'apple-banana-echo.yaml' 
```yaml
---
kind: Pod
apiVersion: v1
metadata:
  name: apple-app
  labels:
    app: apple
spec:
  containers:
    - name: apple-app
      image: hashicorp/http-echo
      args:
        - "-text=apple"
---
kind: Service
apiVersion: v1
metadata:
  name: apple-service
spec:
  selector:
    app: apple
  ports:
    - port: 5678 # Default port for image
```
- Next, create service file called 'expose-apple-banana-app.yaml' to expose the services, for more information to deploy ingress controller please refer (https://kubernetes.github.io/ingress-nginx/deploy/)
```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: example-ingress
  annotations:
    ingress.kubernetes.io/rewrite-target: /
    ingress.kubernetes.io/scheme: internet-facing
    external-dns.alpha.kubernetes.io/hostname: apple.example.com.
    external-dns.alpha.kubernetes.io/target: 10.10.10.1,10.10.10.23
spec:
  rules:
  - http:
      paths:
        - path: /apple
          backend:
            serviceName: apple-service
            servicePort: 5678
```
- Next, create the deployment and service:
```console
$ kubectl create -f apple-banana-echo.yaml
$ kubectl create -f expose-apple-banana-app.yaml
$ kubectl create -f external-dns.yaml
```
- Depending where you run your service it can take a little while for your cloud provider to create an external IP for the service.
- Please verify on the [UltraDNS UI](https://portal.ultradns.neustar), that the records are created under the zone "example.com".
- Finally, cleanup the deployment and service, verify on the UI that those records got deleted from the zone "example.com":
```console
$ kubectl delete -f apple-banana-echo.yaml
$ kubectl delete -f expose-apple-banana-app.yaml
$ kubectl delete -f external-dns.yaml
```
## Creating CNAME Record
- Note: Before Deploying external-dns service make sure to add option `--txt-prefix=txt-` in external-dns.yaml, if not provided the records won't get created
-  First of all create service file called 'apple-banana-echo.yaml'
    - Config file (kubernetes cluster is on-premise not on cloud)
    ```yaml
    ---
    kind: Pod
    apiVersion: v1
    metadata:
      name: apple-app
      labels:
        app: apple
    spec:
      containers:
        - name: apple-app
          image: hashicorp/http-echo
          args:
            - "-text=apple"
    ---
    kind: Service
    apiVersion: v1
    metadata:
      name: apple-service
    spec:
      selector:
        app: apple
      ports:
        - port: 5678 # Default port for image
    ---
    apiVersion: extensions/v1beta1
    kind: Ingress
    metadata:
      name: example-ingress
      annotations:
        ingress.kubernetes.io/rewrite-target: /
        ingress.kubernetes.io/scheme: internet-facing
        external-dns.alpha.kubernetes.io/hostname: apple.example.com.
        external-dns.alpha.kubernetes.io/target: apple.cname.com.
    spec:
      rules:
      - http:
          paths:
            - path: /apple
              backend:
                serviceName: apple-service
                servicePort: 5678
    ```
    - Config file (Using Kuberentes cluster service from different cloud vendors)
    ```yaml
    ---
    kind: Pod
    apiVersion: v1
    metadata:
      name: apple-app
      labels:
        app: apple
    spec:
      containers:
        - name: apple-app
          image: hashicorp/http-echo
          args:
            - "-text=apple"
    ---
    kind: Service
    apiVersion: v1
    metadata:
      name: apple-service
      annotations:
        external-dns.alpha.kubernetes.io/hostname: my-app.example.com.
    spec:
      selector:
        app: apple
      type: LoadBalancer
      ports:
        - protocol: TCP
          port: 5678
          targetPort: 5678
    ```
- Next, create the deployment and service:
```console
$ kubectl create -f apple-banana-echo.yaml
$ kubectl create -f external-dns.yaml
```
- Depending where you run your service it can take a little while for your cloud provider to create an external IP for the service.
- Please verify on the [UltraDNS UI](https://portal.ultradns.neustar), that the records are created under the zone "example.com".
- Finally, cleanup the deployment and service, verify on the UI that those records got deleted from the zone "example.com":
```console
$ kubectl delete -f apple-banana-echo.yaml
$ kubectl delete -f external-dns.yaml
```
## Create Multiple Types Of Records
- Note: Before Deploying external-dns service make sure to add option `--txt-prefix=txt-` in external-dns.yaml. Since, we are also creating CNAME record, if not provided the records won't get created.
-  First of all create service file called 'apple-banana-echo.yaml'
    - Config file (kubernetes cluster is on-premise not on cloud)
    ```yaml
    ---
    kind: Pod
    apiVersion: v1
    metadata:
      name: apple-app
      labels:
        app: apple
    spec:
      containers:
        - name: apple-app
          image: hashicorp/http-echo
          args:
            - "-text=apple"
    ---
    kind: Service
    apiVersion: v1
    metadata:
      name: apple-service
    spec:
      selector:
        app: apple
      ports:
        - port: 5678 # Default port for image
    ---
    kind: Pod
    apiVersion: v1
    metadata:
      name: apple-app1
      labels:
        app: apple1
    spec:
      containers:
        - name: apple-app1
          image: hashicorp/http-echo
          args:
            - "-text=apple"
    ---
    kind: Service
    apiVersion: v1
    metadata:
      name: apple-service1
    spec:
      selector:
        app: apple1
      ports:
        - port: 5679 # Default port for image
    ---
    kind: Pod
    apiVersion: v1
    metadata:
      name: apple-app2
      labels:
        app: apple2
    spec:
      containers:
        - name: apple-app2
          image: hashicorp/http-echo
          args:
            - "-text=apple"
    ---
    kind: Service
    apiVersion: v1
    metadata:
      name: apple-service2
    spec:
      selector:
        app: apple2
      ports:
        - port: 5680 # Default port for image
      apiVersion: extensions/v1beta1
    ---
    apiVersion: extensions/v1beta1
    kind: Ingress
    metadata:
      name: example-ingress
      annotations:
        ingress.kubernetes.io/rewrite-target: /
        ingress.kubernetes.io/scheme: internet-facing
        external-dns.alpha.kubernetes.io/hostname: apple.example.com.
        external-dns.alpha.kubernetes.io/target: apple.cname.com.
    spec:
      rules:
      - http:
          paths:
            - path: /apple
              backend:
                serviceName: apple-service
                servicePort: 5678
    ---
    apiVersion: extensions/v1beta1
    kind: Ingress
    metadata:
      name: example-ingress1
      annotations:
        ingress.kubernetes.io/rewrite-target: /
        ingress.kubernetes.io/scheme: internet-facing
        external-dns.alpha.kubernetes.io/hostname: apple-banana.example.com.
        external-dns.alpha.kubernetes.io/target: 10.10.10.3
    spec:
      rules:
      - http:
          paths:
            - path: /apple
              backend:
                serviceName: apple-service1
                servicePort: 5679
    ---
    apiVersion: extensions/v1beta1
    kind: Ingress
    metadata:
      name: example-ingress2
      annotations:
        ingress.kubernetes.io/rewrite-target: /
        ingress.kubernetes.io/scheme: internet-facing
        external-dns.alpha.kubernetes.io/hostname: banana.example.com.
        external-dns.alpha.kubernetes.io/target: 10.10.10.3,10.10.10.20
    spec:
      rules:
      - http:
          paths:
            - path: /apple
              backend:
                serviceName: apple-service2
                servicePort: 5680
    ```
    - Config file (Using Kuberentes cluster service from different cloud vendors)
    ```yaml
    ---
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: nginx
    spec:
      selector:
        matchLabels:
          app: nginx
      template:
        metadata:
          labels:
            app: nginx
        spec:
          containers:
          - image: nginx
            name: nginx
            ports:
            - containerPort: 80
    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: nginx
      annotations:
        external-dns.alpha.kubernetes.io/hostname: my-app.example.com.
    spec:
      selector:
        app: nginx
      type: LoadBalancer
      ports:
        - protocol: TCP
          port: 80
          targetPort: 80
    ---
    kind: Pod
    apiVersion: v1
    metadata:
      name: apple-app
      labels:
        app: apple
    spec:
      containers:
        - name: apple-app
          image: hashicorp/http-echo
          args:
            - "-text=apple"
    ---
    kind: Service
    apiVersion: v1
    metadata:
      name: apple-service
    spec:
      selector:
        app: apple
      ports:
        - port: 5678 # Default port for image
    ---
    kind: Pod
    apiVersion: v1
    metadata:
      name: apple-app1
      labels:
        app: apple1
    spec:
      containers:
        - name: apple-app1
          image: hashicorp/http-echo
          args:
            - "-text=apple"
    ---
    kind: Service
    apiVersion: v1
    metadata:
      name: apple-service1
    spec:
      selector:
        app: apple1
      ports:
        - port: 5679 # Default port for image
    ---
    kind: Ingress
    metadata:
      name: example-ingress
      annotations:
        ingress.kubernetes.io/rewrite-target: /
        ingress.kubernetes.io/scheme: internet-facing
        external-dns.alpha.kubernetes.io/hostname: apple.example.com.
        external-dns.alpha.kubernetes.io/target: 10.10.10.3,10.10.10.25
    spec:
      rules:
      - http:
          paths:
            - path: /apple
              backend:
                serviceName: apple-service
                servicePort: 5678
    ---
    apiVersion: extensions/v1beta1
    kind: Ingress
    metadata:
      name: example-ingress1
      annotations:
        ingress.kubernetes.io/rewrite-target: /
        ingress.kubernetes.io/scheme: internet-facing
        external-dns.alpha.kubernetes.io/hostname: apple-banana.example.com.
        external-dns.alpha.kubernetes.io/target: 10.10.10.3
    spec:
      rules:
      - http:
          paths:
            - path: /apple
              backend:
                serviceName: apple-service1
                servicePort: 5679
    ```
- Next, create the deployment and service:
```console
$ kubectl create -f apple-banana-echo.yaml
$ kubectl create -f external-dns.yaml
```
- Depending where you run your service it can take a little while for your cloud provider to create an external IP for the service.
- Please verify on the [UltraDNS UI](https://portal.ultradns.neustar), that the records are created under the zone "example.com".
- Finally, cleanup the deployment and service, verify on the UI that those records got deleted from the zone "example.com":
```console
$ kubectl delete -f apple-banana-echo.yaml
$ kubectl delete -f external-dns.yaml```
