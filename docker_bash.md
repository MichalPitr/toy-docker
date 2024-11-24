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

sudo -- sh -c 'echo "50000 100000" > cpu.max'

sudo -- sh -c 'echo "500M" > memory.max'
```

## 2. Container-side setup

### Create isolated environment
```
# following two commands must run from the same root terminal
sudo -i

# Add current process to cgroup
echo $$ > /sys/fs/cgroup/toydocker.slice/container-1/cgroup.procs

# Create new namespaces
unshare \
    --uts \
    --pid \
    --mount \
    --net \
    --ipc \
    --cgroup \
    --fork \
    /bin/bash --norc
```

### Setup isolated environment
```
hostname container-1

# Make container's root private. This way mount events won't propagate in either direction.
mount --make-rprivate /tmp/container-1/merged

cd /tmp/container-1/merged

# Setup necessary mounts
mount -t proc proc proc/
mount -t sysfs sys sys/
mount -t tmpfs tmpfs tmp/
mount -t tmpfs tmpfs run/

# Setup minimal /dev
mount -t tmpfs tmpfs dev/
mkdir -p dev/pts dev/shm
mount -t devpts devpts dev/pts
mount -t tmpfs tmpfs dev/shm

# Setup cgroups
mkdir -p sys/fs/cgroup
mount -t cgroup2 none sys/fs/cgroup

# Finally chroot
chroot . /bin/sh
```

## 3. Let's verify that Cgroups work

```
# run cpu intensive command
yes > /dev/null
```
