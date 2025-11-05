Caching can make your **multi-distro C++ builds** go from minutes to seconds. Here’s what to cache, why it helps, and concrete ways to wire it into your “matrix of container images” workflow.

# What to cache (and why)

1. **Compiler outputs (object files) — `ccache` / `sccache`**

   * Biggest win for C/C++ incremental builds.
   * Reuses previously compiled object files when **source, flags, compiler, and headers** match.
   * Typical speedups: 3–10× on repeat builds.

2. **CMake configure/generate artifacts** (`CMakeFiles/`, `CMakeCache.txt`)

   * Avoids repeating expensive configuration steps when inputs haven’t changed.
   * Works best when you keep a **stable build directory**.

3. **Package manager caches**

   * **Conan/vcpkg**: downloaded/compiled third-party libs.
   * **apt/yum/dnf** metadata + packages inside images built on the fly.
   * Huge time saver on matrix builds where each image would otherwise re-download toolchains and deps.

4. **Language-specific caches** (if you also build other parts)

   * Go module cache (`GOMODCACHE`), pip wheels, Cargo, npm, etc.

5. **Docker/BuildKit layer cache** (when you “derive” images per distro)

   * Reuses layers for system packages and toolchains across runs.

---

# Where to put the cache (so containers can reuse it)

Use **bind mounts** pointing to stable directories on the host (or a shared volume) and pass the right env vars:

```yaml
actions:
  - name: Build inside ${{ matrix.image }}
    container:
      engine: docker
      image: { from: ${{ matrix.image }} }
      workdir: /src
      mounts:
        # 1) ccache dir (per compiler & per distro is safest)
        - type: bind
          source: ~/.cache/buildfab/ccache/${{ matrix.image }}
          target: /ccache
        # 2) Conan cache
        - type: bind
          source: ~/.conan2
          target: /conan
        # 3) vcpkg cache
        - type: bind
          source: ~/.cache/buildfab/vcpkg
          target: /vcpkg-cache
        # 4) Build dir to persist CMake configure results (optional)
        - type: bind
          source: ./_build/${{ matrix.image }}
          target: /src/build
      env:
        CCACHE_DIR: /ccache
        CCACHE_MAXSIZE: 5G
        CCACHE_COMPRESS: "1"
        CONAN_HOME: /conan
        VCPKG_DEFAULT_BINARY_CACHE: /vcpkg-cache
      run:
        - bash -lc "ccache --zero-stats || true"
        - bash -lc "cmake -S . -B build -DCMAKE_BUILD_TYPE=Release -DCMAKE_C_COMPILER_LAUNCHER=ccache -DCMAKE_CXX_COMPILER_LAUNCHER=ccache"
        - bash -lc "cmake --build build -j"
        - bash -lc "ccache --show-stats || true"
```

**Why bind mounts?** Containers are ephemeral; mounts give them a stable place to **restore/use/save** caches across runs and across images.

---

# Good cache keys & isolation

* **Partition by ABI**: isolate caches by distro + compiler version (e.g., `~/.cache/buildfab/ccache/ubuntu-22.04/gcc-12`).
* **Compiler flags matter**: `-O3 -g -march=` changes cache hits. Don’t flip flags frequently.
* **Conan/vcpkg** are already content-addressed; a single shared cache per user is fine.

---

# If you build images on the fly (Dockerfile + BuildKit)

1. Enable **BuildKit** for fast, cached image builds:

```bash
export DOCKER_BUILDKIT=1
```

2. Use cache mounts in your Dockerfile for package managers / ccache:

```dockerfile
# Example for Debian/Ubuntu base
RUN --mount=type=cache,target=/var/cache/apt \
    --mount=type=cache,target=/var/lib/apt \
    apt-get update && apt-get install -y build-essential ccache cmake git

# ccache directory inside image (less ideal than a host bind, but helps layer reuse)
ENV CCACHE_DIR=/root/.ccache
RUN --mount=type=cache,target=/root/.ccache ccache -M 5G
```

3. In buildfab’s container spec, add:

```yaml
container:
  image:
    build:
      dockerfile: ci/Dockerfile.build
      context: .
      args: { BASE: ${{ matrix.image }} }
      # optionally pass --build-arg and enable BuildKit via env
```

**Layer cache** helps avoid re-installing packages each time; **ccache bind mounts** handle your code rebuilds.

---

# Conan & vcpkg quick notes

**Conan 2**

```bash
conan profile detect --force
conan install . --output-folder=build --build=missing
```

* Cache at `$CONAN_HOME` (we mounted `/conan`) avoids repeated downloads/builds.

**vcpkg**

```bash
./vcpkg install fmt:x64-linux
```

* Set `VCPKG_DEFAULT_BINARY_CACHE` to your mounted dir so built packages are reused across runs/images.

---

# When caching helps most

* **Matrix builds** across similar images (e.g., `ubuntu:22.04` vs `ubuntu:24.04`) → hits on large third-party libs via Conan/vcpkg.
* **Iterative local development** → `ccache` makes “tweak code / rebuild” cycles fast.
* **CI with fan-out** → warm caches reduce flakiness & runtime; pair with your `matrix.strategy.max_parallel` so caches aren’t thrashed.

---

# Pitfalls & hygiene

* **Stale cache/ABI mismatch** → weird link errors. Fix by clearing or scoping caches per image+compiler.
* **Permissions** → ensure the UID/GID inside container can write the mounted dirs (`user:` field or `chown` once).
* **Disk usage** → set caps (`CCACHE_MAXSIZE`, periodic cleanup).
* **Reproducibility vs speed** → caches speed builds but can hide “works-on-my-machine”. For release/repro builds, consider a **clean build** job or “cache-disabled” lane.

---

# Minimal side-by-side: no cache vs cached

**No cache**

```yaml
run:
  - bash -lc "cmake -S . -B build -DCMAKE_BUILD_TYPE=Release"
  - bash -lc "cmake --build build -j"
```

**With ccache + dep caches**

```yaml
mounts:
  - { type: bind, source: ~/.cache/buildfab/ccache/${{ matrix.image }}, target: /ccache }
  - { type: bind, source: ~/.conan2, target: /conan }
env:
  CCACHE_DIR: /ccache
  CONAN_HOME: /conan
run:
  - bash -lc "cmake -S . -B build -DCMAKE_BUILD_TYPE=Release -DCMAKE_C_COMPILER_LAUNCHER=ccache -DCACHE_CXX_COMPILER_LAUNCHER=ccache"
  - bash -lc "cmake --build build -j"
```

