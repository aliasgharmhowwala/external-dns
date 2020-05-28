# Setting up ExternalDNS for Services on UltraDNS

This tutorial describes how to setup ExternalDNS for usage within a Kubernetes cluster using UltraDNS DNS.

Make sure to use **>=0.6** version of ExternalDNS for this tutorial.

## Managing DNS with UltraDNS

If you want to read up on UltraDNS service you can read the following tutorial: 
[Introduction to UltraDNS DNS]()

Create a new DNS Zone where you want to create your records in. For the examples we will be using `example.com`

## Creating UltraDNS Credentials

The environment variable `ULTRADNS_USERNAME`,`ULTRADNS_PASSWORD`,`ULTRADNS_BASEURL`,& `ULTRADNS_ACCOUNTNAME` will be needed to run ExternalDNS with UltraDNS.

## Deploy ExternalDNS

Connect your `kubectl` client to the cluster you want to test ExternalDNS with.
Then apply one of the following manifests file to deploy ExternalDNS.

- Note: We are assuming the domain is already present at UltraDNS
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
        - --domain-filter=example.com # (optional) limit to only example.com domains; change to match the zone created above.
        - --provider=ultradns
        env:
        - name: ULTRADNS_USERNAME
          value: ""
        - name: ULTRADNS_PASSWORD
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
        - --source=service # ingress is also possible
        - --domain-filter=example.com # (optional) limit to only example.com domains; change to match the zone created above.
        - --provider=ultradns
        env:
        - name: ULTRADNS_USERNAME
          value: ""
        - name: ULTRADNS_PASSWORD
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
    external-dns.alpha.kubernetes.io/hostname: my-app.example.com
spec:
  selector:
    app: nginx
  type: LoadBalancer
  ports:
    - protocol: TCP
      port: 80
      targetPort: 80
```

Note the annotation on the service; use the same hostname as the UltraDNS DNS zone created above.

ExternalDNS uses this annotation to determine what services should be registered with DNS. Removing the annotation will cause ExternalDNS to remove the corresponding DNS records.

Create the deployment and service:

```console
$ kubectl create -f nginx.yaml
$ kubectl create -f external-dns.yaml
```

Depending where you run your service it can take a little while for your cloud provider to create an external IP for the service.

Once the service has an external IP assigned, ExternalDNS will notice the new service IP address and synchronize the UltraDNS DNS records.

## Verifying UltraDNS DNS records

Check your [UltraDNS UI](https://portal.ultradns.net/) to view the records for your UltraDNS DNS zone.

Click on the zone for the one created above if a different domain was used.

This should show the external IP address of the service as the A record for your domain.

## Cleanup

Now that we have verified that ExternalDNS will automatically manage UltraDNS DNS records, we can delete the tutorial's example:

```
$ kubectl delete service -f nginx.yaml
$ kubectl delete service -f externaldns.yaml
```
