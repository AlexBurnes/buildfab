# Enabling CPU and Memory Limits for Rootless Podman

This guide explains how to configure Podman to support CPU and memory limits in rootless containers by switching to `crun` runtime and delegating cgroup controllers.

## Overview

By default, rootless Podman containers cannot use CPU and memory limits due to cgroup restrictions. This guide shows how to:

1. Switch from `runc` to `crun` OCI runtime
2. Delegate CPU and memory controllers to user sessions
3. Verify the configuration works

## Prerequisites

- Linux system with cgroup v2
- Podman installed
- Root access for system configuration

## 1. Check Current Configuration

First, verify your current setup:

```bash
# Check cgroup version and manager
podman info --debug | grep -i cgroup

# Check current OCI runtime
podman info --format '{{.Host.OCIRuntime.Name}}'

# Check current delegation status
systemctl show user@$(id -u).service -p Delegate -p DelegateControllers
```

Expected output:
```
cgroupVersion: v2
cgroupManager: systemd
runc
Delegate=no
DelegateControllers=
```

## 2. Install and Configure crun

### Install crun

For RHEL-based systems (RHEL, CentOS, AlmaLinux, Rocky Linux):

```bash
sudo dnf install -y crun
```

For Debian/Ubuntu:

```bash
sudo apt update && sudo apt install -y crun
```

### Configure crun as Default Runtime

#### For All Users (System-Wide)

```bash
sudo mkdir -p /etc/containers
sudo tee /etc/containers/containers.conf >/dev/null <<'EOF'
[engine]
runtime = "crun"
EOF
```

#### For Single User (Rootless Only)

```bash
mkdir -p ~/.config/containers
cat > ~/.config/containers/containers.conf <<'EOF'
[engine]
runtime = "crun"
EOF
```

### Verify crun Configuration

```bash
podman info --format '{{.Host.OCIRuntime.Name}}'
# Expected output: crun
```

## 3. Delegate CPU and Memory Controllers

Rootless Podman needs control of the `cpu` and `memory` cgroup controllers under your user slice.

### Create Systemd Drop-In Configuration

```bash
sudo mkdir -p /etc/systemd/system/user@.service.d
sudo tee /etc/systemd/system/user@.service.d/delegate.conf >/dev/null <<'EOF'
[Service]
Delegate=yes
DelegateControllers=cpu cpuset io memory pids
EOF

# Reload systemd configuration
sudo systemctl daemon-reload
```

### Restart User Session

You need to restart your user session for the changes to take effect:

```bash
# Option 1: Log out and log back in
# Option 2: Terminate current user session
loginctl terminate-user "$USER"
```

After logging back in, verify delegation:

```bash
systemctl show user@$(id -u).service -p Delegate -p DelegateControllers
```

Expected output:
```
Delegate=yes
DelegateControllers=cpu cpuset io memory pids
```

## 4. Verify Cgroup Controllers

Check that your user slice has the required controllers:

```bash
cat /sys/fs/cgroup/user.slice/user-$(id -u).slice/cgroup.controllers
```

Expected output (order may vary):
```
cpu cpuset io memory pids
```

## 5. Test CPU and Memory Limits

Test that CPU and memory limits now work:

```bash
# Test CPU limit
podman run --rm --cpus 1.5 alpine:latest sh -c 'echo "CPU test passed"'

# Test memory limit
podman run --rm -m 512M alpine:latest sh -c 'echo "Memory test passed"'

# Test combined limits
podman run --rm --cpus 1.0 -m 256M alpine:latest sh -c 'echo "Combined limits test passed"'
```

If these commands run successfully without errors, your configuration is complete! ✅

## Troubleshooting

### Error: `error setting cgroup config ... cpu.max: no such file or directory`

**Cause:** Missing delegation or using `runc` under cgroup v2.

**Solution:**
1. Ensure you're using `crun`: `podman info --format '{{.Host.OCIRuntime.Name}}'`
2. Verify delegation: `systemctl show user@$(id -u).service -p Delegate`
3. Check controllers: `cat /sys/fs/cgroup/user.slice/user-$(id -u).slice/cgroup.controllers`

### Error: `OCI runtime attempted to invoke a command that was not found`

**Cause:** `crun` not installed or not in `$PATH`.

**Solution:**
```bash
# Reinstall crun
sudo dnf install -y crun  # RHEL-based
sudo apt install -y crun  # Debian-based

# Verify installation
which crun
```

### Error: `Delegate=no` after configuration

**Cause:** User session not restarted after delegation setup.

**Solution:**
```bash
# Restart user session
loginctl terminate-user "$USER"
# Then log back in and verify
systemctl show user@$(id -u).service -p Delegate
```

## Configuration Summary

| Task | Command | Expected Result |
|------|---------|----------------|
| Check runtime | `podman info --format '{{.Host.OCIRuntime.Name}}'` | `crun` |
| Check delegation | `systemctl show user@$(id -u).service -p Delegate` | `yes` |
| Check controllers | `cat /sys/fs/cgroup/user.slice/user-$(id -u).slice/cgroup.controllers` | Includes `cpu`, `memory` |
| Test limits | `podman run --rm --cpus 1 -m 512M alpine:latest` | Runs successfully |

## Alternative: Docker Configuration

If you prefer to use Docker instead of Podman, the same cgroup delegation is required:

```bash
# Same delegation setup as above
sudo mkdir -p /etc/systemd/system/user@.service.d
sudo tee /etc/systemd/system/user@.service.d/delegate.conf >/dev/null <<'EOF'
[Service]
Delegate=yes
DelegateControllers=cpu cpuset io memory pids
EOF

sudo systemctl daemon-reload
loginctl terminate-user "$USER"
```

Then test with Docker:

```bash
docker run --rm --cpus 1.0 -m 512M alpine:latest sh -c 'echo "Docker limits work"'
```

## References

- [Podman Rootless Cgroups v2 Documentation](https://github.com/containers/podman/blob/main/docs/tutorials/rootless_tutorial.md)
- [Systemd Resource Control](https://www.freedesktop.org/software/systemd/man/systemd.resource-control.html)
- [crun OCI Runtime](https://github.com/containers/crun)
- [Podman Rootless Guide](https://github.com/containers/podman/blob/main/docs/tutorials/rootless_tutorial.md)
