# Wider

<img alt="wider.png" height="250" src="logo/wider.png" width="250"/>

Get pods and associated node information. Extend the output with custom-columns by leveraging keys from pod and node specs. Use the standard -n or -l for namespace or label filters.

Pod details remains top level, node information is added under `.node` key. 

## Installation
This is temporary until plugin is added to krew index

- `kubectl krew index add boriscosic https://github.com/boriscosic/wider.git`
- `kubectl krew update`
- `kubectl krew install boriscosic/wider`

## Examples
- `kubectl wider`
- `kubectl wider -n istio-system -o custom-columns="NAME:.metadata.name,NODE:.node.metadata.name`
- `kubectl wider -l app=istio-gateway -n istio-system`
- `kubectl wider -o custom-columns="NAME:.metadata.name,NODE:.node.metadata.name,IP:.status.podIP,ZONE:.node.metadata.labels.topology\.kubernetes\.io/zone" -n kube-system -l k8s-app=kube-dns`

```
POD                    NODE                                        PROVIDER ID
istio-gateway-xxx-aaa  ip-aa-bb-cc-ee.us-west-2.compute.internal   aws:///us-west-2c/i-aaaa
istio-gateway-xxx-bbb  ip-aa-bb-cc-dd.us-west-2.compute.internal   aws:///us-west-2a/i-bbbb
```
