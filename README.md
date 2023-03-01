# olivetree
Toy controller for Zack and Dean


For Talos, relax the security policy on our namespace, after it has been created:

```console
kubectl label ns olivetree-system pod-security.kubernetes.io/enforce=privileged
```

