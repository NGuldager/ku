# Configuration

Customize the left sidebar with `~/.config/kli/config.yaml`.

## Quick Start

```bash
kli config init          # write the default config to populate from
kli config init --force  # overwrite an existing config with the defaults
kli config path          # print the config file location
```

After seeding the file, edit it and restart `kli`. The running TUI reads config
once at startup and never writes it.

## Files

| File | Purpose |
| --- | --- |
| `~/.config/kli/config.yaml` | user-authored config file |
| `~/.config/kli/state.json` | auto-saved context and namespace state |

The config file is separate from session state. `config.yaml` is only written by
you or by `kli config init`; `state.json` is managed automatically.

## Sidebar

Today the config customizes the left sidebar menu. When a `sidebar:` list is
present it replaces the built-in default menu. Without a config file the built-in
defaults are used.

The Overview entry is always available. Resources the cluster does not expose
are dropped, and empty sections are hidden.

```yaml
sidebar:
  - section: Workloads
    items:
      - { label: Pods, resource: pods }
      - { label: Deployments, resource: deployments }
      - { label: HPAs, resource: horizontalpodautoscalers }
      - { label: ScaledObjects, resource: scaledobjects }
  - section: Network
    items:
      - { label: Services, resource: services }
```

The `resource` field accepts anything the resource picker resolves: a plural,
singular, kind, short name, or group-qualified key, such as
`scaledobjects.keda.sh`.

## Built-In Defaults

The default sidebar includes Pods, Deployments, StatefulSets, DaemonSets,
ReplicaSets, Jobs, CronJobs, Services, Ingresses, Endpoints, ConfigMaps,
Secrets, ServiceAccounts, PVCs, PVs, StorageClasses, Nodes, Namespaces, and
Events.

HPAs, KEDA ScaledObjects, and OpenTelemetry collectors are not in the default
menu. Add them to `sidebar:` when your cluster exposes them. A freshly seeded
config lists them as commented opt-in examples.
