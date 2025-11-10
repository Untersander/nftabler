# NFTabler

Small wrapper around [nft](https://netfilter.org/projects/nftables/) that watches a directory of nftables config files and applies them automatically when they change.

## Features
- Watches a directory for changes (create/modify/remove).
- Applies changed files with `nft -f` (or an equivalent apply command).
- Minimal logging to stdout/stderr.

## Why use it
- Keep nftables rules as discrete files in a directory and let the tool keep kernel state in sync.
- Good for GitOps / configuration-managed firewall workflows.

## Quick usage
- Place nft syntax files (examples below) in `/etc/nftabler/`, then run nftabler pointing at that directory.
- nftabler will read each file and run `nft -f <file>` (or the configured apply command) when a file is created or changed.
- Files need to have the file extension `.nft`.
- Make sure to validate your rules and other tools using nftables before deploying, you might break other tools if you don't.

Example file layout
- /etc/nftabler/
    - 10-base.nft
    - 20-ssh.nft
    - 30-blocklists.nft

Example nft file (/etc/nftabler/10-base.nft)
```
flush table ip lannat
define lan_cidr = 10.0.0.0/20
define wan_iface = "eth0"
table ip lannat {
    chain POSTROUTING {
    type nat hook postrouting priority srcnat; policy accept;
    oif $wan_iface ip saddr $lan_cidr ip daddr != $lan_cidr masquerade
    }

    chain OUTPUT {
    type nat hook output priority dstnat; policy accept;
    }

    chain PREROUTING {
    type nat hook prerouting priority dstnat; policy accept;
    }

}
```

### Running in Docker
- The container must be allowed to manipulate networking on the host. Typical options:
    - run in host network namespace (--network=host) so nft commands affect the host.
    - grant NET_ADMIN capability.
    - bind-mount the directory with configs.

Example:
```
docker run -d --name nftabler \
    --network=host \
    --cap-add=NET_ADMIN \
    -v <local-nft-rule-dir>:/etc/nftabler:ro \
    ghcr.io/untersander/nftabler:latest
```

### Running in Kubernetes
- Typical deployment is a DaemonSet so each node runs an instance and applies rules for that node.
- Config files are usually provided via a ConfigMap mounted into the Pod.
- The Pod must share the host network namespace (hostNetwork: true) so nft operations affect the host.
- The Pod usually needs elevated privileges (NET_ADMIN capability and root).

Example DaemonSet/ConfigMap (minimal, adjust policies to your cluster):
```
---
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: nftabler
  namespace: nftables
spec:
  selector:
    matchLabels:
      app: nftabler
  template:
    metadata:
      labels:
        app: nftabler
    spec:
      hostNetwork: true
      containers:
      - name: nft-test-container
        image: ghcr.io/untersander/nftabler:latest
        imagePullPolicy: Always
        securityContext:
          capabilities:
            add: ["NET_ADMIN"]
            drop: ["ALL"]
        volumeMounts:
        - mountPath: /etc/nftabler
          name: nft-config
        - mountPath: /run/xtables.lock
          name: xtables-lock
      volumes:
      - configMap:
          name: nft-config
          optional: true
        name: nft-config
      - hostPath:
          path: /run/xtables.lock
          type: FileOrCreate
        name: xtables-lock
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: nft-config
  namespace: nftables
data:
  lannat.nft: |
    add table ip lannat
    flush table ip lannat
    define lan_cidr = 10.0.0.0/20
    define wan_iface = "enp2s0f0np0"
    table ip lannat {
      chain POSTROUTING {
        type nat hook postrouting priority srcnat; policy accept;
        oif $wan_iface ip saddr $lan_cidr ip daddr != $lan_cidr masquerade
      }

      chain OUTPUT {
        type nat hook output priority dstnat; policy accept;
      }

      chain PREROUTING {
        type nat hook prerouting priority dstnat; policy accept;
      }

    }
```

## Security considerations
- nftabler invokes nft and therefore can fully change host firewall state. Restrict who can modify the watched directory.
- In Kubernetes, restrict RBAC and who can edit the DaemonSet and ConfigMap.
- Audit config files before deploying to production.

## Troubleshooting
- Check container logs (docker logs, kubectl logs) for errors from nft (syntax errors, permission issues).
- Verify nft binary is compatible with host kernel.
- On Kubernetes, verify the DaemonSet is scheduled on nodes with the required privileges and that hostPath mounts are correct.

## Future improvements
- Check config files before trying to apply them.
- More advanced configuration options (e.g. apply command, file extensions).
- Improved debouncing of rapid file changes
- Maybe option to only apply changed files instead of all files on any change (currently all files are applied on any new rule file change to make behavior more predictable).

## Contributing
- Open issues / PRs in the repository for bug reports or feature requests.
