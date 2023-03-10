# olivetree
Toy controller for Zack and Dean


For Talos, we relax the security policy on our namespace by patching it to add the `pod-security.kubernetes.io/enforce=privileged` annotation.  This is in `config/manager/`.

## DAOS commands

Start the server on each node.

```console
/usr/bin/daos_server start -o /etc/daos-server/daos_server.yml --json-logging &
```

Format the storage.

```console
dmg -o /etc/daos-server/daos_control.yml --json-logging storage format
dmg -o /etc/daos-server/daos_control.yml system query -v
```

Start the agent on each node.

```console
/usr/bin/daos_agent -o /etc/daos-agent/daos_agent.yml --json-logging &
```

Create a pool.

```console
dmg -o /etc/daos-server/daos_control.yml pool create --size=11G autotest_pool
dmg -o /etc/daos-server/daos_control.yml pool query autotest_pool
daos pool query autotest_pool
daos pool autotest --pool autotest_pool
```

Note that the `daos` command above didn't need a reference to `/etc/daos-server` to find the pool information.

The args:

Stdout/stderr in json format: `--json-logging`.  But that doesn't use json in the log files in /tmp/.

The `--json` arg is like `--json-logging`, but is more for debugging.



## Pod networking

https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/

```console
[droehrich@dill04 capi-3]$ kubectl get pods -o wide -n olivetree-system
NAME                                 READY   STATUS    RESTARTS   AGE   IP           NODE            NOMINATED NODE   READINESS GATES
olivetree-controller-manager-sm8nx   2/2     Running   0          21s   10.244.1.6   talos-glk-ixb   <none>           <none>
olivetree-controller-manager-v2qvn   2/2     Running   0          21s   10.244.2.2   talos-9tm-gbt   <none>           <none>
```

Then check pod DNS:

```console
: dean-mbp; KUBECONFIG=~/kubeconfig-deanplay2 kubectl exec -it -n olivetree-system   olivetree-controller-manager-v2qvn -- bash
[root@olivetree-controller-manager-v2qvn /]# curl 10-244-1-6.olivetree-system.pod.cluster.local
curl: (7) Failed to connect to 10-244-1-6.olivetree-system.pod.cluster.local port 80: Connection refused
```

So we know we can look up the other pod's DNS name.

