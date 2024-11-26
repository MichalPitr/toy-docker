# container from scratch

## 1. Host-side setup

### Setting up filesystem

```
mkdir -p /tmp/container-1/{lower,upper,work,merged}

cd /tmp/container-1

wget https://dl-cdn.alpinelinux.org/alpine/v3.20/releases/x86_64/alpine-minirootfs-3.20.3-x86_64.tar.gz

tar -xzf alpine-minirootfs-3.20.3-x86_64.tar.gz -C lower

sudo mount -t overlay overlay -o lowerdir=lower,upperdir=upper,workdir=work merged

```
### Setting up control groups v2

```
# TODO: Decide if I want to keep parent slice like this or keep it simpler.
sudo mkdir -p /sys/fs/cgroup/toydocker.slice/container-1

cd /sys/fs/cgroup/toydocker.slice/

sudo -- sh -c 'echo "+memory +cpu" > cgroup.subtree_control'

cd container-1

# 10% cpu
sudo -- sh -c 'echo "10000 100000" > cpu.max'

# 512MiB
sudo -- sh -c 'echo "500M" > memory.max'

# Disable swap
sudo -- sh -c 'echo "0" > memory.swap.max'
```

## 2. Container-side setup

### Create isolated environment
```
# following two commands must run from the same  root terminal
sudo -i

# Add current process to cgroup
echo $$ > /sys/fs/cgroup/toydocker.slice/container-1/cgroup.procs

# Create new namespaces
unshare \
    --uts \
    --pid \
    --mount \
    --mount-proc \
    --net \
    --ipc \
    --cgroup \
    --fork \
    /bin/bash --norc
```

### Setup isolated environment
```
cd /tmp/container-1/merged

# Make container's root private
mount --make-rprivate /

# Create directory for old root
mkdir old_root

# Pivot root
pivot_root . old_root

# Unmount old root and remove the directory
umount -l /old_root
rm -rf /old_root


# Essential device nodes
mknod -m 666 dev/null c 1 3
mknod -m 666 dev/zero c 1 5
mknod -m 666 dev/tty c 5 0

# Essential system mounts
bin/mkdir -p dev/{pts,shm}
bin/mount -t devpts devpts dev/pts
bin/mount -t tmpfs tmpfs dev/shm
bin/mount -t sysfs sysfs sys/
bin/mount -t tmpfs tmpfs run/
bin/mount -t proc proc proc/

# Start shell with clean environment and signal handling
exec /bin/busybox sh
```

## 3. Let's verify that Cgroups work

```
# run cpu intensive command
while true; do true; done
```
