# Experiments

We'll put scripts and examples here to demonstrate how various
isolation primitives work.

# cg-mem.py

This is to explore how memory accounting works across fork, when the
parent and child are in different CGs.  Memory can be allocated at
three points relative to a fork:

* before a fork (allocations A and B in the code).  These allocations
  get billed to the parent cgroup (even if all processes in that
  cgroup die!).  The memory is not reclaimed until both parent and
  children release it.
* after a fork, in parent (allocation C).  Billed to parent.  Nothing
  tricky here.
* after a fork, in child (allocation D).  Billed to child.  Nothing
  tricky here.

We also experimented manually after the parent died, leaving the child
orphaned.  At this point, the dead parent's cgroup still shows the
memory being used.  If you set a memory limit on the parent CG below
what it is using, you get "device busy".  If you remove that CG,
however, there's no issue (this is a memory leak, because the child's
CG still won't claim the memory).
