require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_SYSCALL_H)) {
    die("Never use <bits/syscall.h> directly; include <sys/syscall.h> instead.");
}
eval 'sub __GLIBC_LINUX_VERSION_CODE () {331776;}' unless defined(&__GLIBC_LINUX_VERSION_CODE);
if(defined(&__NR_FAST_atomic_update)) {
    eval 'sub SYS_FAST_atomic_update () { &__NR_FAST_atomic_update;}' unless defined(&SYS_FAST_atomic_update);
}
if(defined(&__NR_FAST_cmpxchg)) {
    eval 'sub SYS_FAST_cmpxchg () { &__NR_FAST_cmpxchg;}' unless defined(&SYS_FAST_cmpxchg);
}
if(defined(&__NR_FAST_cmpxchg64)) {
    eval 'sub SYS_FAST_cmpxchg64 () { &__NR_FAST_cmpxchg64;}' unless defined(&SYS_FAST_cmpxchg64);
}
if(defined(&__NR__llseek)) {
    eval 'sub SYS__llseek () { &__NR__llseek;}' unless defined(&SYS__llseek);
}
if(defined(&__NR__newselect)) {
    eval 'sub SYS__newselect () { &__NR__newselect;}' unless defined(&SYS__newselect);
}
if(defined(&__NR__sysctl)) {
    eval 'sub SYS__sysctl () { &__NR__sysctl;}' unless defined(&SYS__sysctl);
}
if(defined(&__NR_accept)) {
    eval 'sub SYS_accept () { &__NR_accept;}' unless defined(&SYS_accept);
}
if(defined(&__NR_accept4)) {
    eval 'sub SYS_accept4 () { &__NR_accept4;}' unless defined(&SYS_accept4);
}
if(defined(&__NR_access)) {
    eval 'sub SYS_access () { &__NR_access;}' unless defined(&SYS_access);
}
if(defined(&__NR_acct)) {
    eval 'sub SYS_acct () { &__NR_acct;}' unless defined(&SYS_acct);
}
if(defined(&__NR_acl_get)) {
    eval 'sub SYS_acl_get () { &__NR_acl_get;}' unless defined(&SYS_acl_get);
}
if(defined(&__NR_acl_set)) {
    eval 'sub SYS_acl_set () { &__NR_acl_set;}' unless defined(&SYS_acl_set);
}
if(defined(&__NR_add_key)) {
    eval 'sub SYS_add_key () { &__NR_add_key;}' unless defined(&SYS_add_key);
}
if(defined(&__NR_adjtimex)) {
    eval 'sub SYS_adjtimex () { &__NR_adjtimex;}' unless defined(&SYS_adjtimex);
}
if(defined(&__NR_afs_syscall)) {
    eval 'sub SYS_afs_syscall () { &__NR_afs_syscall;}' unless defined(&SYS_afs_syscall);
}
if(defined(&__NR_alarm)) {
    eval 'sub SYS_alarm () { &__NR_alarm;}' unless defined(&SYS_alarm);
}
if(defined(&__NR_alloc_hugepages)) {
    eval 'sub SYS_alloc_hugepages () { &__NR_alloc_hugepages;}' unless defined(&SYS_alloc_hugepages);
}
if(defined(&__NR_arc_gettls)) {
    eval 'sub SYS_arc_gettls () { &__NR_arc_gettls;}' unless defined(&SYS_arc_gettls);
}
if(defined(&__NR_arc_settls)) {
    eval 'sub SYS_arc_settls () { &__NR_arc_settls;}' unless defined(&SYS_arc_settls);
}
if(defined(&__NR_arc_usr_cmpxchg)) {
    eval 'sub SYS_arc_usr_cmpxchg () { &__NR_arc_usr_cmpxchg;}' unless defined(&SYS_arc_usr_cmpxchg);
}
if(defined(&__NR_arch_prctl)) {
    eval 'sub SYS_arch_prctl () { &__NR_arch_prctl;}' unless defined(&SYS_arch_prctl);
}
if(defined(&__NR_arm_fadvise64_64)) {
    eval 'sub SYS_arm_fadvise64_64 () { &__NR_arm_fadvise64_64;}' unless defined(&SYS_arm_fadvise64_64);
}
if(defined(&__NR_arm_sync_file_range)) {
    eval 'sub SYS_arm_sync_file_range () { &__NR_arm_sync_file_range;}' unless defined(&SYS_arm_sync_file_range);
}
if(defined(&__NR_atomic_barrier)) {
    eval 'sub SYS_atomic_barrier () { &__NR_atomic_barrier;}' unless defined(&SYS_atomic_barrier);
}
if(defined(&__NR_atomic_cmpxchg_32)) {
    eval 'sub SYS_atomic_cmpxchg_32 () { &__NR_atomic_cmpxchg_32;}' unless defined(&SYS_atomic_cmpxchg_32);
}
if(defined(&__NR_attrctl)) {
    eval 'sub SYS_attrctl () { &__NR_attrctl;}' unless defined(&SYS_attrctl);
}
if(defined(&__NR_bdflush)) {
    eval 'sub SYS_bdflush () { &__NR_bdflush;}' unless defined(&SYS_bdflush);
}
if(defined(&__NR_bind)) {
    eval 'sub SYS_bind () { &__NR_bind;}' unless defined(&SYS_bind);
}
if(defined(&__NR_bpf)) {
    eval 'sub SYS_bpf () { &__NR_bpf;}' unless defined(&SYS_bpf);
}
if(defined(&__NR_break)) {
    eval 'sub SYS_break () { &__NR_break;}' unless defined(&SYS_break);
}
if(defined(&__NR_breakpoint)) {
    eval 'sub SYS_breakpoint () { &__NR_breakpoint;}' unless defined(&SYS_breakpoint);
}
if(defined(&__NR_brk)) {
    eval 'sub SYS_brk () { &__NR_brk;}' unless defined(&SYS_brk);
}
if(defined(&__NR_cachectl)) {
    eval 'sub SYS_cachectl () { &__NR_cachectl;}' unless defined(&SYS_cachectl);
}
if(defined(&__NR_cacheflush)) {
    eval 'sub SYS_cacheflush () { &__NR_cacheflush;}' unless defined(&SYS_cacheflush);
}
if(defined(&__NR_capget)) {
    eval 'sub SYS_capget () { &__NR_capget;}' unless defined(&SYS_capget);
}
if(defined(&__NR_capset)) {
    eval 'sub SYS_capset () { &__NR_capset;}' unless defined(&SYS_capset);
}
if(defined(&__NR_chdir)) {
    eval 'sub SYS_chdir () { &__NR_chdir;}' unless defined(&SYS_chdir);
}
if(defined(&__NR_chmod)) {
    eval 'sub SYS_chmod () { &__NR_chmod;}' unless defined(&SYS_chmod);
}
if(defined(&__NR_chown)) {
    eval 'sub SYS_chown () { &__NR_chown;}' unless defined(&SYS_chown);
}
if(defined(&__NR_chown32)) {
    eval 'sub SYS_chown32 () { &__NR_chown32;}' unless defined(&SYS_chown32);
}
if(defined(&__NR_chroot)) {
    eval 'sub SYS_chroot () { &__NR_chroot;}' unless defined(&SYS_chroot);
}
if(defined(&__NR_clock_adjtime)) {
    eval 'sub SYS_clock_adjtime () { &__NR_clock_adjtime;}' unless defined(&SYS_clock_adjtime);
}
if(defined(&__NR_clock_adjtime64)) {
    eval 'sub SYS_clock_adjtime64 () { &__NR_clock_adjtime64;}' unless defined(&SYS_clock_adjtime64);
}
if(defined(&__NR_clock_getres)) {
    eval 'sub SYS_clock_getres () { &__NR_clock_getres;}' unless defined(&SYS_clock_getres);
}
if(defined(&__NR_clock_getres_time64)) {
    eval 'sub SYS_clock_getres_time64 () { &__NR_clock_getres_time64;}' unless defined(&SYS_clock_getres_time64);
}
if(defined(&__NR_clock_gettime)) {
    eval 'sub SYS_clock_gettime () { &__NR_clock_gettime;}' unless defined(&SYS_clock_gettime);
}
if(defined(&__NR_clock_gettime64)) {
    eval 'sub SYS_clock_gettime64 () { &__NR_clock_gettime64;}' unless defined(&SYS_clock_gettime64);
}
if(defined(&__NR_clock_nanosleep)) {
    eval 'sub SYS_clock_nanosleep () { &__NR_clock_nanosleep;}' unless defined(&SYS_clock_nanosleep);
}
if(defined(&__NR_clock_nanosleep_time64)) {
    eval 'sub SYS_clock_nanosleep_time64 () { &__NR_clock_nanosleep_time64;}' unless defined(&SYS_clock_nanosleep_time64);
}
if(defined(&__NR_clock_settime)) {
    eval 'sub SYS_clock_settime () { &__NR_clock_settime;}' unless defined(&SYS_clock_settime);
}
if(defined(&__NR_clock_settime64)) {
    eval 'sub SYS_clock_settime64 () { &__NR_clock_settime64;}' unless defined(&SYS_clock_settime64);
}
if(defined(&__NR_clone)) {
    eval 'sub SYS_clone () { &__NR_clone;}' unless defined(&SYS_clone);
}
if(defined(&__NR_clone2)) {
    eval 'sub SYS_clone2 () { &__NR_clone2;}' unless defined(&SYS_clone2);
}
if(defined(&__NR_clone3)) {
    eval 'sub SYS_clone3 () { &__NR_clone3;}' unless defined(&SYS_clone3);
}
if(defined(&__NR_close)) {
    eval 'sub SYS_close () { &__NR_close;}' unless defined(&SYS_close);
}
if(defined(&__NR_close_range)) {
    eval 'sub SYS_close_range () { &__NR_close_range;}' unless defined(&SYS_close_range);
}
if(defined(&__NR_cmpxchg_badaddr)) {
    eval 'sub SYS_cmpxchg_badaddr () { &__NR_cmpxchg_badaddr;}' unless defined(&SYS_cmpxchg_badaddr);
}
if(defined(&__NR_connect)) {
    eval 'sub SYS_connect () { &__NR_connect;}' unless defined(&SYS_connect);
}
if(defined(&__NR_copy_file_range)) {
    eval 'sub SYS_copy_file_range () { &__NR_copy_file_range;}' unless defined(&SYS_copy_file_range);
}
if(defined(&__NR_creat)) {
    eval 'sub SYS_creat () { &__NR_creat;}' unless defined(&SYS_creat);
}
if(defined(&__NR_create_module)) {
    eval 'sub SYS_create_module () { &__NR_create_module;}' unless defined(&SYS_create_module);
}
if(defined(&__NR_delete_module)) {
    eval 'sub SYS_delete_module () { &__NR_delete_module;}' unless defined(&SYS_delete_module);
}
if(defined(&__NR_dipc)) {
    eval 'sub SYS_dipc () { &__NR_dipc;}' unless defined(&SYS_dipc);
}
if(defined(&__NR_dup)) {
    eval 'sub SYS_dup () { &__NR_dup;}' unless defined(&SYS_dup);
}
if(defined(&__NR_dup2)) {
    eval 'sub SYS_dup2 () { &__NR_dup2;}' unless defined(&SYS_dup2);
}
if(defined(&__NR_dup3)) {
    eval 'sub SYS_dup3 () { &__NR_dup3;}' unless defined(&SYS_dup3);
}
if(defined(&__NR_epoll_create)) {
    eval 'sub SYS_epoll_create () { &__NR_epoll_create;}' unless defined(&SYS_epoll_create);
}
if(defined(&__NR_epoll_create1)) {
    eval 'sub SYS_epoll_create1 () { &__NR_epoll_create1;}' unless defined(&SYS_epoll_create1);
}
if(defined(&__NR_epoll_ctl)) {
    eval 'sub SYS_epoll_ctl () { &__NR_epoll_ctl;}' unless defined(&SYS_epoll_ctl);
}
if(defined(&__NR_epoll_ctl_old)) {
    eval 'sub SYS_epoll_ctl_old () { &__NR_epoll_ctl_old;}' unless defined(&SYS_epoll_ctl_old);
}
if(defined(&__NR_epoll_pwait)) {
    eval 'sub SYS_epoll_pwait () { &__NR_epoll_pwait;}' unless defined(&SYS_epoll_pwait);
}
if(defined(&__NR_epoll_pwait2)) {
    eval 'sub SYS_epoll_pwait2 () { &__NR_epoll_pwait2;}' unless defined(&SYS_epoll_pwait2);
}
if(defined(&__NR_epoll_wait)) {
    eval 'sub SYS_epoll_wait () { &__NR_epoll_wait;}' unless defined(&SYS_epoll_wait);
}
if(defined(&__NR_epoll_wait_old)) {
    eval 'sub SYS_epoll_wait_old () { &__NR_epoll_wait_old;}' unless defined(&SYS_epoll_wait_old);
}
if(defined(&__NR_eventfd)) {
    eval 'sub SYS_eventfd () { &__NR_eventfd;}' unless defined(&SYS_eventfd);
}
if(defined(&__NR_eventfd2)) {
    eval 'sub SYS_eventfd2 () { &__NR_eventfd2;}' unless defined(&SYS_eventfd2);
}
if(defined(&__NR_exec_with_loader)) {
    eval 'sub SYS_exec_with_loader () { &__NR_exec_with_loader;}' unless defined(&SYS_exec_with_loader);
}
if(defined(&__NR_execv)) {
    eval 'sub SYS_execv () { &__NR_execv;}' unless defined(&SYS_execv);
}
if(defined(&__NR_execve)) {
    eval 'sub SYS_execve () { &__NR_execve;}' unless defined(&SYS_execve);
}
if(defined(&__NR_execveat)) {
    eval 'sub SYS_execveat () { &__NR_execveat;}' unless defined(&SYS_execveat);
}
if(defined(&__NR_exit)) {
    eval 'sub SYS_exit () { &__NR_exit;}' unless defined(&SYS_exit);
}
if(defined(&__NR_exit_group)) {
    eval 'sub SYS_exit_group () { &__NR_exit_group;}' unless defined(&SYS_exit_group);
}
if(defined(&__NR_faccessat)) {
    eval 'sub SYS_faccessat () { &__NR_faccessat;}' unless defined(&SYS_faccessat);
}
if(defined(&__NR_faccessat2)) {
    eval 'sub SYS_faccessat2 () { &__NR_faccessat2;}' unless defined(&SYS_faccessat2);
}
if(defined(&__NR_fadvise64)) {
    eval 'sub SYS_fadvise64 () { &__NR_fadvise64;}' unless defined(&SYS_fadvise64);
}
if(defined(&__NR_fadvise64_64)) {
    eval 'sub SYS_fadvise64_64 () { &__NR_fadvise64_64;}' unless defined(&SYS_fadvise64_64);
}
if(defined(&__NR_fallocate)) {
    eval 'sub SYS_fallocate () { &__NR_fallocate;}' unless defined(&SYS_fallocate);
}
if(defined(&__NR_fanotify_init)) {
    eval 'sub SYS_fanotify_init () { &__NR_fanotify_init;}' unless defined(&SYS_fanotify_init);
}
if(defined(&__NR_fanotify_mark)) {
    eval 'sub SYS_fanotify_mark () { &__NR_fanotify_mark;}' unless defined(&SYS_fanotify_mark);
}
if(defined(&__NR_fchdir)) {
    eval 'sub SYS_fchdir () { &__NR_fchdir;}' unless defined(&SYS_fchdir);
}
if(defined(&__NR_fchmod)) {
    eval 'sub SYS_fchmod () { &__NR_fchmod;}' unless defined(&SYS_fchmod);
}
if(defined(&__NR_fchmodat)) {
    eval 'sub SYS_fchmodat () { &__NR_fchmodat;}' unless defined(&SYS_fchmodat);
}
if(defined(&__NR_fchown)) {
    eval 'sub SYS_fchown () { &__NR_fchown;}' unless defined(&SYS_fchown);
}
if(defined(&__NR_fchown32)) {
    eval 'sub SYS_fchown32 () { &__NR_fchown32;}' unless defined(&SYS_fchown32);
}
if(defined(&__NR_fchownat)) {
    eval 'sub SYS_fchownat () { &__NR_fchownat;}' unless defined(&SYS_fchownat);
}
if(defined(&__NR_fcntl)) {
    eval 'sub SYS_fcntl () { &__NR_fcntl;}' unless defined(&SYS_fcntl);
}
if(defined(&__NR_fcntl64)) {
    eval 'sub SYS_fcntl64 () { &__NR_fcntl64;}' unless defined(&SYS_fcntl64);
}
if(defined(&__NR_fdatasync)) {
    eval 'sub SYS_fdatasync () { &__NR_fdatasync;}' unless defined(&SYS_fdatasync);
}
if(defined(&__NR_fgetxattr)) {
    eval 'sub SYS_fgetxattr () { &__NR_fgetxattr;}' unless defined(&SYS_fgetxattr);
}
if(defined(&__NR_finit_module)) {
    eval 'sub SYS_finit_module () { &__NR_finit_module;}' unless defined(&SYS_finit_module);
}
if(defined(&__NR_flistxattr)) {
    eval 'sub SYS_flistxattr () { &__NR_flistxattr;}' unless defined(&SYS_flistxattr);
}
if(defined(&__NR_flock)) {
    eval 'sub SYS_flock () { &__NR_flock;}' unless defined(&SYS_flock);
}
if(defined(&__NR_fork)) {
    eval 'sub SYS_fork () { &__NR_fork;}' unless defined(&SYS_fork);
}
if(defined(&__NR_fp_udfiex_crtl)) {
    eval 'sub SYS_fp_udfiex_crtl () { &__NR_fp_udfiex_crtl;}' unless defined(&SYS_fp_udfiex_crtl);
}
if(defined(&__NR_free_hugepages)) {
    eval 'sub SYS_free_hugepages () { &__NR_free_hugepages;}' unless defined(&SYS_free_hugepages);
}
if(defined(&__NR_fremovexattr)) {
    eval 'sub SYS_fremovexattr () { &__NR_fremovexattr;}' unless defined(&SYS_fremovexattr);
}
if(defined(&__NR_fsconfig)) {
    eval 'sub SYS_fsconfig () { &__NR_fsconfig;}' unless defined(&SYS_fsconfig);
}
if(defined(&__NR_fsetxattr)) {
    eval 'sub SYS_fsetxattr () { &__NR_fsetxattr;}' unless defined(&SYS_fsetxattr);
}
if(defined(&__NR_fsmount)) {
    eval 'sub SYS_fsmount () { &__NR_fsmount;}' unless defined(&SYS_fsmount);
}
if(defined(&__NR_fsopen)) {
    eval 'sub SYS_fsopen () { &__NR_fsopen;}' unless defined(&SYS_fsopen);
}
if(defined(&__NR_fspick)) {
    eval 'sub SYS_fspick () { &__NR_fspick;}' unless defined(&SYS_fspick);
}
if(defined(&__NR_fstat)) {
    eval 'sub SYS_fstat () { &__NR_fstat;}' unless defined(&SYS_fstat);
}
if(defined(&__NR_fstat64)) {
    eval 'sub SYS_fstat64 () { &__NR_fstat64;}' unless defined(&SYS_fstat64);
}
if(defined(&__NR_fstatat64)) {
    eval 'sub SYS_fstatat64 () { &__NR_fstatat64;}' unless defined(&SYS_fstatat64);
}
if(defined(&__NR_fstatfs)) {
    eval 'sub SYS_fstatfs () { &__NR_fstatfs;}' unless defined(&SYS_fstatfs);
}
if(defined(&__NR_fstatfs64)) {
    eval 'sub SYS_fstatfs64 () { &__NR_fstatfs64;}' unless defined(&SYS_fstatfs64);
}
if(defined(&__NR_fsync)) {
    eval 'sub SYS_fsync () { &__NR_fsync;}' unless defined(&SYS_fsync);
}
if(defined(&__NR_ftime)) {
    eval 'sub SYS_ftime () { &__NR_ftime;}' unless defined(&SYS_ftime);
}
if(defined(&__NR_ftruncate)) {
    eval 'sub SYS_ftruncate () { &__NR_ftruncate;}' unless defined(&SYS_ftruncate);
}
if(defined(&__NR_ftruncate64)) {
    eval 'sub SYS_ftruncate64 () { &__NR_ftruncate64;}' unless defined(&SYS_ftruncate64);
}
if(defined(&__NR_futex)) {
    eval 'sub SYS_futex () { &__NR_futex;}' unless defined(&SYS_futex);
}
if(defined(&__NR_futex_time64)) {
    eval 'sub SYS_futex_time64 () { &__NR_futex_time64;}' unless defined(&SYS_futex_time64);
}
if(defined(&__NR_futex_waitv)) {
    eval 'sub SYS_futex_waitv () { &__NR_futex_waitv;}' unless defined(&SYS_futex_waitv);
}
if(defined(&__NR_futimesat)) {
    eval 'sub SYS_futimesat () { &__NR_futimesat;}' unless defined(&SYS_futimesat);
}
if(defined(&__NR_get_kernel_syms)) {
    eval 'sub SYS_get_kernel_syms () { &__NR_get_kernel_syms;}' unless defined(&SYS_get_kernel_syms);
}
if(defined(&__NR_get_mempolicy)) {
    eval 'sub SYS_get_mempolicy () { &__NR_get_mempolicy;}' unless defined(&SYS_get_mempolicy);
}
if(defined(&__NR_get_robust_list)) {
    eval 'sub SYS_get_robust_list () { &__NR_get_robust_list;}' unless defined(&SYS_get_robust_list);
}
if(defined(&__NR_get_thread_area)) {
    eval 'sub SYS_get_thread_area () { &__NR_get_thread_area;}' unless defined(&SYS_get_thread_area);
}
if(defined(&__NR_get_tls)) {
    eval 'sub SYS_get_tls () { &__NR_get_tls;}' unless defined(&SYS_get_tls);
}
if(defined(&__NR_getcpu)) {
    eval 'sub SYS_getcpu () { &__NR_getcpu;}' unless defined(&SYS_getcpu);
}
if(defined(&__NR_getcwd)) {
    eval 'sub SYS_getcwd () { &__NR_getcwd;}' unless defined(&SYS_getcwd);
}
if(defined(&__NR_getdents)) {
    eval 'sub SYS_getdents () { &__NR_getdents;}' unless defined(&SYS_getdents);
}
if(defined(&__NR_getdents64)) {
    eval 'sub SYS_getdents64 () { &__NR_getdents64;}' unless defined(&SYS_getdents64);
}
if(defined(&__NR_getdomainname)) {
    eval 'sub SYS_getdomainname () { &__NR_getdomainname;}' unless defined(&SYS_getdomainname);
}
if(defined(&__NR_getdtablesize)) {
    eval 'sub SYS_getdtablesize () { &__NR_getdtablesize;}' unless defined(&SYS_getdtablesize);
}
if(defined(&__NR_getegid)) {
    eval 'sub SYS_getegid () { &__NR_getegid;}' unless defined(&SYS_getegid);
}
if(defined(&__NR_getegid32)) {
    eval 'sub SYS_getegid32 () { &__NR_getegid32;}' unless defined(&SYS_getegid32);
}
if(defined(&__NR_geteuid)) {
    eval 'sub SYS_geteuid () { &__NR_geteuid;}' unless defined(&SYS_geteuid);
}
if(defined(&__NR_geteuid32)) {
    eval 'sub SYS_geteuid32 () { &__NR_geteuid32;}' unless defined(&SYS_geteuid32);
}
if(defined(&__NR_getgid)) {
    eval 'sub SYS_getgid () { &__NR_getgid;}' unless defined(&SYS_getgid);
}
if(defined(&__NR_getgid32)) {
    eval 'sub SYS_getgid32 () { &__NR_getgid32;}' unless defined(&SYS_getgid32);
}
if(defined(&__NR_getgroups)) {
    eval 'sub SYS_getgroups () { &__NR_getgroups;}' unless defined(&SYS_getgroups);
}
if(defined(&__NR_getgroups32)) {
    eval 'sub SYS_getgroups32 () { &__NR_getgroups32;}' unless defined(&SYS_getgroups32);
}
if(defined(&__NR_gethostname)) {
    eval 'sub SYS_gethostname () { &__NR_gethostname;}' unless defined(&SYS_gethostname);
}
if(defined(&__NR_getitimer)) {
    eval 'sub SYS_getitimer () { &__NR_getitimer;}' unless defined(&SYS_getitimer);
}
if(defined(&__NR_getpagesize)) {
    eval 'sub SYS_getpagesize () { &__NR_getpagesize;}' unless defined(&SYS_getpagesize);
}
if(defined(&__NR_getpeername)) {
    eval 'sub SYS_getpeername () { &__NR_getpeername;}' unless defined(&SYS_getpeername);
}
if(defined(&__NR_getpgid)) {
    eval 'sub SYS_getpgid () { &__NR_getpgid;}' unless defined(&SYS_getpgid);
}
if(defined(&__NR_getpgrp)) {
    eval 'sub SYS_getpgrp () { &__NR_getpgrp;}' unless defined(&SYS_getpgrp);
}
if(defined(&__NR_getpid)) {
    eval 'sub SYS_getpid () { &__NR_getpid;}' unless defined(&SYS_getpid);
}
if(defined(&__NR_getpmsg)) {
    eval 'sub SYS_getpmsg () { &__NR_getpmsg;}' unless defined(&SYS_getpmsg);
}
if(defined(&__NR_getppid)) {
    eval 'sub SYS_getppid () { &__NR_getppid;}' unless defined(&SYS_getppid);
}
if(defined(&__NR_getpriority)) {
    eval 'sub SYS_getpriority () { &__NR_getpriority;}' unless defined(&SYS_getpriority);
}
if(defined(&__NR_getrandom)) {
    eval 'sub SYS_getrandom () { &__NR_getrandom;}' unless defined(&SYS_getrandom);
}
if(defined(&__NR_getresgid)) {
    eval 'sub SYS_getresgid () { &__NR_getresgid;}' unless defined(&SYS_getresgid);
}
if(defined(&__NR_getresgid32)) {
    eval 'sub SYS_getresgid32 () { &__NR_getresgid32;}' unless defined(&SYS_getresgid32);
}
if(defined(&__NR_getresuid)) {
    eval 'sub SYS_getresuid () { &__NR_getresuid;}' unless defined(&SYS_getresuid);
}
if(defined(&__NR_getresuid32)) {
    eval 'sub SYS_getresuid32 () { &__NR_getresuid32;}' unless defined(&SYS_getresuid32);
}
if(defined(&__NR_getrlimit)) {
    eval 'sub SYS_getrlimit () { &__NR_getrlimit;}' unless defined(&SYS_getrlimit);
}
if(defined(&__NR_getrusage)) {
    eval 'sub SYS_getrusage () { &__NR_getrusage;}' unless defined(&SYS_getrusage);
}
if(defined(&__NR_getsid)) {
    eval 'sub SYS_getsid () { &__NR_getsid;}' unless defined(&SYS_getsid);
}
if(defined(&__NR_getsockname)) {
    eval 'sub SYS_getsockname () { &__NR_getsockname;}' unless defined(&SYS_getsockname);
}
if(defined(&__NR_getsockopt)) {
    eval 'sub SYS_getsockopt () { &__NR_getsockopt;}' unless defined(&SYS_getsockopt);
}
if(defined(&__NR_gettid)) {
    eval 'sub SYS_gettid () { &__NR_gettid;}' unless defined(&SYS_gettid);
}
if(defined(&__NR_gettimeofday)) {
    eval 'sub SYS_gettimeofday () { &__NR_gettimeofday;}' unless defined(&SYS_gettimeofday);
}
if(defined(&__NR_getuid)) {
    eval 'sub SYS_getuid () { &__NR_getuid;}' unless defined(&SYS_getuid);
}
if(defined(&__NR_getuid32)) {
    eval 'sub SYS_getuid32 () { &__NR_getuid32;}' unless defined(&SYS_getuid32);
}
if(defined(&__NR_getunwind)) {
    eval 'sub SYS_getunwind () { &__NR_getunwind;}' unless defined(&SYS_getunwind);
}
if(defined(&__NR_getxattr)) {
    eval 'sub SYS_getxattr () { &__NR_getxattr;}' unless defined(&SYS_getxattr);
}
if(defined(&__NR_getxgid)) {
    eval 'sub SYS_getxgid () { &__NR_getxgid;}' unless defined(&SYS_getxgid);
}
if(defined(&__NR_getxpid)) {
    eval 'sub SYS_getxpid () { &__NR_getxpid;}' unless defined(&SYS_getxpid);
}
if(defined(&__NR_getxuid)) {
    eval 'sub SYS_getxuid () { &__NR_getxuid;}' unless defined(&SYS_getxuid);
}
if(defined(&__NR_gtty)) {
    eval 'sub SYS_gtty () { &__NR_gtty;}' unless defined(&SYS_gtty);
}
if(defined(&__NR_idle)) {
    eval 'sub SYS_idle () { &__NR_idle;}' unless defined(&SYS_idle);
}
if(defined(&__NR_init_module)) {
    eval 'sub SYS_init_module () { &__NR_init_module;}' unless defined(&SYS_init_module);
}
if(defined(&__NR_inotify_add_watch)) {
    eval 'sub SYS_inotify_add_watch () { &__NR_inotify_add_watch;}' unless defined(&SYS_inotify_add_watch);
}
if(defined(&__NR_inotify_init)) {
    eval 'sub SYS_inotify_init () { &__NR_inotify_init;}' unless defined(&SYS_inotify_init);
}
if(defined(&__NR_inotify_init1)) {
    eval 'sub SYS_inotify_init1 () { &__NR_inotify_init1;}' unless defined(&SYS_inotify_init1);
}
if(defined(&__NR_inotify_rm_watch)) {
    eval 'sub SYS_inotify_rm_watch () { &__NR_inotify_rm_watch;}' unless defined(&SYS_inotify_rm_watch);
}
if(defined(&__NR_io_cancel)) {
    eval 'sub SYS_io_cancel () { &__NR_io_cancel;}' unless defined(&SYS_io_cancel);
}
if(defined(&__NR_io_destroy)) {
    eval 'sub SYS_io_destroy () { &__NR_io_destroy;}' unless defined(&SYS_io_destroy);
}
if(defined(&__NR_io_getevents)) {
    eval 'sub SYS_io_getevents () { &__NR_io_getevents;}' unless defined(&SYS_io_getevents);
}
if(defined(&__NR_io_pgetevents)) {
    eval 'sub SYS_io_pgetevents () { &__NR_io_pgetevents;}' unless defined(&SYS_io_pgetevents);
}
if(defined(&__NR_io_pgetevents_time64)) {
    eval 'sub SYS_io_pgetevents_time64 () { &__NR_io_pgetevents_time64;}' unless defined(&SYS_io_pgetevents_time64);
}
if(defined(&__NR_io_setup)) {
    eval 'sub SYS_io_setup () { &__NR_io_setup;}' unless defined(&SYS_io_setup);
}
if(defined(&__NR_io_submit)) {
    eval 'sub SYS_io_submit () { &__NR_io_submit;}' unless defined(&SYS_io_submit);
}
if(defined(&__NR_io_uring_enter)) {
    eval 'sub SYS_io_uring_enter () { &__NR_io_uring_enter;}' unless defined(&SYS_io_uring_enter);
}
if(defined(&__NR_io_uring_register)) {
    eval 'sub SYS_io_uring_register () { &__NR_io_uring_register;}' unless defined(&SYS_io_uring_register);
}
if(defined(&__NR_io_uring_setup)) {
    eval 'sub SYS_io_uring_setup () { &__NR_io_uring_setup;}' unless defined(&SYS_io_uring_setup);
}
if(defined(&__NR_ioctl)) {
    eval 'sub SYS_ioctl () { &__NR_ioctl;}' unless defined(&SYS_ioctl);
}
if(defined(&__NR_ioperm)) {
    eval 'sub SYS_ioperm () { &__NR_ioperm;}' unless defined(&SYS_ioperm);
}
if(defined(&__NR_iopl)) {
    eval 'sub SYS_iopl () { &__NR_iopl;}' unless defined(&SYS_iopl);
}
if(defined(&__NR_ioprio_get)) {
    eval 'sub SYS_ioprio_get () { &__NR_ioprio_get;}' unless defined(&SYS_ioprio_get);
}
if(defined(&__NR_ioprio_set)) {
    eval 'sub SYS_ioprio_set () { &__NR_ioprio_set;}' unless defined(&SYS_ioprio_set);
}
if(defined(&__NR_ipc)) {
    eval 'sub SYS_ipc () { &__NR_ipc;}' unless defined(&SYS_ipc);
}
if(defined(&__NR_kcmp)) {
    eval 'sub SYS_kcmp () { &__NR_kcmp;}' unless defined(&SYS_kcmp);
}
if(defined(&__NR_kern_features)) {
    eval 'sub SYS_kern_features () { &__NR_kern_features;}' unless defined(&SYS_kern_features);
}
if(defined(&__NR_kexec_file_load)) {
    eval 'sub SYS_kexec_file_load () { &__NR_kexec_file_load;}' unless defined(&SYS_kexec_file_load);
}
if(defined(&__NR_kexec_load)) {
    eval 'sub SYS_kexec_load () { &__NR_kexec_load;}' unless defined(&SYS_kexec_load);
}
if(defined(&__NR_keyctl)) {
    eval 'sub SYS_keyctl () { &__NR_keyctl;}' unless defined(&SYS_keyctl);
}
if(defined(&__NR_kill)) {
    eval 'sub SYS_kill () { &__NR_kill;}' unless defined(&SYS_kill);
}
if(defined(&__NR_landlock_add_rule)) {
    eval 'sub SYS_landlock_add_rule () { &__NR_landlock_add_rule;}' unless defined(&SYS_landlock_add_rule);
}
if(defined(&__NR_landlock_create_ruleset)) {
    eval 'sub SYS_landlock_create_ruleset () { &__NR_landlock_create_ruleset;}' unless defined(&SYS_landlock_create_ruleset);
}
if(defined(&__NR_landlock_restrict_self)) {
    eval 'sub SYS_landlock_restrict_self () { &__NR_landlock_restrict_self;}' unless defined(&SYS_landlock_restrict_self);
}
if(defined(&__NR_lchown)) {
    eval 'sub SYS_lchown () { &__NR_lchown;}' unless defined(&SYS_lchown);
}
if(defined(&__NR_lchown32)) {
    eval 'sub SYS_lchown32 () { &__NR_lchown32;}' unless defined(&SYS_lchown32);
}
if(defined(&__NR_lgetxattr)) {
    eval 'sub SYS_lgetxattr () { &__NR_lgetxattr;}' unless defined(&SYS_lgetxattr);
}
if(defined(&__NR_link)) {
    eval 'sub SYS_link () { &__NR_link;}' unless defined(&SYS_link);
}
if(defined(&__NR_linkat)) {
    eval 'sub SYS_linkat () { &__NR_linkat;}' unless defined(&SYS_linkat);
}
if(defined(&__NR_listen)) {
    eval 'sub SYS_listen () { &__NR_listen;}' unless defined(&SYS_listen);
}
if(defined(&__NR_listxattr)) {
    eval 'sub SYS_listxattr () { &__NR_listxattr;}' unless defined(&SYS_listxattr);
}
if(defined(&__NR_llistxattr)) {
    eval 'sub SYS_llistxattr () { &__NR_llistxattr;}' unless defined(&SYS_llistxattr);
}
if(defined(&__NR_llseek)) {
    eval 'sub SYS_llseek () { &__NR_llseek;}' unless defined(&SYS_llseek);
}
if(defined(&__NR_lock)) {
    eval 'sub SYS_lock () { &__NR_lock;}' unless defined(&SYS_lock);
}
if(defined(&__NR_lookup_dcookie)) {
    eval 'sub SYS_lookup_dcookie () { &__NR_lookup_dcookie;}' unless defined(&SYS_lookup_dcookie);
}
if(defined(&__NR_lremovexattr)) {
    eval 'sub SYS_lremovexattr () { &__NR_lremovexattr;}' unless defined(&SYS_lremovexattr);
}
if(defined(&__NR_lseek)) {
    eval 'sub SYS_lseek () { &__NR_lseek;}' unless defined(&SYS_lseek);
}
if(defined(&__NR_lsetxattr)) {
    eval 'sub SYS_lsetxattr () { &__NR_lsetxattr;}' unless defined(&SYS_lsetxattr);
}
if(defined(&__NR_lstat)) {
    eval 'sub SYS_lstat () { &__NR_lstat;}' unless defined(&SYS_lstat);
}
if(defined(&__NR_lstat64)) {
    eval 'sub SYS_lstat64 () { &__NR_lstat64;}' unless defined(&SYS_lstat64);
}
if(defined(&__NR_madvise)) {
    eval 'sub SYS_madvise () { &__NR_madvise;}' unless defined(&SYS_madvise);
}
if(defined(&__NR_mbind)) {
    eval 'sub SYS_mbind () { &__NR_mbind;}' unless defined(&SYS_mbind);
}
if(defined(&__NR_membarrier)) {
    eval 'sub SYS_membarrier () { &__NR_membarrier;}' unless defined(&SYS_membarrier);
}
if(defined(&__NR_memfd_create)) {
    eval 'sub SYS_memfd_create () { &__NR_memfd_create;}' unless defined(&SYS_memfd_create);
}
if(defined(&__NR_memfd_secret)) {
    eval 'sub SYS_memfd_secret () { &__NR_memfd_secret;}' unless defined(&SYS_memfd_secret);
}
if(defined(&__NR_memory_ordering)) {
    eval 'sub SYS_memory_ordering () { &__NR_memory_ordering;}' unless defined(&SYS_memory_ordering);
}
if(defined(&__NR_migrate_pages)) {
    eval 'sub SYS_migrate_pages () { &__NR_migrate_pages;}' unless defined(&SYS_migrate_pages);
}
if(defined(&__NR_mincore)) {
    eval 'sub SYS_mincore () { &__NR_mincore;}' unless defined(&SYS_mincore);
}
if(defined(&__NR_mkdir)) {
    eval 'sub SYS_mkdir () { &__NR_mkdir;}' unless defined(&SYS_mkdir);
}
if(defined(&__NR_mkdirat)) {
    eval 'sub SYS_mkdirat () { &__NR_mkdirat;}' unless defined(&SYS_mkdirat);
}
if(defined(&__NR_mknod)) {
    eval 'sub SYS_mknod () { &__NR_mknod;}' unless defined(&SYS_mknod);
}
if(defined(&__NR_mknodat)) {
    eval 'sub SYS_mknodat () { &__NR_mknodat;}' unless defined(&SYS_mknodat);
}
if(defined(&__NR_mlock)) {
    eval 'sub SYS_mlock () { &__NR_mlock;}' unless defined(&SYS_mlock);
}
if(defined(&__NR_mlock2)) {
    eval 'sub SYS_mlock2 () { &__NR_mlock2;}' unless defined(&SYS_mlock2);
}
if(defined(&__NR_mlockall)) {
    eval 'sub SYS_mlockall () { &__NR_mlockall;}' unless defined(&SYS_mlockall);
}
if(defined(&__NR_mmap)) {
    eval 'sub SYS_mmap () { &__NR_mmap;}' unless defined(&SYS_mmap);
}
if(defined(&__NR_mmap2)) {
    eval 'sub SYS_mmap2 () { &__NR_mmap2;}' unless defined(&SYS_mmap2);
}
if(defined(&__NR_modify_ldt)) {
    eval 'sub SYS_modify_ldt () { &__NR_modify_ldt;}' unless defined(&SYS_modify_ldt);
}
if(defined(&__NR_mount)) {
    eval 'sub SYS_mount () { &__NR_mount;}' unless defined(&SYS_mount);
}
if(defined(&__NR_mount_setattr)) {
    eval 'sub SYS_mount_setattr () { &__NR_mount_setattr;}' unless defined(&SYS_mount_setattr);
}
if(defined(&__NR_move_mount)) {
    eval 'sub SYS_move_mount () { &__NR_move_mount;}' unless defined(&SYS_move_mount);
}
if(defined(&__NR_move_pages)) {
    eval 'sub SYS_move_pages () { &__NR_move_pages;}' unless defined(&SYS_move_pages);
}
if(defined(&__NR_mprotect)) {
    eval 'sub SYS_mprotect () { &__NR_mprotect;}' unless defined(&SYS_mprotect);
}
if(defined(&__NR_mpx)) {
    eval 'sub SYS_mpx () { &__NR_mpx;}' unless defined(&SYS_mpx);
}
if(defined(&__NR_mq_getsetattr)) {
    eval 'sub SYS_mq_getsetattr () { &__NR_mq_getsetattr;}' unless defined(&SYS_mq_getsetattr);
}
if(defined(&__NR_mq_notify)) {
    eval 'sub SYS_mq_notify () { &__NR_mq_notify;}' unless defined(&SYS_mq_notify);
}
if(defined(&__NR_mq_open)) {
    eval 'sub SYS_mq_open () { &__NR_mq_open;}' unless defined(&SYS_mq_open);
}
if(defined(&__NR_mq_timedreceive)) {
    eval 'sub SYS_mq_timedreceive () { &__NR_mq_timedreceive;}' unless defined(&SYS_mq_timedreceive);
}
if(defined(&__NR_mq_timedreceive_time64)) {
    eval 'sub SYS_mq_timedreceive_time64 () { &__NR_mq_timedreceive_time64;}' unless defined(&SYS_mq_timedreceive_time64);
}
if(defined(&__NR_mq_timedsend)) {
    eval 'sub SYS_mq_timedsend () { &__NR_mq_timedsend;}' unless defined(&SYS_mq_timedsend);
}
if(defined(&__NR_mq_timedsend_time64)) {
    eval 'sub SYS_mq_timedsend_time64 () { &__NR_mq_timedsend_time64;}' unless defined(&SYS_mq_timedsend_time64);
}
if(defined(&__NR_mq_unlink)) {
    eval 'sub SYS_mq_unlink () { &__NR_mq_unlink;}' unless defined(&SYS_mq_unlink);
}
if(defined(&__NR_mremap)) {
    eval 'sub SYS_mremap () { &__NR_mremap;}' unless defined(&SYS_mremap);
}
if(defined(&__NR_msgctl)) {
    eval 'sub SYS_msgctl () { &__NR_msgctl;}' unless defined(&SYS_msgctl);
}
if(defined(&__NR_msgget)) {
    eval 'sub SYS_msgget () { &__NR_msgget;}' unless defined(&SYS_msgget);
}
if(defined(&__NR_msgrcv)) {
    eval 'sub SYS_msgrcv () { &__NR_msgrcv;}' unless defined(&SYS_msgrcv);
}
if(defined(&__NR_msgsnd)) {
    eval 'sub SYS_msgsnd () { &__NR_msgsnd;}' unless defined(&SYS_msgsnd);
}
if(defined(&__NR_msync)) {
    eval 'sub SYS_msync () { &__NR_msync;}' unless defined(&SYS_msync);
}
if(defined(&__NR_multiplexer)) {
    eval 'sub SYS_multiplexer () { &__NR_multiplexer;}' unless defined(&SYS_multiplexer);
}
if(defined(&__NR_munlock)) {
    eval 'sub SYS_munlock () { &__NR_munlock;}' unless defined(&SYS_munlock);
}
if(defined(&__NR_munlockall)) {
    eval 'sub SYS_munlockall () { &__NR_munlockall;}' unless defined(&SYS_munlockall);
}
if(defined(&__NR_munmap)) {
    eval 'sub SYS_munmap () { &__NR_munmap;}' unless defined(&SYS_munmap);
}
if(defined(&__NR_name_to_handle_at)) {
    eval 'sub SYS_name_to_handle_at () { &__NR_name_to_handle_at;}' unless defined(&SYS_name_to_handle_at);
}
if(defined(&__NR_nanosleep)) {
    eval 'sub SYS_nanosleep () { &__NR_nanosleep;}' unless defined(&SYS_nanosleep);
}
if(defined(&__NR_newfstatat)) {
    eval 'sub SYS_newfstatat () { &__NR_newfstatat;}' unless defined(&SYS_newfstatat);
}
if(defined(&__NR_nfsservctl)) {
    eval 'sub SYS_nfsservctl () { &__NR_nfsservctl;}' unless defined(&SYS_nfsservctl);
}
if(defined(&__NR_ni_syscall)) {
    eval 'sub SYS_ni_syscall () { &__NR_ni_syscall;}' unless defined(&SYS_ni_syscall);
}
if(defined(&__NR_nice)) {
    eval 'sub SYS_nice () { &__NR_nice;}' unless defined(&SYS_nice);
}
if(defined(&__NR_old_adjtimex)) {
    eval 'sub SYS_old_adjtimex () { &__NR_old_adjtimex;}' unless defined(&SYS_old_adjtimex);
}
if(defined(&__NR_old_getpagesize)) {
    eval 'sub SYS_old_getpagesize () { &__NR_old_getpagesize;}' unless defined(&SYS_old_getpagesize);
}
if(defined(&__NR_oldfstat)) {
    eval 'sub SYS_oldfstat () { &__NR_oldfstat;}' unless defined(&SYS_oldfstat);
}
if(defined(&__NR_oldlstat)) {
    eval 'sub SYS_oldlstat () { &__NR_oldlstat;}' unless defined(&SYS_oldlstat);
}
if(defined(&__NR_oldolduname)) {
    eval 'sub SYS_oldolduname () { &__NR_oldolduname;}' unless defined(&SYS_oldolduname);
}
if(defined(&__NR_oldstat)) {
    eval 'sub SYS_oldstat () { &__NR_oldstat;}' unless defined(&SYS_oldstat);
}
if(defined(&__NR_oldumount)) {
    eval 'sub SYS_oldumount () { &__NR_oldumount;}' unless defined(&SYS_oldumount);
}
if(defined(&__NR_olduname)) {
    eval 'sub SYS_olduname () { &__NR_olduname;}' unless defined(&SYS_olduname);
}
if(defined(&__NR_open)) {
    eval 'sub SYS_open () { &__NR_open;}' unless defined(&SYS_open);
}
if(defined(&__NR_open_by_handle_at)) {
    eval 'sub SYS_open_by_handle_at () { &__NR_open_by_handle_at;}' unless defined(&SYS_open_by_handle_at);
}
if(defined(&__NR_open_tree)) {
    eval 'sub SYS_open_tree () { &__NR_open_tree;}' unless defined(&SYS_open_tree);
}
if(defined(&__NR_openat)) {
    eval 'sub SYS_openat () { &__NR_openat;}' unless defined(&SYS_openat);
}
if(defined(&__NR_openat2)) {
    eval 'sub SYS_openat2 () { &__NR_openat2;}' unless defined(&SYS_openat2);
}
if(defined(&__NR_or1k_atomic)) {
    eval 'sub SYS_or1k_atomic () { &__NR_or1k_atomic;}' unless defined(&SYS_or1k_atomic);
}
if(defined(&__NR_osf_adjtime)) {
    eval 'sub SYS_osf_adjtime () { &__NR_osf_adjtime;}' unless defined(&SYS_osf_adjtime);
}
if(defined(&__NR_osf_afs_syscall)) {
    eval 'sub SYS_osf_afs_syscall () { &__NR_osf_afs_syscall;}' unless defined(&SYS_osf_afs_syscall);
}
if(defined(&__NR_osf_alt_plock)) {
    eval 'sub SYS_osf_alt_plock () { &__NR_osf_alt_plock;}' unless defined(&SYS_osf_alt_plock);
}
if(defined(&__NR_osf_alt_setsid)) {
    eval 'sub SYS_osf_alt_setsid () { &__NR_osf_alt_setsid;}' unless defined(&SYS_osf_alt_setsid);
}
if(defined(&__NR_osf_alt_sigpending)) {
    eval 'sub SYS_osf_alt_sigpending () { &__NR_osf_alt_sigpending;}' unless defined(&SYS_osf_alt_sigpending);
}
if(defined(&__NR_osf_asynch_daemon)) {
    eval 'sub SYS_osf_asynch_daemon () { &__NR_osf_asynch_daemon;}' unless defined(&SYS_osf_asynch_daemon);
}
if(defined(&__NR_osf_audcntl)) {
    eval 'sub SYS_osf_audcntl () { &__NR_osf_audcntl;}' unless defined(&SYS_osf_audcntl);
}
if(defined(&__NR_osf_audgen)) {
    eval 'sub SYS_osf_audgen () { &__NR_osf_audgen;}' unless defined(&SYS_osf_audgen);
}
if(defined(&__NR_osf_chflags)) {
    eval 'sub SYS_osf_chflags () { &__NR_osf_chflags;}' unless defined(&SYS_osf_chflags);
}
if(defined(&__NR_osf_execve)) {
    eval 'sub SYS_osf_execve () { &__NR_osf_execve;}' unless defined(&SYS_osf_execve);
}
if(defined(&__NR_osf_exportfs)) {
    eval 'sub SYS_osf_exportfs () { &__NR_osf_exportfs;}' unless defined(&SYS_osf_exportfs);
}
if(defined(&__NR_osf_fchflags)) {
    eval 'sub SYS_osf_fchflags () { &__NR_osf_fchflags;}' unless defined(&SYS_osf_fchflags);
}
if(defined(&__NR_osf_fdatasync)) {
    eval 'sub SYS_osf_fdatasync () { &__NR_osf_fdatasync;}' unless defined(&SYS_osf_fdatasync);
}
if(defined(&__NR_osf_fpathconf)) {
    eval 'sub SYS_osf_fpathconf () { &__NR_osf_fpathconf;}' unless defined(&SYS_osf_fpathconf);
}
if(defined(&__NR_osf_fstat)) {
    eval 'sub SYS_osf_fstat () { &__NR_osf_fstat;}' unless defined(&SYS_osf_fstat);
}
if(defined(&__NR_osf_fstatfs)) {
    eval 'sub SYS_osf_fstatfs () { &__NR_osf_fstatfs;}' unless defined(&SYS_osf_fstatfs);
}
if(defined(&__NR_osf_fstatfs64)) {
    eval 'sub SYS_osf_fstatfs64 () { &__NR_osf_fstatfs64;}' unless defined(&SYS_osf_fstatfs64);
}
if(defined(&__NR_osf_fuser)) {
    eval 'sub SYS_osf_fuser () { &__NR_osf_fuser;}' unless defined(&SYS_osf_fuser);
}
if(defined(&__NR_osf_getaddressconf)) {
    eval 'sub SYS_osf_getaddressconf () { &__NR_osf_getaddressconf;}' unless defined(&SYS_osf_getaddressconf);
}
if(defined(&__NR_osf_getdirentries)) {
    eval 'sub SYS_osf_getdirentries () { &__NR_osf_getdirentries;}' unless defined(&SYS_osf_getdirentries);
}
if(defined(&__NR_osf_getdomainname)) {
    eval 'sub SYS_osf_getdomainname () { &__NR_osf_getdomainname;}' unless defined(&SYS_osf_getdomainname);
}
if(defined(&__NR_osf_getfh)) {
    eval 'sub SYS_osf_getfh () { &__NR_osf_getfh;}' unless defined(&SYS_osf_getfh);
}
if(defined(&__NR_osf_getfsstat)) {
    eval 'sub SYS_osf_getfsstat () { &__NR_osf_getfsstat;}' unless defined(&SYS_osf_getfsstat);
}
if(defined(&__NR_osf_gethostid)) {
    eval 'sub SYS_osf_gethostid () { &__NR_osf_gethostid;}' unless defined(&SYS_osf_gethostid);
}
if(defined(&__NR_osf_getitimer)) {
    eval 'sub SYS_osf_getitimer () { &__NR_osf_getitimer;}' unless defined(&SYS_osf_getitimer);
}
if(defined(&__NR_osf_getlogin)) {
    eval 'sub SYS_osf_getlogin () { &__NR_osf_getlogin;}' unless defined(&SYS_osf_getlogin);
}
if(defined(&__NR_osf_getmnt)) {
    eval 'sub SYS_osf_getmnt () { &__NR_osf_getmnt;}' unless defined(&SYS_osf_getmnt);
}
if(defined(&__NR_osf_getrusage)) {
    eval 'sub SYS_osf_getrusage () { &__NR_osf_getrusage;}' unless defined(&SYS_osf_getrusage);
}
if(defined(&__NR_osf_getsysinfo)) {
    eval 'sub SYS_osf_getsysinfo () { &__NR_osf_getsysinfo;}' unless defined(&SYS_osf_getsysinfo);
}
if(defined(&__NR_osf_gettimeofday)) {
    eval 'sub SYS_osf_gettimeofday () { &__NR_osf_gettimeofday;}' unless defined(&SYS_osf_gettimeofday);
}
if(defined(&__NR_osf_kloadcall)) {
    eval 'sub SYS_osf_kloadcall () { &__NR_osf_kloadcall;}' unless defined(&SYS_osf_kloadcall);
}
if(defined(&__NR_osf_kmodcall)) {
    eval 'sub SYS_osf_kmodcall () { &__NR_osf_kmodcall;}' unless defined(&SYS_osf_kmodcall);
}
if(defined(&__NR_osf_lstat)) {
    eval 'sub SYS_osf_lstat () { &__NR_osf_lstat;}' unless defined(&SYS_osf_lstat);
}
if(defined(&__NR_osf_memcntl)) {
    eval 'sub SYS_osf_memcntl () { &__NR_osf_memcntl;}' unless defined(&SYS_osf_memcntl);
}
if(defined(&__NR_osf_mincore)) {
    eval 'sub SYS_osf_mincore () { &__NR_osf_mincore;}' unless defined(&SYS_osf_mincore);
}
if(defined(&__NR_osf_mount)) {
    eval 'sub SYS_osf_mount () { &__NR_osf_mount;}' unless defined(&SYS_osf_mount);
}
if(defined(&__NR_osf_mremap)) {
    eval 'sub SYS_osf_mremap () { &__NR_osf_mremap;}' unless defined(&SYS_osf_mremap);
}
if(defined(&__NR_osf_msfs_syscall)) {
    eval 'sub SYS_osf_msfs_syscall () { &__NR_osf_msfs_syscall;}' unless defined(&SYS_osf_msfs_syscall);
}
if(defined(&__NR_osf_msleep)) {
    eval 'sub SYS_osf_msleep () { &__NR_osf_msleep;}' unless defined(&SYS_osf_msleep);
}
if(defined(&__NR_osf_mvalid)) {
    eval 'sub SYS_osf_mvalid () { &__NR_osf_mvalid;}' unless defined(&SYS_osf_mvalid);
}
if(defined(&__NR_osf_mwakeup)) {
    eval 'sub SYS_osf_mwakeup () { &__NR_osf_mwakeup;}' unless defined(&SYS_osf_mwakeup);
}
if(defined(&__NR_osf_naccept)) {
    eval 'sub SYS_osf_naccept () { &__NR_osf_naccept;}' unless defined(&SYS_osf_naccept);
}
if(defined(&__NR_osf_nfssvc)) {
    eval 'sub SYS_osf_nfssvc () { &__NR_osf_nfssvc;}' unless defined(&SYS_osf_nfssvc);
}
if(defined(&__NR_osf_ngetpeername)) {
    eval 'sub SYS_osf_ngetpeername () { &__NR_osf_ngetpeername;}' unless defined(&SYS_osf_ngetpeername);
}
if(defined(&__NR_osf_ngetsockname)) {
    eval 'sub SYS_osf_ngetsockname () { &__NR_osf_ngetsockname;}' unless defined(&SYS_osf_ngetsockname);
}
if(defined(&__NR_osf_nrecvfrom)) {
    eval 'sub SYS_osf_nrecvfrom () { &__NR_osf_nrecvfrom;}' unless defined(&SYS_osf_nrecvfrom);
}
if(defined(&__NR_osf_nrecvmsg)) {
    eval 'sub SYS_osf_nrecvmsg () { &__NR_osf_nrecvmsg;}' unless defined(&SYS_osf_nrecvmsg);
}
if(defined(&__NR_osf_nsendmsg)) {
    eval 'sub SYS_osf_nsendmsg () { &__NR_osf_nsendmsg;}' unless defined(&SYS_osf_nsendmsg);
}
if(defined(&__NR_osf_ntp_adjtime)) {
    eval 'sub SYS_osf_ntp_adjtime () { &__NR_osf_ntp_adjtime;}' unless defined(&SYS_osf_ntp_adjtime);
}
if(defined(&__NR_osf_ntp_gettime)) {
    eval 'sub SYS_osf_ntp_gettime () { &__NR_osf_ntp_gettime;}' unless defined(&SYS_osf_ntp_gettime);
}
if(defined(&__NR_osf_old_creat)) {
    eval 'sub SYS_osf_old_creat () { &__NR_osf_old_creat;}' unless defined(&SYS_osf_old_creat);
}
if(defined(&__NR_osf_old_fstat)) {
    eval 'sub SYS_osf_old_fstat () { &__NR_osf_old_fstat;}' unless defined(&SYS_osf_old_fstat);
}
if(defined(&__NR_osf_old_getpgrp)) {
    eval 'sub SYS_osf_old_getpgrp () { &__NR_osf_old_getpgrp;}' unless defined(&SYS_osf_old_getpgrp);
}
if(defined(&__NR_osf_old_killpg)) {
    eval 'sub SYS_osf_old_killpg () { &__NR_osf_old_killpg;}' unless defined(&SYS_osf_old_killpg);
}
if(defined(&__NR_osf_old_lstat)) {
    eval 'sub SYS_osf_old_lstat () { &__NR_osf_old_lstat;}' unless defined(&SYS_osf_old_lstat);
}
if(defined(&__NR_osf_old_open)) {
    eval 'sub SYS_osf_old_open () { &__NR_osf_old_open;}' unless defined(&SYS_osf_old_open);
}
if(defined(&__NR_osf_old_sigaction)) {
    eval 'sub SYS_osf_old_sigaction () { &__NR_osf_old_sigaction;}' unless defined(&SYS_osf_old_sigaction);
}
if(defined(&__NR_osf_old_sigblock)) {
    eval 'sub SYS_osf_old_sigblock () { &__NR_osf_old_sigblock;}' unless defined(&SYS_osf_old_sigblock);
}
if(defined(&__NR_osf_old_sigreturn)) {
    eval 'sub SYS_osf_old_sigreturn () { &__NR_osf_old_sigreturn;}' unless defined(&SYS_osf_old_sigreturn);
}
if(defined(&__NR_osf_old_sigsetmask)) {
    eval 'sub SYS_osf_old_sigsetmask () { &__NR_osf_old_sigsetmask;}' unless defined(&SYS_osf_old_sigsetmask);
}
if(defined(&__NR_osf_old_sigvec)) {
    eval 'sub SYS_osf_old_sigvec () { &__NR_osf_old_sigvec;}' unless defined(&SYS_osf_old_sigvec);
}
if(defined(&__NR_osf_old_stat)) {
    eval 'sub SYS_osf_old_stat () { &__NR_osf_old_stat;}' unless defined(&SYS_osf_old_stat);
}
if(defined(&__NR_osf_old_vadvise)) {
    eval 'sub SYS_osf_old_vadvise () { &__NR_osf_old_vadvise;}' unless defined(&SYS_osf_old_vadvise);
}
if(defined(&__NR_osf_old_vtrace)) {
    eval 'sub SYS_osf_old_vtrace () { &__NR_osf_old_vtrace;}' unless defined(&SYS_osf_old_vtrace);
}
if(defined(&__NR_osf_old_wait)) {
    eval 'sub SYS_osf_old_wait () { &__NR_osf_old_wait;}' unless defined(&SYS_osf_old_wait);
}
if(defined(&__NR_osf_oldquota)) {
    eval 'sub SYS_osf_oldquota () { &__NR_osf_oldquota;}' unless defined(&SYS_osf_oldquota);
}
if(defined(&__NR_osf_pathconf)) {
    eval 'sub SYS_osf_pathconf () { &__NR_osf_pathconf;}' unless defined(&SYS_osf_pathconf);
}
if(defined(&__NR_osf_pid_block)) {
    eval 'sub SYS_osf_pid_block () { &__NR_osf_pid_block;}' unless defined(&SYS_osf_pid_block);
}
if(defined(&__NR_osf_pid_unblock)) {
    eval 'sub SYS_osf_pid_unblock () { &__NR_osf_pid_unblock;}' unless defined(&SYS_osf_pid_unblock);
}
if(defined(&__NR_osf_plock)) {
    eval 'sub SYS_osf_plock () { &__NR_osf_plock;}' unless defined(&SYS_osf_plock);
}
if(defined(&__NR_osf_priocntlset)) {
    eval 'sub SYS_osf_priocntlset () { &__NR_osf_priocntlset;}' unless defined(&SYS_osf_priocntlset);
}
if(defined(&__NR_osf_profil)) {
    eval 'sub SYS_osf_profil () { &__NR_osf_profil;}' unless defined(&SYS_osf_profil);
}
if(defined(&__NR_osf_proplist_syscall)) {
    eval 'sub SYS_osf_proplist_syscall () { &__NR_osf_proplist_syscall;}' unless defined(&SYS_osf_proplist_syscall);
}
if(defined(&__NR_osf_reboot)) {
    eval 'sub SYS_osf_reboot () { &__NR_osf_reboot;}' unless defined(&SYS_osf_reboot);
}
if(defined(&__NR_osf_revoke)) {
    eval 'sub SYS_osf_revoke () { &__NR_osf_revoke;}' unless defined(&SYS_osf_revoke);
}
if(defined(&__NR_osf_sbrk)) {
    eval 'sub SYS_osf_sbrk () { &__NR_osf_sbrk;}' unless defined(&SYS_osf_sbrk);
}
if(defined(&__NR_osf_security)) {
    eval 'sub SYS_osf_security () { &__NR_osf_security;}' unless defined(&SYS_osf_security);
}
if(defined(&__NR_osf_select)) {
    eval 'sub SYS_osf_select () { &__NR_osf_select;}' unless defined(&SYS_osf_select);
}
if(defined(&__NR_osf_set_program_attributes)) {
    eval 'sub SYS_osf_set_program_attributes () { &__NR_osf_set_program_attributes;}' unless defined(&SYS_osf_set_program_attributes);
}
if(defined(&__NR_osf_set_speculative)) {
    eval 'sub SYS_osf_set_speculative () { &__NR_osf_set_speculative;}' unless defined(&SYS_osf_set_speculative);
}
if(defined(&__NR_osf_sethostid)) {
    eval 'sub SYS_osf_sethostid () { &__NR_osf_sethostid;}' unless defined(&SYS_osf_sethostid);
}
if(defined(&__NR_osf_setitimer)) {
    eval 'sub SYS_osf_setitimer () { &__NR_osf_setitimer;}' unless defined(&SYS_osf_setitimer);
}
if(defined(&__NR_osf_setlogin)) {
    eval 'sub SYS_osf_setlogin () { &__NR_osf_setlogin;}' unless defined(&SYS_osf_setlogin);
}
if(defined(&__NR_osf_setsysinfo)) {
    eval 'sub SYS_osf_setsysinfo () { &__NR_osf_setsysinfo;}' unless defined(&SYS_osf_setsysinfo);
}
if(defined(&__NR_osf_settimeofday)) {
    eval 'sub SYS_osf_settimeofday () { &__NR_osf_settimeofday;}' unless defined(&SYS_osf_settimeofday);
}
if(defined(&__NR_osf_shmat)) {
    eval 'sub SYS_osf_shmat () { &__NR_osf_shmat;}' unless defined(&SYS_osf_shmat);
}
if(defined(&__NR_osf_signal)) {
    eval 'sub SYS_osf_signal () { &__NR_osf_signal;}' unless defined(&SYS_osf_signal);
}
if(defined(&__NR_osf_sigprocmask)) {
    eval 'sub SYS_osf_sigprocmask () { &__NR_osf_sigprocmask;}' unless defined(&SYS_osf_sigprocmask);
}
if(defined(&__NR_osf_sigsendset)) {
    eval 'sub SYS_osf_sigsendset () { &__NR_osf_sigsendset;}' unless defined(&SYS_osf_sigsendset);
}
if(defined(&__NR_osf_sigstack)) {
    eval 'sub SYS_osf_sigstack () { &__NR_osf_sigstack;}' unless defined(&SYS_osf_sigstack);
}
if(defined(&__NR_osf_sigwaitprim)) {
    eval 'sub SYS_osf_sigwaitprim () { &__NR_osf_sigwaitprim;}' unless defined(&SYS_osf_sigwaitprim);
}
if(defined(&__NR_osf_sstk)) {
    eval 'sub SYS_osf_sstk () { &__NR_osf_sstk;}' unless defined(&SYS_osf_sstk);
}
if(defined(&__NR_osf_stat)) {
    eval 'sub SYS_osf_stat () { &__NR_osf_stat;}' unless defined(&SYS_osf_stat);
}
if(defined(&__NR_osf_statfs)) {
    eval 'sub SYS_osf_statfs () { &__NR_osf_statfs;}' unless defined(&SYS_osf_statfs);
}
if(defined(&__NR_osf_statfs64)) {
    eval 'sub SYS_osf_statfs64 () { &__NR_osf_statfs64;}' unless defined(&SYS_osf_statfs64);
}
if(defined(&__NR_osf_subsys_info)) {
    eval 'sub SYS_osf_subsys_info () { &__NR_osf_subsys_info;}' unless defined(&SYS_osf_subsys_info);
}
if(defined(&__NR_osf_swapctl)) {
    eval 'sub SYS_osf_swapctl () { &__NR_osf_swapctl;}' unless defined(&SYS_osf_swapctl);
}
if(defined(&__NR_osf_swapon)) {
    eval 'sub SYS_osf_swapon () { &__NR_osf_swapon;}' unless defined(&SYS_osf_swapon);
}
if(defined(&__NR_osf_syscall)) {
    eval 'sub SYS_osf_syscall () { &__NR_osf_syscall;}' unless defined(&SYS_osf_syscall);
}
if(defined(&__NR_osf_sysinfo)) {
    eval 'sub SYS_osf_sysinfo () { &__NR_osf_sysinfo;}' unless defined(&SYS_osf_sysinfo);
}
if(defined(&__NR_osf_table)) {
    eval 'sub SYS_osf_table () { &__NR_osf_table;}' unless defined(&SYS_osf_table);
}
if(defined(&__NR_osf_uadmin)) {
    eval 'sub SYS_osf_uadmin () { &__NR_osf_uadmin;}' unless defined(&SYS_osf_uadmin);
}
if(defined(&__NR_osf_usleep_thread)) {
    eval 'sub SYS_osf_usleep_thread () { &__NR_osf_usleep_thread;}' unless defined(&SYS_osf_usleep_thread);
}
if(defined(&__NR_osf_uswitch)) {
    eval 'sub SYS_osf_uswitch () { &__NR_osf_uswitch;}' unless defined(&SYS_osf_uswitch);
}
if(defined(&__NR_osf_utc_adjtime)) {
    eval 'sub SYS_osf_utc_adjtime () { &__NR_osf_utc_adjtime;}' unless defined(&SYS_osf_utc_adjtime);
}
if(defined(&__NR_osf_utc_gettime)) {
    eval 'sub SYS_osf_utc_gettime () { &__NR_osf_utc_gettime;}' unless defined(&SYS_osf_utc_gettime);
}
if(defined(&__NR_osf_utimes)) {
    eval 'sub SYS_osf_utimes () { &__NR_osf_utimes;}' unless defined(&SYS_osf_utimes);
}
if(defined(&__NR_osf_utsname)) {
    eval 'sub SYS_osf_utsname () { &__NR_osf_utsname;}' unless defined(&SYS_osf_utsname);
}
if(defined(&__NR_osf_wait4)) {
    eval 'sub SYS_osf_wait4 () { &__NR_osf_wait4;}' unless defined(&SYS_osf_wait4);
}
if(defined(&__NR_osf_waitid)) {
    eval 'sub SYS_osf_waitid () { &__NR_osf_waitid;}' unless defined(&SYS_osf_waitid);
}
if(defined(&__NR_pause)) {
    eval 'sub SYS_pause () { &__NR_pause;}' unless defined(&SYS_pause);
}
if(defined(&__NR_pciconfig_iobase)) {
    eval 'sub SYS_pciconfig_iobase () { &__NR_pciconfig_iobase;}' unless defined(&SYS_pciconfig_iobase);
}
if(defined(&__NR_pciconfig_read)) {
    eval 'sub SYS_pciconfig_read () { &__NR_pciconfig_read;}' unless defined(&SYS_pciconfig_read);
}
if(defined(&__NR_pciconfig_write)) {
    eval 'sub SYS_pciconfig_write () { &__NR_pciconfig_write;}' unless defined(&SYS_pciconfig_write);
}
if(defined(&__NR_perf_event_open)) {
    eval 'sub SYS_perf_event_open () { &__NR_perf_event_open;}' unless defined(&SYS_perf_event_open);
}
if(defined(&__NR_perfctr)) {
    eval 'sub SYS_perfctr () { &__NR_perfctr;}' unless defined(&SYS_perfctr);
}
if(defined(&__NR_perfmonctl)) {
    eval 'sub SYS_perfmonctl () { &__NR_perfmonctl;}' unless defined(&SYS_perfmonctl);
}
if(defined(&__NR_personality)) {
    eval 'sub SYS_personality () { &__NR_personality;}' unless defined(&SYS_personality);
}
if(defined(&__NR_pidfd_getfd)) {
    eval 'sub SYS_pidfd_getfd () { &__NR_pidfd_getfd;}' unless defined(&SYS_pidfd_getfd);
}
if(defined(&__NR_pidfd_open)) {
    eval 'sub SYS_pidfd_open () { &__NR_pidfd_open;}' unless defined(&SYS_pidfd_open);
}
if(defined(&__NR_pidfd_send_signal)) {
    eval 'sub SYS_pidfd_send_signal () { &__NR_pidfd_send_signal;}' unless defined(&SYS_pidfd_send_signal);
}
if(defined(&__NR_pipe)) {
    eval 'sub SYS_pipe () { &__NR_pipe;}' unless defined(&SYS_pipe);
}
if(defined(&__NR_pipe2)) {
    eval 'sub SYS_pipe2 () { &__NR_pipe2;}' unless defined(&SYS_pipe2);
}
if(defined(&__NR_pivot_root)) {
    eval 'sub SYS_pivot_root () { &__NR_pivot_root;}' unless defined(&SYS_pivot_root);
}
if(defined(&__NR_pkey_alloc)) {
    eval 'sub SYS_pkey_alloc () { &__NR_pkey_alloc;}' unless defined(&SYS_pkey_alloc);
}
if(defined(&__NR_pkey_free)) {
    eval 'sub SYS_pkey_free () { &__NR_pkey_free;}' unless defined(&SYS_pkey_free);
}
if(defined(&__NR_pkey_mprotect)) {
    eval 'sub SYS_pkey_mprotect () { &__NR_pkey_mprotect;}' unless defined(&SYS_pkey_mprotect);
}
if(defined(&__NR_poll)) {
    eval 'sub SYS_poll () { &__NR_poll;}' unless defined(&SYS_poll);
}
if(defined(&__NR_ppoll)) {
    eval 'sub SYS_ppoll () { &__NR_ppoll;}' unless defined(&SYS_ppoll);
}
if(defined(&__NR_ppoll_time64)) {
    eval 'sub SYS_ppoll_time64 () { &__NR_ppoll_time64;}' unless defined(&SYS_ppoll_time64);
}
if(defined(&__NR_prctl)) {
    eval 'sub SYS_prctl () { &__NR_prctl;}' unless defined(&SYS_prctl);
}
if(defined(&__NR_pread64)) {
    eval 'sub SYS_pread64 () { &__NR_pread64;}' unless defined(&SYS_pread64);
}
if(defined(&__NR_preadv)) {
    eval 'sub SYS_preadv () { &__NR_preadv;}' unless defined(&SYS_preadv);
}
if(defined(&__NR_preadv2)) {
    eval 'sub SYS_preadv2 () { &__NR_preadv2;}' unless defined(&SYS_preadv2);
}
if(defined(&__NR_prlimit64)) {
    eval 'sub SYS_prlimit64 () { &__NR_prlimit64;}' unless defined(&SYS_prlimit64);
}
if(defined(&__NR_process_madvise)) {
    eval 'sub SYS_process_madvise () { &__NR_process_madvise;}' unless defined(&SYS_process_madvise);
}
if(defined(&__NR_process_mrelease)) {
    eval 'sub SYS_process_mrelease () { &__NR_process_mrelease;}' unless defined(&SYS_process_mrelease);
}
if(defined(&__NR_process_vm_readv)) {
    eval 'sub SYS_process_vm_readv () { &__NR_process_vm_readv;}' unless defined(&SYS_process_vm_readv);
}
if(defined(&__NR_process_vm_writev)) {
    eval 'sub SYS_process_vm_writev () { &__NR_process_vm_writev;}' unless defined(&SYS_process_vm_writev);
}
if(defined(&__NR_prof)) {
    eval 'sub SYS_prof () { &__NR_prof;}' unless defined(&SYS_prof);
}
if(defined(&__NR_profil)) {
    eval 'sub SYS_profil () { &__NR_profil;}' unless defined(&SYS_profil);
}
if(defined(&__NR_pselect6)) {
    eval 'sub SYS_pselect6 () { &__NR_pselect6;}' unless defined(&SYS_pselect6);
}
if(defined(&__NR_pselect6_time64)) {
    eval 'sub SYS_pselect6_time64 () { &__NR_pselect6_time64;}' unless defined(&SYS_pselect6_time64);
}
if(defined(&__NR_ptrace)) {
    eval 'sub SYS_ptrace () { &__NR_ptrace;}' unless defined(&SYS_ptrace);
}
if(defined(&__NR_putpmsg)) {
    eval 'sub SYS_putpmsg () { &__NR_putpmsg;}' unless defined(&SYS_putpmsg);
}
if(defined(&__NR_pwrite64)) {
    eval 'sub SYS_pwrite64 () { &__NR_pwrite64;}' unless defined(&SYS_pwrite64);
}
if(defined(&__NR_pwritev)) {
    eval 'sub SYS_pwritev () { &__NR_pwritev;}' unless defined(&SYS_pwritev);
}
if(defined(&__NR_pwritev2)) {
    eval 'sub SYS_pwritev2 () { &__NR_pwritev2;}' unless defined(&SYS_pwritev2);
}
if(defined(&__NR_query_module)) {
    eval 'sub SYS_query_module () { &__NR_query_module;}' unless defined(&SYS_query_module);
}
if(defined(&__NR_quotactl)) {
    eval 'sub SYS_quotactl () { &__NR_quotactl;}' unless defined(&SYS_quotactl);
}
if(defined(&__NR_quotactl_fd)) {
    eval 'sub SYS_quotactl_fd () { &__NR_quotactl_fd;}' unless defined(&SYS_quotactl_fd);
}
if(defined(&__NR_read)) {
    eval 'sub SYS_read () { &__NR_read;}' unless defined(&SYS_read);
}
if(defined(&__NR_readahead)) {
    eval 'sub SYS_readahead () { &__NR_readahead;}' unless defined(&SYS_readahead);
}
if(defined(&__NR_readdir)) {
    eval 'sub SYS_readdir () { &__NR_readdir;}' unless defined(&SYS_readdir);
}
if(defined(&__NR_readlink)) {
    eval 'sub SYS_readlink () { &__NR_readlink;}' unless defined(&SYS_readlink);
}
if(defined(&__NR_readlinkat)) {
    eval 'sub SYS_readlinkat () { &__NR_readlinkat;}' unless defined(&SYS_readlinkat);
}
if(defined(&__NR_readv)) {
    eval 'sub SYS_readv () { &__NR_readv;}' unless defined(&SYS_readv);
}
if(defined(&__NR_reboot)) {
    eval 'sub SYS_reboot () { &__NR_reboot;}' unless defined(&SYS_reboot);
}
if(defined(&__NR_recv)) {
    eval 'sub SYS_recv () { &__NR_recv;}' unless defined(&SYS_recv);
}
if(defined(&__NR_recvfrom)) {
    eval 'sub SYS_recvfrom () { &__NR_recvfrom;}' unless defined(&SYS_recvfrom);
}
if(defined(&__NR_recvmmsg)) {
    eval 'sub SYS_recvmmsg () { &__NR_recvmmsg;}' unless defined(&SYS_recvmmsg);
}
if(defined(&__NR_recvmmsg_time64)) {
    eval 'sub SYS_recvmmsg_time64 () { &__NR_recvmmsg_time64;}' unless defined(&SYS_recvmmsg_time64);
}
if(defined(&__NR_recvmsg)) {
    eval 'sub SYS_recvmsg () { &__NR_recvmsg;}' unless defined(&SYS_recvmsg);
}
if(defined(&__NR_remap_file_pages)) {
    eval 'sub SYS_remap_file_pages () { &__NR_remap_file_pages;}' unless defined(&SYS_remap_file_pages);
}
if(defined(&__NR_removexattr)) {
    eval 'sub SYS_removexattr () { &__NR_removexattr;}' unless defined(&SYS_removexattr);
}
if(defined(&__NR_rename)) {
    eval 'sub SYS_rename () { &__NR_rename;}' unless defined(&SYS_rename);
}
if(defined(&__NR_renameat)) {
    eval 'sub SYS_renameat () { &__NR_renameat;}' unless defined(&SYS_renameat);
}
if(defined(&__NR_renameat2)) {
    eval 'sub SYS_renameat2 () { &__NR_renameat2;}' unless defined(&SYS_renameat2);
}
if(defined(&__NR_request_key)) {
    eval 'sub SYS_request_key () { &__NR_request_key;}' unless defined(&SYS_request_key);
}
if(defined(&__NR_restart_syscall)) {
    eval 'sub SYS_restart_syscall () { &__NR_restart_syscall;}' unless defined(&SYS_restart_syscall);
}
if(defined(&__NR_riscv_flush_icache)) {
    eval 'sub SYS_riscv_flush_icache () { &__NR_riscv_flush_icache;}' unless defined(&SYS_riscv_flush_icache);
}
if(defined(&__NR_rmdir)) {
    eval 'sub SYS_rmdir () { &__NR_rmdir;}' unless defined(&SYS_rmdir);
}
if(defined(&__NR_rseq)) {
    eval 'sub SYS_rseq () { &__NR_rseq;}' unless defined(&SYS_rseq);
}
if(defined(&__NR_rt_sigaction)) {
    eval 'sub SYS_rt_sigaction () { &__NR_rt_sigaction;}' unless defined(&SYS_rt_sigaction);
}
if(defined(&__NR_rt_sigpending)) {
    eval 'sub SYS_rt_sigpending () { &__NR_rt_sigpending;}' unless defined(&SYS_rt_sigpending);
}
if(defined(&__NR_rt_sigprocmask)) {
    eval 'sub SYS_rt_sigprocmask () { &__NR_rt_sigprocmask;}' unless defined(&SYS_rt_sigprocmask);
}
if(defined(&__NR_rt_sigqueueinfo)) {
    eval 'sub SYS_rt_sigqueueinfo () { &__NR_rt_sigqueueinfo;}' unless defined(&SYS_rt_sigqueueinfo);
}
if(defined(&__NR_rt_sigreturn)) {
    eval 'sub SYS_rt_sigreturn () { &__NR_rt_sigreturn;}' unless defined(&SYS_rt_sigreturn);
}
if(defined(&__NR_rt_sigsuspend)) {
    eval 'sub SYS_rt_sigsuspend () { &__NR_rt_sigsuspend;}' unless defined(&SYS_rt_sigsuspend);
}
if(defined(&__NR_rt_sigtimedwait)) {
    eval 'sub SYS_rt_sigtimedwait () { &__NR_rt_sigtimedwait;}' unless defined(&SYS_rt_sigtimedwait);
}
if(defined(&__NR_rt_sigtimedwait_time64)) {
    eval 'sub SYS_rt_sigtimedwait_time64 () { &__NR_rt_sigtimedwait_time64;}' unless defined(&SYS_rt_sigtimedwait_time64);
}
if(defined(&__NR_rt_tgsigqueueinfo)) {
    eval 'sub SYS_rt_tgsigqueueinfo () { &__NR_rt_tgsigqueueinfo;}' unless defined(&SYS_rt_tgsigqueueinfo);
}
if(defined(&__NR_rtas)) {
    eval 'sub SYS_rtas () { &__NR_rtas;}' unless defined(&SYS_rtas);
}
if(defined(&__NR_s390_guarded_storage)) {
    eval 'sub SYS_s390_guarded_storage () { &__NR_s390_guarded_storage;}' unless defined(&SYS_s390_guarded_storage);
}
if(defined(&__NR_s390_pci_mmio_read)) {
    eval 'sub SYS_s390_pci_mmio_read () { &__NR_s390_pci_mmio_read;}' unless defined(&SYS_s390_pci_mmio_read);
}
if(defined(&__NR_s390_pci_mmio_write)) {
    eval 'sub SYS_s390_pci_mmio_write () { &__NR_s390_pci_mmio_write;}' unless defined(&SYS_s390_pci_mmio_write);
}
if(defined(&__NR_s390_runtime_instr)) {
    eval 'sub SYS_s390_runtime_instr () { &__NR_s390_runtime_instr;}' unless defined(&SYS_s390_runtime_instr);
}
if(defined(&__NR_s390_sthyi)) {
    eval 'sub SYS_s390_sthyi () { &__NR_s390_sthyi;}' unless defined(&SYS_s390_sthyi);
}
if(defined(&__NR_sched_get_affinity)) {
    eval 'sub SYS_sched_get_affinity () { &__NR_sched_get_affinity;}' unless defined(&SYS_sched_get_affinity);
}
if(defined(&__NR_sched_get_priority_max)) {
    eval 'sub SYS_sched_get_priority_max () { &__NR_sched_get_priority_max;}' unless defined(&SYS_sched_get_priority_max);
}
if(defined(&__NR_sched_get_priority_min)) {
    eval 'sub SYS_sched_get_priority_min () { &__NR_sched_get_priority_min;}' unless defined(&SYS_sched_get_priority_min);
}
if(defined(&__NR_sched_getaffinity)) {
    eval 'sub SYS_sched_getaffinity () { &__NR_sched_getaffinity;}' unless defined(&SYS_sched_getaffinity);
}
if(defined(&__NR_sched_getattr)) {
    eval 'sub SYS_sched_getattr () { &__NR_sched_getattr;}' unless defined(&SYS_sched_getattr);
}
if(defined(&__NR_sched_getparam)) {
    eval 'sub SYS_sched_getparam () { &__NR_sched_getparam;}' unless defined(&SYS_sched_getparam);
}
if(defined(&__NR_sched_getscheduler)) {
    eval 'sub SYS_sched_getscheduler () { &__NR_sched_getscheduler;}' unless defined(&SYS_sched_getscheduler);
}
if(defined(&__NR_sched_rr_get_interval)) {
    eval 'sub SYS_sched_rr_get_interval () { &__NR_sched_rr_get_interval;}' unless defined(&SYS_sched_rr_get_interval);
}
if(defined(&__NR_sched_rr_get_interval_time64)) {
    eval 'sub SYS_sched_rr_get_interval_time64 () { &__NR_sched_rr_get_interval_time64;}' unless defined(&SYS_sched_rr_get_interval_time64);
}
if(defined(&__NR_sched_set_affinity)) {
    eval 'sub SYS_sched_set_affinity () { &__NR_sched_set_affinity;}' unless defined(&SYS_sched_set_affinity);
}
if(defined(&__NR_sched_setaffinity)) {
    eval 'sub SYS_sched_setaffinity () { &__NR_sched_setaffinity;}' unless defined(&SYS_sched_setaffinity);
}
if(defined(&__NR_sched_setattr)) {
    eval 'sub SYS_sched_setattr () { &__NR_sched_setattr;}' unless defined(&SYS_sched_setattr);
}
if(defined(&__NR_sched_setparam)) {
    eval 'sub SYS_sched_setparam () { &__NR_sched_setparam;}' unless defined(&SYS_sched_setparam);
}
if(defined(&__NR_sched_setscheduler)) {
    eval 'sub SYS_sched_setscheduler () { &__NR_sched_setscheduler;}' unless defined(&SYS_sched_setscheduler);
}
if(defined(&__NR_sched_yield)) {
    eval 'sub SYS_sched_yield () { &__NR_sched_yield;}' unless defined(&SYS_sched_yield);
}
if(defined(&__NR_seccomp)) {
    eval 'sub SYS_seccomp () { &__NR_seccomp;}' unless defined(&SYS_seccomp);
}
if(defined(&__NR_security)) {
    eval 'sub SYS_security () { &__NR_security;}' unless defined(&SYS_security);
}
if(defined(&__NR_select)) {
    eval 'sub SYS_select () { &__NR_select;}' unless defined(&SYS_select);
}
if(defined(&__NR_semctl)) {
    eval 'sub SYS_semctl () { &__NR_semctl;}' unless defined(&SYS_semctl);
}
if(defined(&__NR_semget)) {
    eval 'sub SYS_semget () { &__NR_semget;}' unless defined(&SYS_semget);
}
if(defined(&__NR_semop)) {
    eval 'sub SYS_semop () { &__NR_semop;}' unless defined(&SYS_semop);
}
if(defined(&__NR_semtimedop)) {
    eval 'sub SYS_semtimedop () { &__NR_semtimedop;}' unless defined(&SYS_semtimedop);
}
if(defined(&__NR_semtimedop_time64)) {
    eval 'sub SYS_semtimedop_time64 () { &__NR_semtimedop_time64;}' unless defined(&SYS_semtimedop_time64);
}
if(defined(&__NR_send)) {
    eval 'sub SYS_send () { &__NR_send;}' unless defined(&SYS_send);
}
if(defined(&__NR_sendfile)) {
    eval 'sub SYS_sendfile () { &__NR_sendfile;}' unless defined(&SYS_sendfile);
}
if(defined(&__NR_sendfile64)) {
    eval 'sub SYS_sendfile64 () { &__NR_sendfile64;}' unless defined(&SYS_sendfile64);
}
if(defined(&__NR_sendmmsg)) {
    eval 'sub SYS_sendmmsg () { &__NR_sendmmsg;}' unless defined(&SYS_sendmmsg);
}
if(defined(&__NR_sendmsg)) {
    eval 'sub SYS_sendmsg () { &__NR_sendmsg;}' unless defined(&SYS_sendmsg);
}
if(defined(&__NR_sendto)) {
    eval 'sub SYS_sendto () { &__NR_sendto;}' unless defined(&SYS_sendto);
}
if(defined(&__NR_set_mempolicy)) {
    eval 'sub SYS_set_mempolicy () { &__NR_set_mempolicy;}' unless defined(&SYS_set_mempolicy);
}
if(defined(&__NR_set_robust_list)) {
    eval 'sub SYS_set_robust_list () { &__NR_set_robust_list;}' unless defined(&SYS_set_robust_list);
}
if(defined(&__NR_set_thread_area)) {
    eval 'sub SYS_set_thread_area () { &__NR_set_thread_area;}' unless defined(&SYS_set_thread_area);
}
if(defined(&__NR_set_tid_address)) {
    eval 'sub SYS_set_tid_address () { &__NR_set_tid_address;}' unless defined(&SYS_set_tid_address);
}
if(defined(&__NR_set_tls)) {
    eval 'sub SYS_set_tls () { &__NR_set_tls;}' unless defined(&SYS_set_tls);
}
if(defined(&__NR_setdomainname)) {
    eval 'sub SYS_setdomainname () { &__NR_setdomainname;}' unless defined(&SYS_setdomainname);
}
if(defined(&__NR_setfsgid)) {
    eval 'sub SYS_setfsgid () { &__NR_setfsgid;}' unless defined(&SYS_setfsgid);
}
if(defined(&__NR_setfsgid32)) {
    eval 'sub SYS_setfsgid32 () { &__NR_setfsgid32;}' unless defined(&SYS_setfsgid32);
}
if(defined(&__NR_setfsuid)) {
    eval 'sub SYS_setfsuid () { &__NR_setfsuid;}' unless defined(&SYS_setfsuid);
}
if(defined(&__NR_setfsuid32)) {
    eval 'sub SYS_setfsuid32 () { &__NR_setfsuid32;}' unless defined(&SYS_setfsuid32);
}
if(defined(&__NR_setgid)) {
    eval 'sub SYS_setgid () { &__NR_setgid;}' unless defined(&SYS_setgid);
}
if(defined(&__NR_setgid32)) {
    eval 'sub SYS_setgid32 () { &__NR_setgid32;}' unless defined(&SYS_setgid32);
}
if(defined(&__NR_setgroups)) {
    eval 'sub SYS_setgroups () { &__NR_setgroups;}' unless defined(&SYS_setgroups);
}
if(defined(&__NR_setgroups32)) {
    eval 'sub SYS_setgroups32 () { &__NR_setgroups32;}' unless defined(&SYS_setgroups32);
}
if(defined(&__NR_sethae)) {
    eval 'sub SYS_sethae () { &__NR_sethae;}' unless defined(&SYS_sethae);
}
if(defined(&__NR_sethostname)) {
    eval 'sub SYS_sethostname () { &__NR_sethostname;}' unless defined(&SYS_sethostname);
}
if(defined(&__NR_setitimer)) {
    eval 'sub SYS_setitimer () { &__NR_setitimer;}' unless defined(&SYS_setitimer);
}
if(defined(&__NR_setns)) {
    eval 'sub SYS_setns () { &__NR_setns;}' unless defined(&SYS_setns);
}
if(defined(&__NR_setpgid)) {
    eval 'sub SYS_setpgid () { &__NR_setpgid;}' unless defined(&SYS_setpgid);
}
if(defined(&__NR_setpgrp)) {
    eval 'sub SYS_setpgrp () { &__NR_setpgrp;}' unless defined(&SYS_setpgrp);
}
if(defined(&__NR_setpriority)) {
    eval 'sub SYS_setpriority () { &__NR_setpriority;}' unless defined(&SYS_setpriority);
}
if(defined(&__NR_setregid)) {
    eval 'sub SYS_setregid () { &__NR_setregid;}' unless defined(&SYS_setregid);
}
if(defined(&__NR_setregid32)) {
    eval 'sub SYS_setregid32 () { &__NR_setregid32;}' unless defined(&SYS_setregid32);
}
if(defined(&__NR_setresgid)) {
    eval 'sub SYS_setresgid () { &__NR_setresgid;}' unless defined(&SYS_setresgid);
}
if(defined(&__NR_setresgid32)) {
    eval 'sub SYS_setresgid32 () { &__NR_setresgid32;}' unless defined(&SYS_setresgid32);
}
if(defined(&__NR_setresuid)) {
    eval 'sub SYS_setresuid () { &__NR_setresuid;}' unless defined(&SYS_setresuid);
}
if(defined(&__NR_setresuid32)) {
    eval 'sub SYS_setresuid32 () { &__NR_setresuid32;}' unless defined(&SYS_setresuid32);
}
if(defined(&__NR_setreuid)) {
    eval 'sub SYS_setreuid () { &__NR_setreuid;}' unless defined(&SYS_setreuid);
}
if(defined(&__NR_setreuid32)) {
    eval 'sub SYS_setreuid32 () { &__NR_setreuid32;}' unless defined(&SYS_setreuid32);
}
if(defined(&__NR_setrlimit)) {
    eval 'sub SYS_setrlimit () { &__NR_setrlimit;}' unless defined(&SYS_setrlimit);
}
if(defined(&__NR_setsid)) {
    eval 'sub SYS_setsid () { &__NR_setsid;}' unless defined(&SYS_setsid);
}
if(defined(&__NR_setsockopt)) {
    eval 'sub SYS_setsockopt () { &__NR_setsockopt;}' unless defined(&SYS_setsockopt);
}
if(defined(&__NR_settimeofday)) {
    eval 'sub SYS_settimeofday () { &__NR_settimeofday;}' unless defined(&SYS_settimeofday);
}
if(defined(&__NR_setuid)) {
    eval 'sub SYS_setuid () { &__NR_setuid;}' unless defined(&SYS_setuid);
}
if(defined(&__NR_setuid32)) {
    eval 'sub SYS_setuid32 () { &__NR_setuid32;}' unless defined(&SYS_setuid32);
}
if(defined(&__NR_setxattr)) {
    eval 'sub SYS_setxattr () { &__NR_setxattr;}' unless defined(&SYS_setxattr);
}
if(defined(&__NR_sgetmask)) {
    eval 'sub SYS_sgetmask () { &__NR_sgetmask;}' unless defined(&SYS_sgetmask);
}
if(defined(&__NR_shmat)) {
    eval 'sub SYS_shmat () { &__NR_shmat;}' unless defined(&SYS_shmat);
}
if(defined(&__NR_shmctl)) {
    eval 'sub SYS_shmctl () { &__NR_shmctl;}' unless defined(&SYS_shmctl);
}
if(defined(&__NR_shmdt)) {
    eval 'sub SYS_shmdt () { &__NR_shmdt;}' unless defined(&SYS_shmdt);
}
if(defined(&__NR_shmget)) {
    eval 'sub SYS_shmget () { &__NR_shmget;}' unless defined(&SYS_shmget);
}
if(defined(&__NR_shutdown)) {
    eval 'sub SYS_shutdown () { &__NR_shutdown;}' unless defined(&SYS_shutdown);
}
if(defined(&__NR_sigaction)) {
    eval 'sub SYS_sigaction () { &__NR_sigaction;}' unless defined(&SYS_sigaction);
}
if(defined(&__NR_sigaltstack)) {
    eval 'sub SYS_sigaltstack () { &__NR_sigaltstack;}' unless defined(&SYS_sigaltstack);
}
if(defined(&__NR_signal)) {
    eval 'sub SYS_signal () { &__NR_signal;}' unless defined(&SYS_signal);
}
if(defined(&__NR_signalfd)) {
    eval 'sub SYS_signalfd () { &__NR_signalfd;}' unless defined(&SYS_signalfd);
}
if(defined(&__NR_signalfd4)) {
    eval 'sub SYS_signalfd4 () { &__NR_signalfd4;}' unless defined(&SYS_signalfd4);
}
if(defined(&__NR_sigpending)) {
    eval 'sub SYS_sigpending () { &__NR_sigpending;}' unless defined(&SYS_sigpending);
}
if(defined(&__NR_sigprocmask)) {
    eval 'sub SYS_sigprocmask () { &__NR_sigprocmask;}' unless defined(&SYS_sigprocmask);
}
if(defined(&__NR_sigreturn)) {
    eval 'sub SYS_sigreturn () { &__NR_sigreturn;}' unless defined(&SYS_sigreturn);
}
if(defined(&__NR_sigsuspend)) {
    eval 'sub SYS_sigsuspend () { &__NR_sigsuspend;}' unless defined(&SYS_sigsuspend);
}
if(defined(&__NR_socket)) {
    eval 'sub SYS_socket () { &__NR_socket;}' unless defined(&SYS_socket);
}
if(defined(&__NR_socketcall)) {
    eval 'sub SYS_socketcall () { &__NR_socketcall;}' unless defined(&SYS_socketcall);
}
if(defined(&__NR_socketpair)) {
    eval 'sub SYS_socketpair () { &__NR_socketpair;}' unless defined(&SYS_socketpair);
}
if(defined(&__NR_splice)) {
    eval 'sub SYS_splice () { &__NR_splice;}' unless defined(&SYS_splice);
}
if(defined(&__NR_spu_create)) {
    eval 'sub SYS_spu_create () { &__NR_spu_create;}' unless defined(&SYS_spu_create);
}
if(defined(&__NR_spu_run)) {
    eval 'sub SYS_spu_run () { &__NR_spu_run;}' unless defined(&SYS_spu_run);
}
if(defined(&__NR_ssetmask)) {
    eval 'sub SYS_ssetmask () { &__NR_ssetmask;}' unless defined(&SYS_ssetmask);
}
if(defined(&__NR_stat)) {
    eval 'sub SYS_stat () { &__NR_stat;}' unless defined(&SYS_stat);
}
if(defined(&__NR_stat64)) {
    eval 'sub SYS_stat64 () { &__NR_stat64;}' unless defined(&SYS_stat64);
}
if(defined(&__NR_statfs)) {
    eval 'sub SYS_statfs () { &__NR_statfs;}' unless defined(&SYS_statfs);
}
if(defined(&__NR_statfs64)) {
    eval 'sub SYS_statfs64 () { &__NR_statfs64;}' unless defined(&SYS_statfs64);
}
if(defined(&__NR_statx)) {
    eval 'sub SYS_statx () { &__NR_statx;}' unless defined(&SYS_statx);
}
if(defined(&__NR_stime)) {
    eval 'sub SYS_stime () { &__NR_stime;}' unless defined(&SYS_stime);
}
if(defined(&__NR_stty)) {
    eval 'sub SYS_stty () { &__NR_stty;}' unless defined(&SYS_stty);
}
if(defined(&__NR_subpage_prot)) {
    eval 'sub SYS_subpage_prot () { &__NR_subpage_prot;}' unless defined(&SYS_subpage_prot);
}
if(defined(&__NR_swapcontext)) {
    eval 'sub SYS_swapcontext () { &__NR_swapcontext;}' unless defined(&SYS_swapcontext);
}
if(defined(&__NR_swapoff)) {
    eval 'sub SYS_swapoff () { &__NR_swapoff;}' unless defined(&SYS_swapoff);
}
if(defined(&__NR_swapon)) {
    eval 'sub SYS_swapon () { &__NR_swapon;}' unless defined(&SYS_swapon);
}
if(defined(&__NR_switch_endian)) {
    eval 'sub SYS_switch_endian () { &__NR_switch_endian;}' unless defined(&SYS_switch_endian);
}
if(defined(&__NR_symlink)) {
    eval 'sub SYS_symlink () { &__NR_symlink;}' unless defined(&SYS_symlink);
}
if(defined(&__NR_symlinkat)) {
    eval 'sub SYS_symlinkat () { &__NR_symlinkat;}' unless defined(&SYS_symlinkat);
}
if(defined(&__NR_sync)) {
    eval 'sub SYS_sync () { &__NR_sync;}' unless defined(&SYS_sync);
}
if(defined(&__NR_sync_file_range)) {
    eval 'sub SYS_sync_file_range () { &__NR_sync_file_range;}' unless defined(&SYS_sync_file_range);
}
if(defined(&__NR_sync_file_range2)) {
    eval 'sub SYS_sync_file_range2 () { &__NR_sync_file_range2;}' unless defined(&SYS_sync_file_range2);
}
if(defined(&__NR_syncfs)) {
    eval 'sub SYS_syncfs () { &__NR_syncfs;}' unless defined(&SYS_syncfs);
}
if(defined(&__NR_sys_debug_setcontext)) {
    eval 'sub SYS_sys_debug_setcontext () { &__NR_sys_debug_setcontext;}' unless defined(&SYS_sys_debug_setcontext);
}
if(defined(&__NR_sys_epoll_create)) {
    eval 'sub SYS_sys_epoll_create () { &__NR_sys_epoll_create;}' unless defined(&SYS_sys_epoll_create);
}
if(defined(&__NR_sys_epoll_ctl)) {
    eval 'sub SYS_sys_epoll_ctl () { &__NR_sys_epoll_ctl;}' unless defined(&SYS_sys_epoll_ctl);
}
if(defined(&__NR_sys_epoll_wait)) {
    eval 'sub SYS_sys_epoll_wait () { &__NR_sys_epoll_wait;}' unless defined(&SYS_sys_epoll_wait);
}
if(defined(&__NR_syscall)) {
    eval 'sub SYS_syscall () { &__NR_syscall;}' unless defined(&SYS_syscall);
}
if(defined(&__NR_sysfs)) {
    eval 'sub SYS_sysfs () { &__NR_sysfs;}' unless defined(&SYS_sysfs);
}
if(defined(&__NR_sysinfo)) {
    eval 'sub SYS_sysinfo () { &__NR_sysinfo;}' unless defined(&SYS_sysinfo);
}
if(defined(&__NR_syslog)) {
    eval 'sub SYS_syslog () { &__NR_syslog;}' unless defined(&SYS_syslog);
}
if(defined(&__NR_sysmips)) {
    eval 'sub SYS_sysmips () { &__NR_sysmips;}' unless defined(&SYS_sysmips);
}
if(defined(&__NR_tee)) {
    eval 'sub SYS_tee () { &__NR_tee;}' unless defined(&SYS_tee);
}
if(defined(&__NR_tgkill)) {
    eval 'sub SYS_tgkill () { &__NR_tgkill;}' unless defined(&SYS_tgkill);
}
if(defined(&__NR_time)) {
    eval 'sub SYS_time () { &__NR_time;}' unless defined(&SYS_time);
}
if(defined(&__NR_timer_create)) {
    eval 'sub SYS_timer_create () { &__NR_timer_create;}' unless defined(&SYS_timer_create);
}
if(defined(&__NR_timer_delete)) {
    eval 'sub SYS_timer_delete () { &__NR_timer_delete;}' unless defined(&SYS_timer_delete);
}
if(defined(&__NR_timer_getoverrun)) {
    eval 'sub SYS_timer_getoverrun () { &__NR_timer_getoverrun;}' unless defined(&SYS_timer_getoverrun);
}
if(defined(&__NR_timer_gettime)) {
    eval 'sub SYS_timer_gettime () { &__NR_timer_gettime;}' unless defined(&SYS_timer_gettime);
}
if(defined(&__NR_timer_gettime64)) {
    eval 'sub SYS_timer_gettime64 () { &__NR_timer_gettime64;}' unless defined(&SYS_timer_gettime64);
}
if(defined(&__NR_timer_settime)) {
    eval 'sub SYS_timer_settime () { &__NR_timer_settime;}' unless defined(&SYS_timer_settime);
}
if(defined(&__NR_timer_settime64)) {
    eval 'sub SYS_timer_settime64 () { &__NR_timer_settime64;}' unless defined(&SYS_timer_settime64);
}
if(defined(&__NR_timerfd)) {
    eval 'sub SYS_timerfd () { &__NR_timerfd;}' unless defined(&SYS_timerfd);
}
if(defined(&__NR_timerfd_create)) {
    eval 'sub SYS_timerfd_create () { &__NR_timerfd_create;}' unless defined(&SYS_timerfd_create);
}
if(defined(&__NR_timerfd_gettime)) {
    eval 'sub SYS_timerfd_gettime () { &__NR_timerfd_gettime;}' unless defined(&SYS_timerfd_gettime);
}
if(defined(&__NR_timerfd_gettime64)) {
    eval 'sub SYS_timerfd_gettime64 () { &__NR_timerfd_gettime64;}' unless defined(&SYS_timerfd_gettime64);
}
if(defined(&__NR_timerfd_settime)) {
    eval 'sub SYS_timerfd_settime () { &__NR_timerfd_settime;}' unless defined(&SYS_timerfd_settime);
}
if(defined(&__NR_timerfd_settime64)) {
    eval 'sub SYS_timerfd_settime64 () { &__NR_timerfd_settime64;}' unless defined(&SYS_timerfd_settime64);
}
if(defined(&__NR_times)) {
    eval 'sub SYS_times () { &__NR_times;}' unless defined(&SYS_times);
}
if(defined(&__NR_tkill)) {
    eval 'sub SYS_tkill () { &__NR_tkill;}' unless defined(&SYS_tkill);
}
if(defined(&__NR_truncate)) {
    eval 'sub SYS_truncate () { &__NR_truncate;}' unless defined(&SYS_truncate);
}
if(defined(&__NR_truncate64)) {
    eval 'sub SYS_truncate64 () { &__NR_truncate64;}' unless defined(&SYS_truncate64);
}
if(defined(&__NR_tuxcall)) {
    eval 'sub SYS_tuxcall () { &__NR_tuxcall;}' unless defined(&SYS_tuxcall);
}
if(defined(&__NR_udftrap)) {
    eval 'sub SYS_udftrap () { &__NR_udftrap;}' unless defined(&SYS_udftrap);
}
if(defined(&__NR_ugetrlimit)) {
    eval 'sub SYS_ugetrlimit () { &__NR_ugetrlimit;}' unless defined(&SYS_ugetrlimit);
}
if(defined(&__NR_ulimit)) {
    eval 'sub SYS_ulimit () { &__NR_ulimit;}' unless defined(&SYS_ulimit);
}
if(defined(&__NR_umask)) {
    eval 'sub SYS_umask () { &__NR_umask;}' unless defined(&SYS_umask);
}
if(defined(&__NR_umount)) {
    eval 'sub SYS_umount () { &__NR_umount;}' unless defined(&SYS_umount);
}
if(defined(&__NR_umount2)) {
    eval 'sub SYS_umount2 () { &__NR_umount2;}' unless defined(&SYS_umount2);
}
if(defined(&__NR_uname)) {
    eval 'sub SYS_uname () { &__NR_uname;}' unless defined(&SYS_uname);
}
if(defined(&__NR_unlink)) {
    eval 'sub SYS_unlink () { &__NR_unlink;}' unless defined(&SYS_unlink);
}
if(defined(&__NR_unlinkat)) {
    eval 'sub SYS_unlinkat () { &__NR_unlinkat;}' unless defined(&SYS_unlinkat);
}
if(defined(&__NR_unshare)) {
    eval 'sub SYS_unshare () { &__NR_unshare;}' unless defined(&SYS_unshare);
}
if(defined(&__NR_uselib)) {
    eval 'sub SYS_uselib () { &__NR_uselib;}' unless defined(&SYS_uselib);
}
if(defined(&__NR_userfaultfd)) {
    eval 'sub SYS_userfaultfd () { &__NR_userfaultfd;}' unless defined(&SYS_userfaultfd);
}
if(defined(&__NR_usr26)) {
    eval 'sub SYS_usr26 () { &__NR_usr26;}' unless defined(&SYS_usr26);
}
if(defined(&__NR_usr32)) {
    eval 'sub SYS_usr32 () { &__NR_usr32;}' unless defined(&SYS_usr32);
}
if(defined(&__NR_ustat)) {
    eval 'sub SYS_ustat () { &__NR_ustat;}' unless defined(&SYS_ustat);
}
if(defined(&__NR_utime)) {
    eval 'sub SYS_utime () { &__NR_utime;}' unless defined(&SYS_utime);
}
if(defined(&__NR_utimensat)) {
    eval 'sub SYS_utimensat () { &__NR_utimensat;}' unless defined(&SYS_utimensat);
}
if(defined(&__NR_utimensat_time64)) {
    eval 'sub SYS_utimensat_time64 () { &__NR_utimensat_time64;}' unless defined(&SYS_utimensat_time64);
}
if(defined(&__NR_utimes)) {
    eval 'sub SYS_utimes () { &__NR_utimes;}' unless defined(&SYS_utimes);
}
if(defined(&__NR_utrap_install)) {
    eval 'sub SYS_utrap_install () { &__NR_utrap_install;}' unless defined(&SYS_utrap_install);
}
if(defined(&__NR_vfork)) {
    eval 'sub SYS_vfork () { &__NR_vfork;}' unless defined(&SYS_vfork);
}
if(defined(&__NR_vhangup)) {
    eval 'sub SYS_vhangup () { &__NR_vhangup;}' unless defined(&SYS_vhangup);
}
if(defined(&__NR_vm86)) {
    eval 'sub SYS_vm86 () { &__NR_vm86;}' unless defined(&SYS_vm86);
}
if(defined(&__NR_vm86old)) {
    eval 'sub SYS_vm86old () { &__NR_vm86old;}' unless defined(&SYS_vm86old);
}
if(defined(&__NR_vmsplice)) {
    eval 'sub SYS_vmsplice () { &__NR_vmsplice;}' unless defined(&SYS_vmsplice);
}
if(defined(&__NR_vserver)) {
    eval 'sub SYS_vserver () { &__NR_vserver;}' unless defined(&SYS_vserver);
}
if(defined(&__NR_wait4)) {
    eval 'sub SYS_wait4 () { &__NR_wait4;}' unless defined(&SYS_wait4);
}
if(defined(&__NR_waitid)) {
    eval 'sub SYS_waitid () { &__NR_waitid;}' unless defined(&SYS_waitid);
}
if(defined(&__NR_waitpid)) {
    eval 'sub SYS_waitpid () { &__NR_waitpid;}' unless defined(&SYS_waitpid);
}
if(defined(&__NR_write)) {
    eval 'sub SYS_write () { &__NR_write;}' unless defined(&SYS_write);
}
if(defined(&__NR_writev)) {
    eval 'sub SYS_writev () { &__NR_writev;}' unless defined(&SYS_writev);
}
1;
