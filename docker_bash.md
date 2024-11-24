# container from scratch

## Host-side setup

### Setting up filesystem

`mkdir /tmp/container-1/{lower,upper,work,merged}`

`cd /tmp/container-1`

`wget https://dl-cdn.alpinelinux.org/alpine/v3.20/releases/x86_64/alpine-minirootfs-3.20.3-x86_64.tar.gz`

`tar -xzf alpine-minirootfs-3.20.3-x86_64.tar.gz -C lower`

(Remember to unmount once done!)
`sudo mount -t overlay overlay -o lowerdir=lower,upperdir=upper,workdir=work merged`

### Setting up control groups v2

Creating parent cgroup for our container.

`sudo mkdir -p /sys/fs/cgroup/toydocker.slice/container-1`

This let's child cgroup modify memory and cpu limits.

`sudo -- sh -c 'echo "+memory +cpu" > cgroup.subtree_control'`

To convert requests and limits, we need to set:

1. requests.cpu -> cpu.weight
2. limits.cpu -> cpu.max
3. requests.memory -> nothing
4. limits.memory -> memory.max

Let's skip requests and just set limits to 500m CPU and 500MiB

`vim cpu.max`

`vim memory.max`

With this, we are done with host-side setup.

## Container-side setup

Next, we setup the namespaces for the container.

`echo $$ > /sys/fs/cgroup/toydocker.slice/container-1/cgroup.procs`

```
unshare --user --map-user=1000 -r \
        --uts \
        --pid \
        --cgroup \
        --kill-child \
        /bin/bash --norc
```

let's check hostname, should be same as original.
`hostname`

Let's change it
`hostname container-1 && hostname`

`cd /tmp/container-1/merged`

`chroot . bin/sh`

`unshare --mount /bin/sh`

`mount -t proc proc /proc`


Now we have an 
