# Real-World Project Examples

This directory contains `.project.yml` configurations taken from real production projects.
They are provided **for reference and viewing only**.

> **These examples will not run in the buildfab repository itself.**
> They depend on specific project source trees, external toolchains (CMake, Conan, rpmbuild,
> Docker/Podman, Perl, etc.), credentials, and internal infrastructure that are not present here.
> They are not buildfab test fixtures — use `examples/*.yml` for that.

The purpose of these examples is to show how buildfab is used to orchestrate complex,
real-world build systems across different languages, packaging formats, and deployment targets.
They can serve as a starting point when setting up buildfab for a new project.

---

## Examples

### `С++ library/`

A C++ shared library project managed with CMake and the Conan package manager.

**Demonstrates:**
- Multi-compiler matrix builds (`gcc` / `clang` × `Release` / `Debug`)
- Conan dependency resolution and lockfile management
- CMake configure → build → test → install pipeline
- Creating and uploading a Conan package recipe to a private Artifactory repository
- In-container builds via Podman against multiple distros (`almalinux:9`, `centos:7`, `centos:8`, `alpine:3.22`)
- `conan-create` + test-build verify that the packaged library reports the correct version
- Code quality: `cppcheck`, `CodeChecker`, `clang-format` style enforcement
- Valgrind memory checks and AddressSanitizer builds
- Pre-push gate: version check, git hygiene, YAML lint, GitLab CI YAML validation

**Key stages:** `build`, `build-all`, `compiler-build`, `compiler-build-all`, `images-build`

---

### `C++ app/`

A C++ application project that consumes Conan packages and ships RPM packages and a Docker image.

**Demonstrates:**
- Same multi-compiler matrix as the library example
- RPM packaging: `rpmbuild` with version macros injected by buildfab interpolation
- Uploading RPMs to an internal repository (`rpm.example.com`)
- Building and slimming a Docker image (`docker-slim` produces a 30x+ smaller image)
- In-container CI simulation: running `compiler-build` stage inside Podman containers for each target distro
- OS-variant `install-requirements` actions (different Oracle client packages for CentOS 7 vs AlmaLinux)
- Uploading built Conan binaries to Artifactory after a successful build

**Key stages:** `build`, `compiler-build`, `rpm-build`, `images-build`

---

### `Perl with golang tool package/`

A mixed-language service project: Perl/shell scripts as the main deliverable, a compiled
binary controller (`ctrl`) built with CMake/Conan, all co-packaged into RPM and deployed
as a Docker image.

This is the most complex example and showcases buildfab's ability to orchestrate a
heterogeneous build pipeline end-to-end.

**Pipeline overview:**

```
[build ctrl binary]     [generate scripts]
  (CMake/Conan in          (interpolation.sh
   Podman container)         templates)
         |                       |
         +----------+------------+
                    |
              [rpm-build]
         (rpmbuild in Docker
          for centos:7/8, alma:9)
                    |
              [docker-build]
         (assemble Docker image
          from installed RPMs)
                    |
              [docker-push]
```

**Demonstrates:**
- Cross-container artifact chaining: ctrl binary built in one container, consumed by RPM build in another
- `artifacts:` extraction — built binary is copied out of the Podman container back to the host
- Parallel multi-distro RPM builds with `max_parallel: 1` (sequential to avoid registry conflicts)
- Full `images-build-all` stage: version gate → ctrl build → RPM upload → Docker build/push
- Separate quick stages for development iteration: `build-ctrl`, `build-rpm`, `docker-build`
- Shell script generation from templates via `interpolation.sh`

**Key stages:** `build-ctrl`, `build-rpm`, `upload-rpm`, `images-build`, `images-build-all`, `docker-build`

---

### `Project Metapackage/`

An Ansible/Perl metapackage project: a Perl orchestration script assembles a distribution
archive that bundles a Go submodule binary together with Perl modules, then packages
everything into RPMs for multiple distros.

**Demonstrates:**
- Calling a Perl script (`scripts/playbook.pl`) as the primary build step
- Embedding a versioned Go submodule (`module/`) whose version is extracted separately
- RPM packaging that injects both the outer project version and the inner module version as macros
- `onerror: warn` on git-status checks so non-fatal drift does not block the pre-push gate
- Multi-distro container RPM builds and uploads in a single `images-build-all` stage

**Key stages:** `build-rpm`, `upload-rpm`, `images-build`, `images-build-all`

---

### `project/`

A collection of **reusable include files** shared across the C++ library and C++ app examples
above. These are split by concern and pulled in via `include:` in each project's `.project.yml`.

| File | Contents |
|------|----------|
| `buildfab.yml` | Version and binary install actions |
| `git.yml` | `git-untracked`, `git-uncommitted`, `git-modified` stage definitions |
| `gitlab.yml` | GitLab CI YAML lint and API validation actions |
| `conan.yml` | Conan check, profile detection, and cache actions |
| `cmake.yml` | CMake prerequisite checks |
| `clang.yml` | clang-format style check/fix actions |
| `valgrind.yml` | Valgrind memory and sanitizer test actions |
| `pvs-studio.yml` | PVS-Studio static analysis actions |
| `cpp-check.yml` | cppcheck static analysis actions |
| `update-checking-stages.yml` / `update-checking-actions.yml` | Shared update-checking pipeline used by the Perl and Metapackage examples |
| `colors.sh` | ANSI color helpers sourced by shell actions |

This layout shows the **include-based composition** pattern: a shared library of actions
and stages, pulled into each project with one or two `include:` lines.

---

## Common patterns shown across all examples

- **`${{ os }}` / `${{ os_version }}` interpolation** to select platform-specific env files, conan profiles, and spec files
- **`variants:` with `when:`** for OS-conditional action bodies instead of if-chains in scripts
- **`matrix:` + `max_parallel:`** to iterate over distro images or compiler combinations
- **`container.artifacts`** to extract build outputs from containers back to the host
- **`env_file:`** to pass credentials and environment to container runs without hardcoding them
- **`onerror: warn`** for informational checks that should not block the pipeline
- **Pre-push gates** (`version-check`, `version-greatest`, `git-untracked`, YAML lint) as a
  lightweight local CI step that runs before every `git push`
