# Project Specification: buildfab

## 1) Purpose & Scope

**buildfab** is a Go-based runner for project automations defined in a YAML file. It executes **stages** composed of **steps** (actions), supports **parallel** and **sequential** execution via dependencies, exposes a **library API** for embedding (e.g., `pre-push`), and provides a **CLI** for manual runs and scripting.

### Primary Goals

* Run **stages** (pipelines) and **actions** (units) on demand from CLI and from an embedded API
* Preserve/extend semantics already validated by the `pre-push` utility: `run`, `uses`, `require`, `onerror`, `only`, variables, etc.
* Provide clear, colorized output with concise repro tips on failure

### Non-goals (v1)

* Matrix builds, reusable workflows, or remote executors
* Secrets manager and complex environment templating (basic env pass-through only in v1)

## 2) Key Concepts & Terminology

* **Action**: a named unit defined under `actions:`. Either:
  * `run:` — shell command block, or
  * `uses:` — a built-in function (e.g., `git@untracked`, `git@uncommitted`, `git@modified`)
* **Stage**: a pipeline of **steps** (each step references an **action**), defined under `stages:` (e.g., `pre-push`). Steps may declare `require:` dependencies and `onerror:` policies
* **Variables**: GitHub-style `${{ ... }}` placeholders resolvable from repo state (`tag`, `branch`), and version library values when embedded

## 3) Configuration Schema

The YAML file contains:

* `project`:
  * `name` (string), `modules` (list), optional `bin` directory
* `actions` (list of objects):
  * `name` (string),
  * Either `run: |` (multiline shell) **or** `uses: <provider@builtin>`
* `stages` (map of stage→definition):
  * Each stage has `steps` (list), each step has:
    * `action: <name>` (required),
    * `require:` string or list of action names,
    * `onerror:` `warn` | `stop` (default `stop`),
    * `only:` list of labels/conditions (e.g., `[release]`)

**Validation expectations (v1):**
* All referenced actions exist
* No cyclic dependencies among steps (DAG)
* `only:` is syntactically valid and left to the runner's condition evaluator

## 4) CLI Specification

### 4.1 Command Forms

* `buildfab run <stage>` — run a stage named `<stage>` (if present)
* `buildfab run pre-push` — run stage `pre-push`
* `buildfab run pre-push version-check` — run **a single step** (`version-check`) inside stage `pre-push` (respecting its `require` chain only if `--with-requires` is provided; default is *just that step*)
* `buildfab action <action>` — run a **standalone action** named `<action>` directly
* `buildfab list-actions` — list available built-in actions
* `buildfab validate` — validate project.yml configuration

**Name resolution rule:** If an identifier matches both a stage and an action, **stage takes priority**. An explicit `--action` can force action mode.

### 4.2 Global Options

* `-c, --config <path>`: path to YAML (default: `project.yml`)
* `-v, --verbose`: show commands and captured outputs
* `-d, --debug`: include internals (vars, cwd, timing)
* `--color auto|always|never`: color policy
* `--env KEY=VAL` (repeatable): export env vars to actions
* `--only <label>[,<label>...]`: inject labels for `only:` evaluation (e.g., `--only release`)
* `--with-requires`: when running a single step by name (stage context), also run its prerequisites (transitive)
* `--max-parallel N`: cap concurrency (default: logical CPUs)

**Exit codes**
* `0` on success or only warnings
* `1` if any step with `onerror: stop` fails
* `2` for configuration/validation errors (missing action, cycle, invalid YAML)

## 5) Execution Model (DAG Scheduler)

### 5.1 Resolution & Planning

* Build a DAG per **stage**:
  * Node = step (references an action),
  * Edge = `require:` dependencies
* Validate: missing actions, non-existent `require`, cycles → error (exit 2)
* Determine runnable "waves": steps whose requires are satisfied run **in parallel**. (Wave-by-wave schedule)

### 5.2 Running Actions

* **run:** execute the multi-line shell with `/bin/sh -lc` (Linux/macOS) or `cmd /c` (Windows). Inherit environment plus `--env` values
* **uses:** dispatch to registered built-ins (e.g., `git@untracked`, `git@uncommitted`, `git@modified`)

### 5.3 Conditions & Policies

* **`only:`** a list of labels; a step runs only if all required labels are present in the current run context. Example: `only: [release]`. The CLI `--only` flag provides labels. If `only` is set and the label is absent → step is skipped (status = SKIP)
* **`onerror:`** `warn` allows the DAG to continue; `stop` blocks dependents from starting (default)

### 5.4 Output & UX

* Per-step status line:
  * **OK**: `✔` green, **WARN**: `⚠` yellow, **ERROR**: `✖` red, **SKIP**: `○` dim
* `--verbose`: print the command(s) and captured stdout/stderr (step-scoped, with truncation; configurable)
* On failure:
  * **uses**: print a concise **repro** hint (the built-in's helper command sequence)
  * **run**: print "To reproduce:" and the exact block to paste
* Deterministic final **summary** in topological order

## 6) Variables & Context

### 6.1 Interpolation

* Replace `${{ ... }}` in `run:` blocks and in future action inputs:
  * Core: `tag`, `branch`
  * Embedded version values (when used as a library): e.g., `${{ version.version }}`, `${{ version.project }}`, etc.

### 6.2 Resolution Policy

* Fail fast (configuration error) if a placeholder cannot be resolved; suggest similar keys
* Provide `--dump-context` (debug) to print available variables for troubleshooting

## 7) Library API (for embedding)

### 7.1 Packages

* `pkg/buildfab`: main API functions
  * `RunStage(ctx, name, opts) (Report, error)`
  * `RunAction(ctx, name, opts) (Report, error)`
  * `RunStageStep(ctx, stage, step, opts)` (with/without requires)
* `internal/actions`: register and implement `uses:` actions (`git@untracked`, etc.)
* `internal/variables`: variable providers (git state, version lib adapter)

### 7.2 Embedding Pattern (pre-push)

* `pre-push` loads the same `project.yml`, constructs `Runner`, then calls:
  * `RunStage(ctx, "pre-push", opts)`
    matching the exact steps shown in the working YAML

## 8) Built-in Actions

### Git Actions
- `git@untracked` - Fail if untracked files present
- `git@uncommitted` - Fail if staged/unstaged changes present  
- `git@modified` - Fail if working tree differs from HEAD

### Version Actions
- `version@check` - Validate version format and consistency
- `version@check-greatest` - Ensure current version is greatest

## 9) Security & Safety

* `run:` executes shell; document this clearly and recommend reviewing YAML in VCS
* Provide `--allow-run` (default true) and `--deny-run` mode (only `uses:` allowed) for locked environments
* Sanitize/escape variable expansions where injected into shell; support disabling interpolation per step in future

## 10) Acceptance Criteria

1. Can run `buildfab run pre-push` and reproduce the behavior from the attached YAML, honoring `require`, `only`, and `onerror`
2. Can run a **single action**: `buildfab action run-tests` (standalone)
3. Can run a **single step** in a stage: `buildfab run pre-push version-module`
4. Produces colorized, deterministic summaries; prints repro commands on failures
5. Library API callable by the `pre-push` binary to run `pre-push` stage