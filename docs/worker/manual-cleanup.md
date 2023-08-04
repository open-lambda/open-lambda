# Manual Cleanup

When the worker exits cleanly, it frees lots of systems resources,
like mounts and cgroups.  If the worker panics are is killed by the
OOM (out of memory killer), these resources remain.

You may see messages saying "May require manual cleanup" when you try
restarting from this state.

Remedies:
1. `ol worker force-cleanup` will try to delete these old resources (this is analogous to [fsck](https://en.wikipedia.org/wiki/Fsck) for file system recovery)
2. `force-cleanup` doesn't always work -- in particular, we've seen cases where mounts get in a weird state and cannot be unmounted, even manually.  Rebooting the VM often solves this (resets mount points and cgroups)
3. when re-launching the worker, you can use a different directory with `-p WORKER_DIR` -- this will use different mount points and cgroups, hopefully avoiding issues (though if the Linux kernel is in a weird state, performance might be affected).
