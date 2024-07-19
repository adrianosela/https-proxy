# https-proxy

A simple demo HTTPS proxy. This repo has been set-up to reproduce an issue with `kubectl` not supporting `exec` over HTTPS proxies - highlighted in https://github.com/kubernetes/kubernetes/issues/126134

## Kubernetes HTTPS Proxy Set-Up

1) Hard-code a dns entry for `kubernetes-https-proxy-test.com` pointing to `127.0.0.1` in your hosts file e.g.

```
19:53 $ cat /etc/hosts

# This will result in DNS requests for kubernetes-https-proxy-test.com
# resolving to 127.0.0.1 (localhost), where our HTTPS proxy is being served.
127.0.0.1 kubernetes-https-proxy-test.com

##
# Host Database
#
# localhost is used to configure the loopback interface
# when the system is booting.  Do not change this entry.
##
127.0.0.1	localhost
255.255.255.255	broadcasthost
::1             localhost
```

2) Set `proxy-url` for your cluster's configuration in your `~/.kube/config`

> Note how I have proxy url set to `https://kubernetes-https-proxy-test.com:8443`, you should set yours to that exact value as well

```
19:54 $ cat ~/.kube/config
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: LS0tLS1CRUdJTi....
    server: https://127.0.0.1:59591
    proxy-url: https://kubernetes-https-proxy-test.com:8443
  name: kind-kind
contexts:
- context:
    cluster: kind-kind
    user: kind-kind
  name: kind-kind
current-context: kind-kind
kind: Config
preferences: {}
users:
- name: kind-kind
  user:
    client-certificate-data: LS0tLS1CRUdJTiBDR....
    client-key-data: LS0tLS1CRUdJTiBSU0Eg....
```

3) Override `certificate-authority-data` for your cluster to include the CA used to sign the certificate for the proxy server

- The `certificate-authority-data` setting is just base64 encoded newline-delimeted PEM blocks
- You can find the pem file for the proxy server's CA in `_test_certs_/ca.pem`
- If you have `yq` (yaml parser analogous to jq for json, on mac `brew install yq`), you can run the following steps

3.1) Write the current base64 decoded PEM to a file (note that I'm doing it for the cluster at index 0 in the `~/.kube/config` yaml)

```
cat ~/.kube/config | yq .clusters[0].cluster.certificate-authority-data | base64 -d > current_value.pem
```

3.2) Append the CA certificate as PEM to `current_value.pem`

```
cat _test_certs_/ca.pem >> current_value.pem
```

3.3) Base64 encode the contents of `current_value.pem`

```
cat current_value.pem | base64
```

> Note: you can copy it to clipboard with `cat current_value.pem | base64 | pbcopy`

3.4) Set the base64 encoded value as the value of `certificate-authority-data` for the cluster
  
4) Run the proxy with `go run main.go`

5) Run `kubectl` as you normally would

---

If you are using `kubectl` >= v1.30, you MUST set the env `KUBECTL_REMOTE_COMMAND_WEBSOCKETS` to `false` or otherwise you will get the error `error: proxy: unknown scheme: https`.

Sample output:

```
20:06 $ kubectl exec -it alpine-6b45654984-dt54w -- sh
error: proxy: unknown scheme: https
```

```
20:06 $ KUBECTL_REMOTE_COMMAND_WEBSOCKETS=false kubectl exec -it alpine-6b45654984-dt54w -- sh
/ # echo hello world!
hello world!
```