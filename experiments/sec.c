#include <errno.h>
#include <stddef.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <linux/audit.h>
#include <linux/filter.h>
#include <seccomp.h>
#include <sys/prctl.h>
#include <sys/syscall.h>
#include <time.h>
#include <assert.h>
#include <sys/types.h>
#include <sys/wait.h>

// sudo apt-get install libseccomp-dev

// gcc sec2.c -lseccomp -lrt; ./a.out 

// seccom init:
// allow many
// allow few

// getpid latency:
// (1) no seccomp,
// (2) allow many, succeed (3) allow few, succeed
// (4) allow many, fail (5) allow few, fail

// sudo apt-get install libseccomp-dev

long long diff_nsec(struct timespec t1, struct timespec t0) {
  long long diff = (t1.tv_sec - t0.tv_sec);
  diff *= 1000000000;
  assert(diff >= 0);
  diff += t1.tv_nsec - t0.tv_nsec;
  assert(diff >= 0);
  return diff;
}

void test(int filter_min, int filter_max) {
    struct timespec t0, t1, t2;

    int pid = fork();
    assert(pid >= 0);
    if (pid) {
      waitpid(pid, NULL, 0);
      return;
    }

    clock_gettime(CLOCK_REALTIME, &t0);

    if (filter_min >= 0) {
      scmp_filter_ctx ctx = seccomp_init(SCMP_ACT_ERRNO(123));

      // https://github.com/moby/moby/blob/master/profiles/seccomp/default.json
      int calls[] = {
                     SCMP_SYS(getpid),
                     SCMP_SYS(write),
                     SCMP_SYS(clock_gettime),
                     SCMP_SYS(clock_gettime64),
                     SCMP_SYS(exit),
                     SCMP_SYS(accept),
                     SCMP_SYS(accept4),
                     SCMP_SYS(access),
                     SCMP_SYS(adjtimex),
                     SCMP_SYS(alarm),
                     SCMP_SYS(bind),
                     SCMP_SYS(brk),
                     SCMP_SYS(capget),
                     SCMP_SYS(capset),
                     SCMP_SYS(chdir),
                     SCMP_SYS(chmod),
                     SCMP_SYS(chown),
                     SCMP_SYS(chown32),
                     SCMP_SYS(clock_adjtime),
                     SCMP_SYS(clock_adjtime64),
                     SCMP_SYS(clock_getres),
                     SCMP_SYS(clock_getres_time64),
                     SCMP_SYS(clock_nanosleep),
                     SCMP_SYS(clock_nanosleep_time64),
                     SCMP_SYS(close),
                     SCMP_SYS(connect),
                     SCMP_SYS(copy_file_range),
                     SCMP_SYS(creat),
                     SCMP_SYS(dup),
                     SCMP_SYS(dup2),
                     SCMP_SYS(dup3),
                     SCMP_SYS(epoll_create),
                     SCMP_SYS(epoll_create1),
                     SCMP_SYS(epoll_ctl),
                     SCMP_SYS(epoll_ctl_old),
                     SCMP_SYS(epoll_pwait),
                     SCMP_SYS(epoll_wait),
                     SCMP_SYS(epoll_wait_old),
                     SCMP_SYS(eventfd),
                     SCMP_SYS(eventfd2),
                     SCMP_SYS(execve),
                     SCMP_SYS(execveat),
                     SCMP_SYS(exit_group),
                     SCMP_SYS(faccessat),
                     SCMP_SYS(fadvise64),
                     SCMP_SYS(fadvise64_64),
                     SCMP_SYS(fallocate),
                     SCMP_SYS(fanotify_mark),
                     SCMP_SYS(fchdir),
                     SCMP_SYS(fchmod),
                     SCMP_SYS(fchmodat),
                     SCMP_SYS(fchown),
                     SCMP_SYS(fchown32),
                     SCMP_SYS(fchownat),
                     SCMP_SYS(fcntl),
                     SCMP_SYS(fcntl64),
                     SCMP_SYS(fdatasync),
                     SCMP_SYS(fgetxattr),
                     SCMP_SYS(flistxattr),
                     SCMP_SYS(flock),
                     SCMP_SYS(fork),
                     SCMP_SYS(fremovexattr),
                     SCMP_SYS(fsetxattr),
                     SCMP_SYS(fstat),
                     SCMP_SYS(fstat64),
                     SCMP_SYS(fstatat64),
                     SCMP_SYS(fstatfs),
                     SCMP_SYS(fstatfs64),
                     SCMP_SYS(fsync),
                     SCMP_SYS(ftruncate),
                     SCMP_SYS(ftruncate64),
                     SCMP_SYS(futex),
                     SCMP_SYS(futex_time64),
                     SCMP_SYS(futimesat),
                     SCMP_SYS(getcpu),
                     SCMP_SYS(getcwd),
                     SCMP_SYS(getdents),
                     SCMP_SYS(getdents64),
                     SCMP_SYS(getegid),
                     SCMP_SYS(getegid32),
                     SCMP_SYS(geteuid),
                     SCMP_SYS(geteuid32),
                     SCMP_SYS(getgid),
                     SCMP_SYS(getgid32),
                     SCMP_SYS(getgroups),
                     SCMP_SYS(getgroups32),
                     SCMP_SYS(getitimer),
                     SCMP_SYS(getpeername),
                     SCMP_SYS(getpgid),
                     SCMP_SYS(getpgrp),
                     SCMP_SYS(getppid),
                     SCMP_SYS(getpriority),
                     SCMP_SYS(getrandom),
                     SCMP_SYS(getresgid),
                     SCMP_SYS(getresgid32),
                     SCMP_SYS(getresuid),
                     SCMP_SYS(getresuid32),
                     SCMP_SYS(getrlimit),
                     SCMP_SYS(get_robust_list),
                     SCMP_SYS(getrusage),
                     SCMP_SYS(getsid),
                     SCMP_SYS(getsockname),
                     SCMP_SYS(getsockopt),
                     SCMP_SYS(get_thread_area),
                     SCMP_SYS(gettid),
                     SCMP_SYS(gettimeofday),
                     SCMP_SYS(getuid),
                     SCMP_SYS(getuid32),
                     SCMP_SYS(getxattr),
                     SCMP_SYS(inotify_add_watch),
                     SCMP_SYS(inotify_init),
                     SCMP_SYS(inotify_init1),
                     SCMP_SYS(inotify_rm_watch),
                     SCMP_SYS(io_cancel),
                     SCMP_SYS(ioctl),
                     SCMP_SYS(io_destroy),
                     SCMP_SYS(io_getevents),
                     SCMP_SYS(io_pgetevents),
                     SCMP_SYS(io_pgetevents_time64),
                     SCMP_SYS(ioprio_get),
                     SCMP_SYS(ioprio_set),
                     SCMP_SYS(io_setup),
                     SCMP_SYS(io_submit),
                     SCMP_SYS(io_uring_enter),
                     SCMP_SYS(io_uring_register),
                     SCMP_SYS(io_uring_setup),
                     SCMP_SYS(ipc),
                     SCMP_SYS(kill),
                     SCMP_SYS(lchown),
                     SCMP_SYS(lchown32),
                     SCMP_SYS(lgetxattr),
                     SCMP_SYS(link),
                     SCMP_SYS(linkat),
                     SCMP_SYS(listen),
                     SCMP_SYS(listxattr),
                     SCMP_SYS(llistxattr),
                     SCMP_SYS(_llseek),
                     SCMP_SYS(lremovexattr),
                     SCMP_SYS(lseek),
                     SCMP_SYS(lsetxattr),
                     SCMP_SYS(lstat),
                     SCMP_SYS(lstat64),
                     SCMP_SYS(madvise),
                     SCMP_SYS(membarrier),
                     SCMP_SYS(memfd_create),
                     SCMP_SYS(mincore),
                     SCMP_SYS(mkdir),
                     SCMP_SYS(mkdirat),
                     SCMP_SYS(mknod),
                     SCMP_SYS(mknodat),
                     SCMP_SYS(mlock),
                     SCMP_SYS(mlock2),
                     SCMP_SYS(mlockall),
                     SCMP_SYS(mmap),
                     SCMP_SYS(mmap2),
                     SCMP_SYS(mprotect),
                     SCMP_SYS(mq_getsetattr),
                     SCMP_SYS(mq_notify),
                     SCMP_SYS(mq_open),
                     SCMP_SYS(mq_timedreceive),
                     SCMP_SYS(mq_timedreceive_time64),
                     SCMP_SYS(mq_timedsend),
                     SCMP_SYS(mq_timedsend_time64),
                     SCMP_SYS(mq_unlink),
                     SCMP_SYS(mremap),
                     SCMP_SYS(msgctl),
                     SCMP_SYS(msgget),
                     SCMP_SYS(msgrcv),
                     SCMP_SYS(msgsnd),
                     SCMP_SYS(msync),
                     SCMP_SYS(munlock),
                     SCMP_SYS(munlockall),
                     SCMP_SYS(munmap),
                     SCMP_SYS(nanosleep),
                     SCMP_SYS(newfstatat),
                     SCMP_SYS(_newselect),
                     SCMP_SYS(open),
                     SCMP_SYS(openat),
                     SCMP_SYS(pause),
                     SCMP_SYS(pidfd_open),
                     SCMP_SYS(pidfd_send_signal),
                     SCMP_SYS(pipe),
                     SCMP_SYS(pipe2),
                     SCMP_SYS(poll),
                     SCMP_SYS(ppoll),
                     SCMP_SYS(ppoll_time64),
                     SCMP_SYS(prctl),
                     SCMP_SYS(pread64),
                     SCMP_SYS(preadv),
                     SCMP_SYS(preadv2),
                     SCMP_SYS(prlimit64),
                     SCMP_SYS(pselect6),
                     SCMP_SYS(pselect6_time64),
                     SCMP_SYS(pwrite64),
                     SCMP_SYS(pwritev),
                     SCMP_SYS(pwritev2),
                     SCMP_SYS(read),
                     SCMP_SYS(readahead),
                     SCMP_SYS(readlink),
                     SCMP_SYS(readlinkat),
                     SCMP_SYS(readv),
                     SCMP_SYS(recv),
                     SCMP_SYS(recvfrom),
                     SCMP_SYS(recvmmsg),
                     SCMP_SYS(recvmmsg_time64),
                     SCMP_SYS(recvmsg),
                     SCMP_SYS(remap_file_pages),
                     SCMP_SYS(removexattr),
                     SCMP_SYS(rename),
                     SCMP_SYS(renameat),
                     SCMP_SYS(renameat2),
                     SCMP_SYS(restart_syscall),
                     SCMP_SYS(rmdir),
                     SCMP_SYS(rseq),
                     SCMP_SYS(rt_sigaction),
                     SCMP_SYS(rt_sigpending),
                     SCMP_SYS(rt_sigprocmask),
                     SCMP_SYS(rt_sigqueueinfo),
                     SCMP_SYS(rt_sigreturn),
                     SCMP_SYS(rt_sigsuspend),
                     SCMP_SYS(rt_sigtimedwait),
                     SCMP_SYS(rt_sigtimedwait_time64),
                     SCMP_SYS(rt_tgsigqueueinfo),
                     SCMP_SYS(sched_getaffinity),
                     SCMP_SYS(sched_getattr),
                     SCMP_SYS(sched_getparam),
                     SCMP_SYS(sched_get_priority_max),
                     SCMP_SYS(sched_get_priority_min),
                     SCMP_SYS(sched_getscheduler),
                     SCMP_SYS(sched_rr_get_interval),
                     SCMP_SYS(sched_rr_get_interval_time64),
                     SCMP_SYS(sched_setaffinity),
                     SCMP_SYS(sched_setattr),
                     SCMP_SYS(sched_setparam),
                     SCMP_SYS(sched_setscheduler),
                     SCMP_SYS(sched_yield),
                     SCMP_SYS(seccomp),
                     SCMP_SYS(select),
                     SCMP_SYS(semctl),
                     SCMP_SYS(semget),
                     SCMP_SYS(semop),
                     SCMP_SYS(semtimedop),
                     SCMP_SYS(semtimedop_time64),
                     SCMP_SYS(send),
                     SCMP_SYS(sendfile),
                     SCMP_SYS(sendfile64),
                     SCMP_SYS(sendmmsg),
                     SCMP_SYS(sendmsg),
                     SCMP_SYS(sendto),
                     SCMP_SYS(setfsgid),
                     SCMP_SYS(setfsgid32),
                     SCMP_SYS(setfsuid),
                     SCMP_SYS(setfsuid32),
                     SCMP_SYS(setgid),
                     SCMP_SYS(setgid32),
                     SCMP_SYS(setgroups),
                     SCMP_SYS(setgroups32),
                     SCMP_SYS(setitimer),
                     SCMP_SYS(setpgid),
                     SCMP_SYS(setpriority),
                     SCMP_SYS(setregid),
                     SCMP_SYS(setregid32),
                     SCMP_SYS(setresgid),
                     SCMP_SYS(setresgid32),
                     SCMP_SYS(setresuid),
                     SCMP_SYS(setresuid32),
                     SCMP_SYS(setreuid),
                     SCMP_SYS(setreuid32),
                     SCMP_SYS(setrlimit),
                     SCMP_SYS(set_robust_list),
                     SCMP_SYS(setsid),
                     SCMP_SYS(setsockopt),
                     SCMP_SYS(set_thread_area),
                     SCMP_SYS(set_tid_address),
                     SCMP_SYS(setuid),
                     SCMP_SYS(setuid32),
                     SCMP_SYS(setxattr),
                     SCMP_SYS(shmat),
                     SCMP_SYS(shmctl),
                     SCMP_SYS(shmdt),
                     SCMP_SYS(shmget),
                     SCMP_SYS(shutdown),
                     SCMP_SYS(sigaltstack),
                     SCMP_SYS(signalfd),
                     SCMP_SYS(signalfd4),
                     SCMP_SYS(sigprocmask),
                     SCMP_SYS(sigreturn),
                     SCMP_SYS(socket),
                     SCMP_SYS(socketcall),
                     SCMP_SYS(socketpair),
                     SCMP_SYS(splice),
                     SCMP_SYS(stat),
                     SCMP_SYS(stat64),
                     SCMP_SYS(statfs),
                     SCMP_SYS(statfs64),
                     SCMP_SYS(statx),
                     SCMP_SYS(symlink),
                     SCMP_SYS(symlinkat),
                     SCMP_SYS(sync),
                     SCMP_SYS(sync_file_range),
                     SCMP_SYS(syncfs),
                     SCMP_SYS(sysinfo),
                     SCMP_SYS(tee),
                     SCMP_SYS(tgkill),
                     SCMP_SYS(time),
                     SCMP_SYS(timer_create),
                     SCMP_SYS(timer_delete),
                     SCMP_SYS(timer_getoverrun),
                     SCMP_SYS(timer_gettime),
                     SCMP_SYS(timer_gettime64),
                     SCMP_SYS(timer_settime),
                     SCMP_SYS(timer_settime64),
                     SCMP_SYS(timerfd_create),
                     SCMP_SYS(timerfd_gettime),
                     SCMP_SYS(timerfd_gettime64),
                     SCMP_SYS(timerfd_settime),
                     SCMP_SYS(timerfd_settime64),
                     SCMP_SYS(times),
                     SCMP_SYS(tkill),
                     SCMP_SYS(truncate),
                     SCMP_SYS(truncate64),
                     SCMP_SYS(ugetrlimit),
                     SCMP_SYS(umask),
                     SCMP_SYS(uname),
                     SCMP_SYS(unlink),
                     SCMP_SYS(unlinkat),
                     SCMP_SYS(utime),
                     SCMP_SYS(utimensat),
                     SCMP_SYS(utimensat_time64),
                     SCMP_SYS(utimes),
                     SCMP_SYS(vfork),
                     SCMP_SYS(vmsplice),
                     SCMP_SYS(wait4),
                     SCMP_SYS(waitid),
                     SCMP_SYS(waitpid),
                     SCMP_SYS(writev)
      };

      //SCMP_SYS(close_range),
      //SCMP_SYS(epoll_pwait2),
      //SCMP_SYS(faccessat2),
      //SCMP_SYS(openat2),

      assert(filter_max <= sizeof(calls)/sizeof(calls[0]));
      for (int i=filter_min; i<filter_max; i++) {
        seccomp_rule_add(ctx, SCMP_ACT_ALLOW, calls[i], 0);
      }

      seccomp_load(ctx);
    }

    clock_gettime(CLOCK_REALTIME, &t1);

    int iters = 10000000; // 10 mil
    for (int i=0; i<iters; i++)
      getpid();

    clock_gettime(CLOCK_REALTIME, &t2);

    printf("%d,%d,%lld,%lld\n", filter_min, filter_max,
           diff_nsec(t1, t0), diff_nsec(t2, t1)/iters);
    exit(0);
}

int main() {
  printf("min,max,setup,syscall\n");
  test(-1,-1);
  test(0,3);
  test(1,4);
  test(0,335);
  test(1,336);
  return 0;
}
