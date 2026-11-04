# openshift-install-tui

An interactive terminal UI for driving the OpenShift installer, built with
[Bubble Tea](https://github.com/charmbracelet/bubbletea).

All commands run in-process through the installer's asset store and command
packages. The TUI does not shell out to `openshift-install`.

Current scope:

| Stage | Status |
|---|---|
| Install Config | Collected in the TUI form, then written and validated through the asset store |
| Manifests | Wired in-process via the asset store |
| Ignition Configs | Wired in-process via the asset store |
| Install Cluster | Runs in-process with a live component diagram and logs |
| Gather | Runs the bootstrap gather in-process |
| Wait for | Runs bootstrap/install completion waits in-process |
| Explain | Browses the InstallConfig schema left-to-right and renders the CLI explain output |

The TUI starts with a command picker for the main installer workflows. Install
is wired through the stage flow above; Gather, Wait for, and Explain call the
installer packages directly; the remaining non-install commands are placeholders.

Analyze is intentionally a placeholder while the asset-store wiring is tested.

## Build

From the repository root:

```sh
./hack/build-installer-tui.sh
```

This writes `bin/openshift-install-tui` (the repo's `bin/` is gitignored). The
script follows the same shape as `hack/build.sh` and supports `MODE=release`
(default) / `MODE=dev` and an `OUTPUT` override:

```sh
MODE=dev OUTPUT=/tmp/openshift-install-tui ./hack/build-installer-tui.sh
```

Or build directly:

```sh
go build -o bin/openshift-install-tui ./cmd/openshift-install-tui
```

## Run

```sh
./bin/openshift-install-tui
```

No external installer binary is required.

## What it does

1. **Asks what you want to do.** The command picker offers Install, Gather,
   Wait for, Explain, Analyze, Destroy, and Completion. Install clears any
   previous directory selection and runs the stage flow below. Gather asks for
   an assets directory and infers bootstrap and control-plane hosts from it,
   Wait for asks for the event and then the assets directory, Explain opens the
   InstallConfig schema browser, and Destroy asks for an assets directory and
   tears down the cluster. Use `--version` on the command line to print version
   information.
2. **Asks for an assets directory** when the selected command needs one. You
   can type an existing directory or a new name; new directories are created
   for you. The prompt also lists subdirectories of the current path so you can
   pick one with the arrow keys. Pressing `enter` without typing opens the first
   listed directory (or the current directory when none are listed). Typing
   filters the list by prefix, and hidden directories are skipped.
3. **Detects the last installer stage** by inspecting (in order):
   - `.openshift_install_state.json` asset keys
   - on-disk artifacts (`install-config.yaml`, `manifests/`, `openshift/`,
     `bootstrap.ign`, `master.ign`, `worker.ign`, `auth/kubeconfig`,
     `metadata.json`, Terraform files)
   - `.openshift_install.log` command history (fallback only)
4. **After a stage completes**, prints a green `success: <stage>` line below
   the last log output, then shows the stages in order with completed stages
   marked `✓`:
   - Install Config
   - `m` → Manifests
   - `i` → Ignition Configs
   - `c` → Install Cluster
   - `l` → full installer log from `<assets-dir>/.openshift_install.log`
   - `esc` → stage menu
   - `q` → quit
5. **If the cluster stage was attempted**, scans `.openshift_install.log` for
   errors and offers the gather/analyze actions.
6. **If the directory has no installer state**, shows a menu of stages to run.
   Selecting a later stage automatically runs any missing stages first, starting
   from the directory's current stage. Each completed intermediate stage is
   shown briefly before the next one starts; pressing a key cancels the
   automatic transition.

### In-process stages

- **Install Config** asks for the values one at a time, in dependency order.
  The platform picker starts with no selection; pick Azure, AWS, or one of the
  other supported platforms.
  - **Azure**: credentials → region → base domain → cluster name → publish →
    SSH key → pull secret. Credentials are prefilled from
    `~/.azure/osServicePrincipal.json` (or `AZURE_AUTH_LOCATION`) and saved back
    to that file after a successful connection, so later in-process stages use
    them. When credentials are found, the credentials step is skipped and the
    Azure connection starts automatically; if it fails, the credentials step is
    shown again. Regions and base domains are fetched from the subscription. The
    region list is searchable (type to filter), and base domains are selected
    from the available Azure DNS zones.
  - **AWS**: region → base domain → cluster name → publish → SSH key → pull
    secret. Base domains are selected from the account's public Route 53 hosted
    zones; the lookup uses the ambient credential chain first and asks for
    access keys if needed. `ctrl+g` switches to manual base-domain entry and
    `ctrl+r` retries the lookup.
  - **Other platforms** use manual base-domain entry.

  The SSH key step discovers `*.pub` keys in `~/.ssh` and offers them as a
  selection, along with `<none>` and (when editing) a keep-existing option.

  Saving writes `install-config.yaml` and runs the install-config asset through
  the asset store for validation. No survey prompts are used.

  Navigating back and forward without changing a step preserves its value.
  When a step's value is changed and you move forward, values in all later
  steps are cleared so dependent data is not reused.
- **Manifests** and **Ignition Configs** run the installer's asset store
  directly, streaming logrus output into the log pane.
- Logs are appended to `<assets-dir>/.openshift_install.log`, which keeps stage
  detection working across runs. The success, failure, and cluster-check
  screens offer `l` to view the full installer log inside the TUI.
- Platform-specific fields beyond the collected region/base domain are preserved
  from an existing `install-config.yaml`. For other platforms, add the
  remaining platform fields to the file or use the CLI if the form's defaults
  are insufficient.

### Gather and wait-for

- **Gather** asks for an assets directory and infers the bootstrap and control
  plane hosts from `install-config.yaml`, as the CLI does. The `--bootstrap`,
  `--master`, and `--key` flags override or extend the inferred values, and
  `--skipAnalysis` skips bundle analysis.
- **Wait for** asks which event to wait for (`bootstrap-complete` or
  `install-complete`), then for the assets directory. It waits in-process using
  the installer's command package.

### Explain

- **Explain** opens a Miller-column browser for the InstallConfig schema. The
  leftmost column is `installconfig`; `→` expands the selected field into the
  next column, `←` goes back, `/` filters the current column's siblings, and
  `enter` opens a detail screen with the same output as
  `openshift-install explain <path>`. Press `s` to search all
  `installconfig.*` paths and jump directly to a match.

### Install Cluster

- The cluster stage runs in-process and shows a left-to-right component
  diagram. Components appear only after they are created: network, DNS,
  load balancer, bootstrap PC, temporary control plane, control-plane PCs,
  compute PCs, bootstrap completion, cluster operators, and install completion.
- DNS records appear one at a time; machine boxes show PC markers.
- Component state is inferred from installer log lines. Newly active
  components blink for a short time.
- `↑/↓` selects a component and shows its description.
- A separate log section sits below the diagram and scrolls with
  `pgup`/`pgdown` and `←`/`→`.
- Press `l` to expand the log section into a full scrollable log screen; press
  `l` again to return to the diagram.
- Log lines are not reflowed, so their original formatting and indentation are
  preserved; use `←`/`→` to scroll horizontally.
- Press `f` to cycle the log level filter:
  `all → debug → info → warn → error → all`. `info` shows info/warn/error,
  `warn` shows warn/error, and `error` shows errors/fatals. Debug logs are
  captured and shown when `all` or `debug` is selected. The active filter is
  shown in the log header.

### Destroy

- **Destroy** always clears the current directory selection and asks for the
  assets directory again, then runs the in-process cluster destroyer: it
  destroys platform resources, destroys cluster assets, removes the state file
  and Terraform files, and cleans up Cluster API artifacts.
- If `OPENSHIFT_INSTALL_REPORT_QUOTA_FOOTPRINT=true`, the quota report is
  written as the CLI does.
- After a successful destroy, the TUI asks whether to delete the assets
  directory. Press `y` to remove it and clear the directory selection; press
  `n` to keep it.

### Placeholders

- **Analyze** logs `done` and shows a placeholder screen. It does not collect
  anything yet.
- **Completion** shows the equivalent `openshift-install` invocation and
  returns to the command picker.

## Flags

| Flag | Behavior |
|---|---|
| `--dir string` | Assets directory. When explicitly set, the TUI opens it immediately instead of prompting, then shows the command picker unless a stage flag is also given. |
| `--log-level string` | Log level for asset generation and gather/wait-for (`debug`, `info`, `warn`, `error`). |
| `--version` | Print version information and exit. |
| `--config` | Open the install-config form immediately. |
| `--manifests` | Run the manifests stage, running missing prerequisites first. |
| `--ignition` | Run the ignition-configs stage, running missing prerequisites first. |
| `--ignition-configs` | Alias for `--ignition`. |
| `--cluster` | Run the cluster stage, running missing prerequisites first. |
| `--skip-password-print` | Do not print the generated user password in wait-for output. |
| `--bootstrap string` | Overrides the inferred bootstrap host for Gather. |
| `--master string` | Adds a control plane host for Gather (repeatable). |
| `--key string` | Adds an SSH private key for Gather (repeatable). |
| `--skipAnalysis` | Skip analysis of the gathered bundle. |
| `--file string` | Accepted for compatibility; analyze is a placeholder. |
| `-h, --help` | Show usage. |

`--config`, `--manifests`, `--ignition`, and `--cluster` are mutually
exclusive. When one is given, the TUI skips the resume/picker screens and runs
that stage, plus any missing prerequisites, as soon as the directory is known:

```sh
./bin/openshift-install-tui --dir ~/ocp-install --manifests
./bin/openshift-install-tui --dir ~/ocp-install --ignition
./bin/openshift-install-tui --dir ~/ocp-install --log-level debug
```

## Key bindings

| Screen | Keys |
|---|---|
| Command picker | `up/down`, `1-9`, `enter` run, `q` quit |
| Directory | `enter` open (first listed directory when empty), `up/down` pick a listed directory, `esc` command picker, `ctrl+c` quit |
| Wait for | `up/down`, `1-2`, `enter` continue, `esc` command picker, `q` quit |
| Explain | `up/down` select, `→` children, `←` back, `/` filter siblings, `s` search all paths, `enter` explain, `esc` command picker |
| Explain search | type to search, `up/down` select match, `enter` jump, `esc` cancel |
| Explain detail | `↑/↓`, `pgup/pgdown` scroll, `esc` back, `q` quit |
| Stage menu | `up/down`, `1-4`, `enter` run, `esc` command picker, `q` quit |
| Resume | `y` continue, `n` choose another stage, `q` quit |
| Cluster check | `g` gather, `a` analyze, `c` continue cluster, `m` menu, `q` quit |
| Cluster install | `↑/↓` select component, `pgup/pgdown` and `←/→` scroll logs, `f` log filter, `l` expand logs, `q/esc` cancel |
| Cluster logs | `↑/↓`, `pgup/pgdown`, `←/→` scroll, `f` log filter, `l` diagram, `q/esc` cancel |
| Running | `q/esc` cancel, arrows/pgup/pgdown scroll, `←/→` horizontal scroll, `f` log filter |
| Install Config wizard | `enter`/`tab` next, `esc` back, type to filter Azure regions, `←/→` choose region/base-domain/SSH key, `ctrl+r` retry base-domain lookup, `ctrl+g` manual base domain, `ctrl+s` save |
| Stage success | `m` manifests, `i` ignition, `c` cluster, `l` full log, `esc` menu, `q` quit |
| Logs | `↑/↓`, `pgup/pgdown`, `←/→` scroll, `f` log filter, `esc` back, `q` quit |
| Placeholder | `esc`/`m` back, `q` quit |

## Tests

```sh
go test ./cmd/openshift-install-tui
```

The tests cover stage detection, log error scanning, log bundle selection,
flag parsing, command-picker routing, wait-for staging, explain schema
navigation/search/detail, directory-list behavior, install-config wizard steps,
Azure region/base-domain selection, AWS base-domain selection, credential-entry
fallback, and install-config generation (including preserving existing platform
fields and SSH keys).

## Notes

- Selection lists use a date-picker-style wheel: the current choice is
  highlighted with a `#333333` background while neighboring entries fade out
  above and below. Async lookups use an inline spinner.
- The main panel pads the top of its content, leaving room for decorative UI
  above the inputs.
- Stage detection is best-effort. The asset store is authoritative when the
  installer resumes a partially generated directory.
- `q`/`esc` while a stage is running cancels the in-process asset generation or
  wait.
- Cluster component inference is best-effort and based on installer log text.
