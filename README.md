<p align="center">
  <a href="https://mdxjs.com">
    <img alt="wider.png" height="250" src="logo/wider.png" width="250"/>
  </a>
</p>

# kubectl wider

A kubectl plugin to extend pod output with attached relationships. Extend the output with custom-columns by leveraging keys from pod and node specs. Use the standard -n or -l for namespace or label filters.

Supports extensions on node, service account and pvc.

## Custom columns

Use them like you would when retrieving a resource, except, add a resource for prefix.

For example:

Before:
`kubectl get nodes -o custom-columns="NAME:.metadata.name,OS:.metadata.labels.kubernetes\.io/os"`

```
NAME      OS
homek8s   linux
homek8s   linux
homek8s   linux
```

After (with .node):
`kubectl wider -o custom-columns="NAME:.node.metadata.name,OS:.node.metadata.labels.kubernetes\.io/os"`

```
NAME      OS
homek8s   linux
homek8s   linux
homek8s   linux
```

The above example is not great because because we're still retrieving a single resource. 
To retrieve details across multiple resources, for example, what OS do my pods run on, you would use:
`kubectl wider -o custom-columns="POD:.pod.metadata.name,NODE:.node.metadata.name,OS:.node.metadata.labels.kubernetes\.io/os"`

```
POD                       NODE      OS
pod-6cfd57b89f-4mcgm      homek8s   linux
pod-665994f986-wqnxl      homek8s   linux
pod-86dc786d97-mgtb6      homek8s   linux
```

Supported resources:
- `.node`
- `.pod`
- `.serviceAccount` or `.sa`
- `.pvc` or `.pvcs`

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
