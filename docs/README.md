# kli docs

A keyboard-driven Kubernetes TUI. These pages cover how to install and run it,
configure the sidebar, use the keys, and understand each feature.

- [Getting started](getting-started.md) - install, run, flags, config, upgrade.
- [Keybindings](keybindings.md) - every key, by context.
- [Features](features.md) - cockpit, tables, config, YAML, logs, shell, actions.

kli uses your default kubeconfig (`$KUBECONFIG`, then `~/.kube/config`) and the
current context unless you pass `--context`.

Created by [x.com/iamdothash](https://x.com/iamdothash).
