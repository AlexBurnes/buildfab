# Project Requirements Document — **buildfab** CLI

## 1) Purpose & Scope

**buildfab** is a Go-based runner for project automations defined in a YAML file (same schema as used by your `pre-push` utility). It executes **stages** composed of **steps** (actions), supports **parallel** and **sequential** execution via dependencies, exposes a **library API** for embedding (e.g., `pre-push`), and provides a **CLI** for manual runs and scripting. The reference schema and working example are in the attached `project.yml`.&#x20;

### Primary goals

* Run **stages** (pipelines) and **actions** (units) on demand from CLI and from an embedded API.
* Preserve/extend semantics already validated by the `pre-push` utility: `run`, `uses`, `require`, `onerror`, `only`, variables, etc.&#x20;
* Provide clear, colorized output with concise repro tips on failure.

### Non-goals (v1)

* Matrix builds, reusable workflows, or remote executors.
* Secrets manager and complex environment templating (basic env pass-through only in v1).

---

## 2) Key Concepts & Terminology

* **Action**: a named unit defined under `actions:`. Either:

  * `run:` — shell command block, or
  * `uses:` — a built-in function (e.g., `git@untracked`, `git@uncommitted`, `git@modified`).&#x20;
* **Stage**: a pipeline of **steps** (each step references an **action**), defined under `stages:` (e.g., `pre-push`). Steps may declare `require:` dependencies and `onerror:` policies.&#x20;
* **Variables**: GitHub-style `${{ ... }}` placeholders resolvable from repo state (`tag`, `branch`), and version library values when embedded. (Your example shows `${{ tag }}`.)&#x20;

---

## 3) Configuration Schema (high-level)

The YAML file contains:

* `project`:

  * `name` (string), `modules` (list), optional `bin` directory. (Your example sets `name`, `modules`, `bin`.)&#x20;
* `actions` (list of objects):

  * `name` (string),
  * Either `run: |` (multiline shell) **or** `uses: <provider@builtin>`.
    Examples in your file: `version-check`, `version-greatest`, `version-module`, `run-tests`, `git-untracked`, `git-uncommitted`, `git-modified`.&#x20;
* `stages` (map of stage→definition):

  * Each stage has `steps` (list), each step has:

    * `action: <name>` (required),
    * `require:` string or list of action names,
    * `onerror:` `warn` | `stop` (default `stop`),
    * `only:` list of labels/conditions (e.g., `[release]`).
      (Your `pre-push` stage shows a realistic mix, including `require` and `only`.)&#x20;

**Validation expectations (v1):**

* All referenced actions exist.
* No cyclic dependencies among steps (DAG).
* `only:` is syntactically valid and left to the runner’s condition evaluator (see §6).

---

## 4) CLI Specification

### 4.1 Command forms

* `buildfab [stage] build` — run a stage named `build` (if present).
* `buildfab pre-push` — run stage `pre-push`.
* `buildfab pre-push version-check` — run **a single step** (`version-check`) inside stage `pre-push` (respecting its `require` chain only if `--with-requires` is provided; default is *just that step*).
* `buildfab [action] run-tests` — run a **standalone action** named `run-tests` directly.
* `buildfab check` — run stage `check` (e.g., linters/sanitizers).

**Name resolution rule:** If an identifier matches both a stage and an action, **stage takes priority** (as requested). An explicit `--action` can force action mode.

### 4.2 Global options

* `-f, --file <path>`: path to YAML (default: `project.yml`).
* `-v, --verbose`: show commands and captured outputs.
* `-d, --debug`: include internals (vars, cwd, timing).
* `--color auto|always|never`: color policy.
* `--env KEY=VAL` (repeatable): export env vars to actions.
* `--only <label>[,<label>...]`: inject labels for `only:` evaluation (e.g., `--only release`).
* `--with-requires`: when running a single step by name (stage context), also run its prerequisites (transitive).
* `--max-parallel N`: cap concurrency (default: logical CPUs).

**Exit codes**

* `0` on success or only warnings.
* `1` if any step with `onerror: stop` fails.
* `2` for configuration/validation errors (missing action, cycle, invalid YAML).

---

## 5) Execution Model (DAG Scheduler)

### 5.1 Resolution & planning

* Build a DAG per **stage**:

  * Node = step (references an action),
  * Edge = `require:` dependencies.
* Validate: missing actions, non-existent `require`, cycles → error (exit 2).
* Determine runnable “waves”: steps whose requires are satisfied run **in parallel**. (Wave-by-wave schedule.)

### 5.2 Running actions

* **run:** execute the multi-line shell with `/bin/sh -lc` (Linux/macOS) or `cmd /c` (Windows). Inherit environment plus `--env` values.
* **uses:** dispatch to registered built-ins (e.g., `git@untracked`, `git@uncommitted`, `git@modified`). Same semantics as in your working example.&#x20;

### 5.3 Conditions & policies

* **`only:`** a list of labels; a step runs only if all required labels are present in the current run context. Example: `only: [release]` (you use this for test gating). The CLI `--only` flag provides labels. If `only` is set and the label is absent → step is skipped (status = SKIP).&#x20;
* **`onerror:`** `warn` allows the DAG to continue; `stop` blocks dependents from starting (default). Your example shows `onerror: warn` for `git-modified`.&#x20;

### 5.4 Output & UX

* Per-step status line:

  * **OK**: `✔` green, **WARN**: `⚠` yellow, **ERROR**: `✖` red, **SKIP**: `○` dim.
* `--verbose`: print the command(s) and captured stdout/stderr (step-scoped, with truncation; configurable).
* On failure:

  * **uses**: print a concise **repro** hint (the built-in’s helper command sequence).
  * **run**: print “To reproduce:” and the exact block to paste.
* Deterministic final **summary** in topological order.

---

## 6) Variables & Context

### 6.1 Interpolation

* Replace `${{ ... }}` in `run:` blocks and in future action inputs:

  * Core: `tag`, `branch`.
  * Embedded version values (when used as a library): e.g., `${{ version.version }}`, `${{ version.project }}`, etc., same as your pre-push utility. (Your example demonstrates `${{ tag }}` already.)&#x20;

### 6.2 Resolution policy

* Fail fast (configuration error) if a placeholder cannot be resolved; suggest similar keys.
* Provide `--dump-context` (debug) to print available variables for troubleshooting.

---

## 7) Library API (for embedding)

### 7.1 Packages

* `pkg/config`: load & validate YAML → in-memory model (actions, stages, steps).
* `pkg/runtime`: execution API:

  * `Runner.RunStage(ctx, name, opts) (Report, error)`
  * `Runner.RunAction(ctx, name, opts) (Report, error)`
  * `Runner.RunStageStep(ctx, stage, step, opts)` (with/without requires)
* `pkg/builtins`: register and implement `uses:` actions (`git@untracked`, etc.).
* `pkg/vars`: variable providers (git state, version lib adapter).

### 7.2 Embedding pattern (pre-push)

* `pre-push` loads the same `project.yml`, constructs `Runner`, then calls:

  * `RunStage(ctx, "pre-push", opts)`
    matching the exact steps shown in your working YAML.&#x20;

---

## 8) Compatibility with the Attached `project.yml`

* Recognize `project.name`, `project.modules`, `project.bin`.&#x20;
* Support actions declared as **shell** (`run:`) and **built-ins** (`uses:`). Your file includes both.&#x20;
* Honor the `pre-push` **stage** with the listed **steps**, including:

  * `version-check`, `version-greatest`, `version-module`, `run-tests` (gated by `only: [release]` + `require: [version-module]`), and the Git checks with `onerror: warn` for `git-modified`.&#x20;

---

## 9) Configuration & Defaults

* Default YAML path: `project.yml`.
* Default stage when none provided in CLI: **none** (require explicit name) — but keep `pre-push` discoverable for `pre-push` embedding.
* Default concurrency: logical CPU count; override via `--max-parallel`.
* Default `onerror`: `stop`.
* If a CLI token matches both a stage and an action, **stage** is chosen unless `--action` is specified.

---

## 10) Build, Test, Release

* **Language:** Go 1.22+.
* **Tests:** unit tests (parsing, interpolation, planner, scheduler) + e2e tests against a temp git repo and your sample YAML.
* **Releases:** via GoReleaser (multi-platform artifacts). (You already do this for the `pre-push` utility; reuse/adapt pipeline settings.)
* **Distribution:** GitHub Releases; Homebrew/Scoop formulas (optional).

---

## 11) Security & Safety

* `run:` executes shell; document this clearly and recommend reviewing YAML in VCS.
* Provide `--allow-run` (default true) and `--deny-run` mode (only `uses:` allowed) for locked environments.
* Sanitize/escape variable expansions where injected into shell; support disabling interpolation per step in future.

---

## 12) Telemetry & Logs (optional v1)

* `--log-json` to emit machine-readable logs for CI aggregation.
* Exit codes as in §4.2 for CI signals.

---

## 13) Future Enhancements (post-v1)

* `env:` and `if:` per step; reusable action files; matrix strategies.
* Pluggable providers for `uses:` via Go plugins or sidecar binaries.
* Caching, artifacts, and conditional retries.

---

## 14) Acceptance Criteria

1. Can run `buildfab pre-push` and reproduce the behavior from the attached YAML, honoring `require`, `only`, and `onerror`.&#x20;
2. Can run a **single action**: `buildfab run-tests` (standalone).&#x20;
3. Can run a **single step** in a stage: `buildfab pre-push version-module`.&#x20;
4. Produces colorized, deterministic summaries; prints repro commands on failures.
5. Library API callable by the `pre-push` binary to run `pre-push` stage.
