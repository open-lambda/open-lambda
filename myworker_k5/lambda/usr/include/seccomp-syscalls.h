/**
 * Seccomp Library
 *
 * Copyright (c) 2019 Cisco Systems <pmoore2@cisco.com>
 * Author: Paul Moore <paul@paul-moore.com>
 */

/*
 * This library is free software; you can redistribute it and/or modify it
 * under the terms of version 2.1 of the GNU Lesser General Public License as
 * published by the Free Software Foundation.
 *
 * This library is distributed in the hope that it will be useful, but WITHOUT
 * ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
 * FITNESS FOR A PARTICULAR PURPOSE.  See the GNU Lesser General Public License
 * for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with this library; if not, see <http://www.gnu.org/licenses>.
 */

#ifndef _SECCOMP_H
#error "do not include seccomp-syscalls.h directly, use seccomp.h instead"
#endif

/*
 * psuedo syscall definitions
 */

/* socket syscalls */

#define __PNR_socket				-101
#define __PNR_bind				-102
#define __PNR_connect				-103
#define __PNR_listen				-104
#define __PNR_accept				-105
#define __PNR_getsockname			-106
#define __PNR_getpeername			-107
#define __PNR_socketpair			-108
#define __PNR_send				-109
#define __PNR_recv				-110
#define __PNR_sendto				-111
#define __PNR_recvfrom				-112
#define __PNR_shutdown				-113
#define __PNR_setsockopt			-114
#define __PNR_getsockopt			-115
#define __PNR_sendmsg				-116
#define __PNR_recvmsg				-117
#define __PNR_accept4				-118
#define __PNR_recvmmsg				-119
#define __PNR_sendmmsg				-120

/* ipc syscalls */

#define __PNR_semop				-201
#define __PNR_semget				-202
#define __PNR_semctl				-203
#define __PNR_semtimedop			-204
#define __PNR_msgsnd				-211
#define __PNR_msgrcv				-212
#define __PNR_msgget				-213
#define __PNR_msgctl				-214
#define __PNR_shmat				-221
#define __PNR_shmdt				-222
#define __PNR_shmget				-223
#define __PNR_shmctl				-224

/* single syscalls */

#define __PNR_arch_prctl			-10001
#define __PNR_bdflush				-10002
#define __PNR_break				-10003
#define __PNR_chown32				-10004
#define __PNR_epoll_ctl_old			-10005
#define __PNR_epoll_wait_old			-10006
#define __PNR_fadvise64_64			-10007
#define __PNR_fchown32				-10008
#define __PNR_fcntl64				-10009
#define __PNR_fstat64				-10010
#define __PNR_fstatat64				-10011
#define __PNR_fstatfs64				-10012
#define __PNR_ftime				-10013
#define __PNR_ftruncate64			-10014
#define __PNR_getegid32				-10015
#define __PNR_geteuid32				-10016
#define __PNR_getgid32				-10017
#define __PNR_getgroups32			-10018
#define __PNR_getresgid32			-10019
#define __PNR_getresuid32			-10020
#define __PNR_getuid32				-10021
#define __PNR_gtty				-10022
#define __PNR_idle				-10023
#define __PNR_ipc				-10024
#define __PNR_lchown32				-10025
#define __PNR__llseek				-10026
#define __PNR_lock				-10027
#define __PNR_lstat64				-10028
#define __PNR_mmap2				-10029
#define __PNR_mpx				-10030
#define __PNR_newfstatat			-10031
#define __PNR__newselect			-10032
#define __PNR_nice				-10033
#define __PNR_oldfstat				-10034
#define __PNR_oldlstat				-10035
#define __PNR_oldolduname			-10036
#define __PNR_oldstat				-10037
#define __PNR_olduname				-10038
#define __PNR_prof				-10039
#define __PNR_profil				-10040
#define __PNR_readdir				-10041
#define __PNR_security				-10042
#define __PNR_sendfile64			-10043
#define __PNR_setfsgid32			-10044
#define __PNR_setfsuid32			-10045
#define __PNR_setgid32				-10046
#define __PNR_setgroups32			-10047
#define __PNR_setregid32			-10048
#define __PNR_setresgid32			-10049
#define __PNR_setresuid32			-10050
#define __PNR_setreuid32			-10051
#define __PNR_setuid32				-10052
#define __PNR_sgetmask				-10053
#define __PNR_sigaction				-10054
#define __PNR_signal				-10055
#define __PNR_sigpending			-10056
#define __PNR_sigprocmask			-10057
#define __PNR_sigreturn				-10058
#define __PNR_sigsuspend			-10059
#define __PNR_socketcall			-10060
#define __PNR_ssetmask				-10061
#define __PNR_stat64				-10062
#define __PNR_statfs64				-10063
#define __PNR_stime				-10064
#define __PNR_stty				-10065
#define __PNR_truncate64			-10066
#define __PNR_tuxcall				-10067
#define __PNR_ugetrlimit			-10068
#define __PNR_ulimit				-10069
#define __PNR_umount				-10070
#define __PNR_vm86				-10071
#define __PNR_vm86old				-10072
#define __PNR_waitpid				-10073
#define __PNR_create_module			-10074
#define __PNR_get_kernel_syms			-10075
#define __PNR_get_thread_area			-10076
#define __PNR_nfsservctl			-10077
#define __PNR_query_module			-10078
#define __PNR_set_thread_area			-10079
#define __PNR__sysctl				-10080
#define __PNR_uselib				-10081
#define __PNR_vserver				-10082
#define __PNR_arm_fadvise64_64			-10083
#define __PNR_arm_sync_file_range		-10084
#define __PNR_pciconfig_iobase			-10086
#define __PNR_pciconfig_read			-10087
#define __PNR_pciconfig_write			-10088
#define __PNR_sync_file_range2			-10089
#define __PNR_syscall				-10090
#define __PNR_afs_syscall			-10091
#define __PNR_fadvise64				-10092
#define __PNR_getpmsg				-10093
#define __PNR_ioperm				-10094
#define __PNR_iopl				-10095
#define __PNR_migrate_pages			-10097
#define __PNR_modify_ldt			-10098
#define __PNR_putpmsg				-10099
#define __PNR_sync_file_range			-10100
#define __PNR_select				-10101
#define __PNR_vfork				-10102
#define __PNR_cachectl				-10103
#define __PNR_cacheflush			-10104
#define __PNR_sysmips				-10106
#define __PNR_timerfd				-10107
#define __PNR_time				-10108
#define __PNR_getrandom				-10109
#define __PNR_memfd_create			-10110
#define __PNR_kexec_file_load			-10111
#define __PNR_sysfs				-10145
#define __PNR_oldwait4				-10146
#define __PNR_access				-10147
#define __PNR_alarm				-10148
#define __PNR_chmod				-10149
#define __PNR_chown				-10150
#define __PNR_creat				-10151
#define __PNR_dup2				-10152
#define __PNR_epoll_create			-10153
#define __PNR_epoll_wait			-10154
#define __PNR_eventfd				-10155
#define __PNR_fork				-10156
#define __PNR_futimesat				-10157
#define __PNR_getdents				-10158
#define __PNR_getpgrp				-10159
#define __PNR_inotify_init			-10160
#define __PNR_lchown				-10161
#define __PNR_link				-10162
#define __PNR_lstat				-10163
#define __PNR_mkdir				-10164
#define __PNR_mknod				-10165
#define __PNR_open				-10166
#define __PNR_pause				-10167
#define __PNR_pipe				-10168
#define __PNR_poll				-10169
#define __PNR_readlink				-10170
#define __PNR_rename				-10171
#define __PNR_rmdir				-10172
#define __PNR_signalfd				-10173
#define __PNR_stat				-10174
#define __PNR_symlink				-10175
#define __PNR_unlink				-10176
#define __PNR_ustat				-10177
#define __PNR_utime				-10178
#define __PNR_utimes				-10179
#define __PNR_getrlimit				-10180
#define __PNR_mmap				-10181
#define __PNR_breakpoint			-10182
#define __PNR_set_tls				-10183
#define __PNR_usr26				-10184
#define __PNR_usr32				-10185
#define __PNR_multiplexer			-10186
#define __PNR_rtas				-10187
#define __PNR_spu_create			-10188
#define __PNR_spu_run				-10189
#define __PNR_swapcontext			-10190
#define __PNR_sys_debug_setcontext		-10191
#define __PNR_switch_endian			-10191
#define __PNR_get_mempolicy			-10192
#define __PNR_move_pages			-10193
#define __PNR_mbind				-10194
#define __PNR_set_mempolicy			-10195
#define __PNR_s390_runtime_instr		-10196
#define __PNR_s390_pci_mmio_read		-10197
#define __PNR_s390_pci_mmio_write		-10198
#define __PNR_membarrier			-10199
#define __PNR_userfaultfd			-10200
#define __PNR_pkey_mprotect			-10201
#define __PNR_pkey_alloc			-10202
#define __PNR_pkey_free				-10203
#define __PNR_get_tls				-10204
#define __PNR_s390_guarded_storage		-10205
#define __PNR_s390_sthyi			-10206
#define __PNR_subpage_prot			-10207
#define __PNR_statx				-10208
#define __PNR_io_pgetevents			-10209
#define __PNR_rseq				-10210
#define __PNR_setrlimit				-10211
#define __PNR_clock_adjtime64			-10212
#define __PNR_clock_getres_time64		-10213
#define __PNR_clock_gettime64			-10214
#define __PNR_clock_nanosleep_time64		-10215
#define __PNR_clock_settime64			-10216
#define __PNR_clone3				-10217
#define __PNR_fsconfig				-10218
#define __PNR_fsmount				-10219
#define __PNR_fsopen				-10220
#define __PNR_fspick				-10221
#define __PNR_futex_time64			-10222
#define __PNR_io_pgetevents_time64		-10223
#define __PNR_move_mount			-10224
#define __PNR_mq_timedreceive_time64		-10225
#define __PNR_mq_timedsend_time64		-10226
#define __PNR_open_tree				-10227
#define __PNR_pidfd_open			-10228
#define __PNR_pidfd_send_signal			-10229
#define __PNR_ppoll_time64			-10230
#define __PNR_pselect6_time64			-10231
#define __PNR_recvmmsg_time64			-10232
#define __PNR_rt_sigtimedwait_time64		-10233
#define __PNR_sched_rr_get_interval_time64	-10234
#define __PNR_semtimedop_time64			-10235
#define __PNR_timer_gettime64			-10236
#define __PNR_timer_settime64			-10237
#define __PNR_timerfd_gettime64			-10238
#define __PNR_timerfd_settime64			-10239
#define __PNR_utimensat_time64			-10240
#define __PNR_ppoll				-10241
#define __PNR_renameat				-10242
#define __PNR_riscv_flush_icache		-10243
#define __PNR_memfd_secret			-10244

/*
 * libseccomp syscall definitions
 */

#ifdef __NR__llseek
#define __SNR__llseek			__NR__llseek
#else
#define __SNR__llseek			__PNR__llseek
#endif

#ifdef __NR__newselect
#define __SNR__newselect		__NR__newselect
#else
#define __SNR__newselect		__PNR__newselect
#endif

#ifdef __NR__sysctl
#define __SNR__sysctl			__NR__sysctl
#else
#define __SNR__sysctl			__PNR__sysctl
#endif

#ifdef __NR_accept
#define __SNR_accept			__NR_accept
#else
#define __SNR_accept			__PNR_accept
#endif

#ifdef __NR_accept4
#define __SNR_accept4			__NR_accept4
#else
#define __SNR_accept4			__PNR_accept4
#endif

#ifdef __NR_access
#define __SNR_access			__NR_access
#else
#define __SNR_access			__PNR_access
#endif

#define __SNR_acct			__NR_acct

#define __SNR_add_key			__NR_add_key

#define __SNR_adjtimex			__NR_adjtimex

#ifdef __NR_afs_syscall
#define __SNR_afs_syscall		__NR_afs_syscall
#else
#define __SNR_afs_syscall		__PNR_afs_syscall
#endif

#ifdef __NR_alarm
#define __SNR_alarm			__NR_alarm
#else
#define __SNR_alarm			__PNR_alarm
#endif

#ifdef __NR_arm_fadvise64_64
#define __SNR_arm_fadvise64_64		__NR_arm_fadvise64_64
#else
#define __SNR_arm_fadvise64_64		__PNR_arm_fadvise64_64
#endif

#ifdef __NR_arm_sync_file_range
#define __SNR_arm_sync_file_range	__NR_arm_sync_file_range
#else
#define __SNR_arm_sync_file_range	__PNR_arm_sync_file_range
#endif

#ifdef __NR_arch_prctl
#define __SNR_arch_prctl		__NR_arch_prctl
#else
#define __SNR_arch_prctl		__PNR_arch_prctl
#endif

#ifdef __NR_bdflush
#define __SNR_bdflush			__NR_bdflush
#else
#define __SNR_bdflush			__PNR_bdflush
#endif

#ifdef __NR_bind
#define __SNR_bind			__NR_bind
#else
#define __SNR_bind			__PNR_bind
#endif

#define __SNR_bpf			__NR_bpf

#ifdef __NR_break
#define __SNR_break			__NR_break
#else
#define __SNR_break			__PNR_break
#endif

#ifdef __NR_breakpoint
#ifdef __ARM_NR_breakpoint
#define __SNR_breakpoint		__ARM_NR_breakpoint
#else
#define __SNR_breakpoint		__NR_breakpoint
#endif
#else
#define __SNR_breakpoint		__PNR_breakpoint
#endif

#define __SNR_brk			__NR_brk

#ifdef __NR_cachectl
#define __SNR_cachectl			__NR_cachectl
#else
#define __SNR_cachectl			__PNR_cachectl
#endif

#ifdef __NR_cacheflush
#ifdef __ARM_NR_cacheflush
#define __SNR_cacheflush		__ARM_NR_cacheflush
#else
#define __SNR_cacheflush		__NR_cacheflush
#endif
#else
#define __SNR_cacheflush		__PNR_cacheflush
#endif

#define __SNR_capget			__NR_capget

#define __SNR_capset			__NR_capset

#define __SNR_chdir			__NR_chdir

#ifdef __NR_chmod
#define __SNR_chmod			__NR_chmod
#else
#define __SNR_chmod			__PNR_chmod
#endif

#ifdef __NR_chown
#define __SNR_chown			__NR_chown
#else
#define __SNR_chown			__PNR_chown
#endif

#ifdef __NR_chown32
#define __SNR_chown32			__NR_chown32
#else
#define __SNR_chown32			__PNR_chown32
#endif

#define __SNR_chroot			__NR_chroot

#define __SNR_clock_adjtime		__NR_clock_adjtime

#ifdef __NR_clock_adjtime64
#define __SNR_clock_adjtime64		__NR_clock_adjtime64
#else
#define __SNR_clock_adjtime64		__PNR_clock_adjtime64
#endif

#define __SNR_clock_getres		__NR_clock_getres

#ifdef __NR_clock_getres_time64
#define __SNR_clock_getres_time64	__NR_clock_getres_time64
#else
#define __SNR_clock_getres_time64	__PNR_clock_getres_time64
#endif

#define __SNR_clock_gettime		__NR_clock_gettime

#ifdef __NR_clock_gettime64
#define __SNR_clock_gettime64		__NR_clock_gettime64
#else
#define __SNR_clock_gettime64		__PNR_clock_gettime64
#endif

#define __SNR_clock_nanosleep		__NR_clock_nanosleep

#ifdef __NR_clock_nanosleep_time64
#define __SNR_clock_nanosleep_time64	__NR_clock_nanosleep_time64
#else
#define __SNR_clock_nanosleep_time64	__PNR_clock_nanosleep_time64
#endif

#define __SNR_clock_settime		__NR_clock_settime

#ifdef __NR_clock_settime64
#define __SNR_clock_settime64		__NR_clock_settime64
#else
#define __SNR_clock_settime64		__PNR_clock_settime64
#endif

#define __SNR_clone			__NR_clone

#ifdef __NR_clone3
#define __SNR_clone3			__NR_clone3
#else
#define __SNR_clone3			__PNR_clone3
#endif

#define __SNR_close			__NR_close

#define __SNR_close_range		__NR_close_range

#ifdef __NR_connect
#define __SNR_connect			__NR_connect
#else
#define __SNR_connect			__PNR_connect
#endif

#define __SNR_copy_file_range		__NR_copy_file_range

#ifdef __NR_creat
#define __SNR_creat			__NR_creat
#else
#define __SNR_creat			__PNR_creat
#endif

#ifdef __NR_create_module
#define __SNR_create_module		__NR_create_module
#else
#define __SNR_create_module		__PNR_create_module
#endif

#define __SNR_delete_module		__NR_delete_module

#ifdef __NR_dup
#define __SNR_dup			__NR_dup
#else
#define __SNR_dup			__PNR_dup
#endif

#ifdef __NR_dup2
#define __SNR_dup2			__NR_dup2
#else
#define __SNR_dup2			__PNR_dup2
#endif

#define __SNR_dup3			__NR_dup3

#ifdef __NR_epoll_create
#define __SNR_epoll_create		__NR_epoll_create
#else
#define __SNR_epoll_create		__PNR_epoll_create
#endif

#define __SNR_epoll_create1		__NR_epoll_create1

#ifdef __NR_epoll_ctl
#define __SNR_epoll_ctl			__NR_epoll_ctl
#else
#define __SNR_epoll_ctl			__PNR_epoll_ctl
#endif

#ifdef __NR_epoll_ctl_old
#define __SNR_epoll_ctl_old		__NR_epoll_ctl_old
#else
#define __SNR_epoll_ctl_old		__PNR_epoll_ctl_old
#endif

#define __SNR_epoll_pwait		__NR_epoll_pwait

#define __SNR_epoll_pwait2		__NR_epoll_pwait2

#ifdef __NR_epoll_wait
#define __SNR_epoll_wait		__NR_epoll_wait
#else
#define __SNR_epoll_wait		__PNR_epoll_wait
#endif

#ifdef __NR_epoll_wait_old
#define __SNR_epoll_wait_old		__NR_epoll_wait_old
#else
#define __SNR_epoll_wait_old		__PNR_epoll_wait_old
#endif

#ifdef __NR_eventfd
#define __SNR_eventfd			__NR_eventfd
#else
#define __SNR_eventfd			__PNR_eventfd
#endif

#define __SNR_eventfd2			__NR_eventfd2

#define __SNR_execve			__NR_execve

#define __SNR_execveat			__NR_execveat

#define __SNR_exit			__NR_exit

#define __SNR_exit_group		__NR_exit_group

#define __SNR_faccessat			__NR_faccessat

#define __SNR_faccessat2		__NR_faccessat2

#ifdef __NR_fadvise64
#define __SNR_fadvise64			__NR_fadvise64
#else
#define __SNR_fadvise64			__PNR_fadvise64
#endif

#ifdef __NR_fadvise64_64
#define __SNR_fadvise64_64		__NR_fadvise64_64
#else
#define __SNR_fadvise64_64		__PNR_fadvise64_64
#endif

#define __SNR_fallocate			__NR_fallocate

#define __SNR_fanotify_init		__NR_fanotify_init

#define __SNR_fanotify_mark		__NR_fanotify_mark

#define __SNR_fchdir			__NR_fchdir

#define __SNR_fchmod			__NR_fchmod

#define __SNR_fchmodat			__NR_fchmodat

#ifdef __NR_fchown
#define __SNR_fchown			__NR_fchown
#else
#define __SNR_fchown			__PNR_fchown
#endif

#ifdef __NR_fchown32
#define __SNR_fchown32			__NR_fchown32
#else
#define __SNR_fchown32			__PNR_fchown32
#endif

#define __SNR_fchownat			__NR_fchownat

#ifdef __NR_fcntl
#define __SNR_fcntl			__NR_fcntl
#else
#define __SNR_fcntl			__PNR_fcntl
#endif

#ifdef __NR_fcntl64
#define __SNR_fcntl64			__NR_fcntl64
#else
#define __SNR_fcntl64			__PNR_fcntl64
#endif

#define __SNR_fdatasync			__NR_fdatasync

#define __SNR_fgetxattr			__NR_fgetxattr

#define __SNR_finit_module		__NR_finit_module

#define __SNR_flistxattr		__NR_flistxattr

#define __SNR_flock			__NR_flock

#ifdef __NR_fork
#define __SNR_fork			__NR_fork
#else
#define __SNR_fork			__PNR_fork
#endif

#define __SNR_fremovexattr		__NR_fremovexattr

#ifdef __NR_fsconfig
#define __SNR_fsconfig			__NR_fsconfig
#else
#define __SNR_fsconfig			__PNR_fsconfig
#endif

#define __SNR_fsetxattr			__NR_fsetxattr

#ifdef __NR_fsmount
#define __SNR_fsmount			__NR_fsmount
#else
#define __SNR_fsmount			__PNR_fsmount
#endif

#ifdef __NR_fsopen
#define __SNR_fsopen			__NR_fsopen
#else
#define __SNR_fsopen			__PNR_fsopen
#endif

#ifdef __NR_fspick
#define __SNR_fspick			__NR_fspick
#else
#define __SNR_fspick			__PNR_fspick
#endif

#ifdef __NR_fstat
#define __SNR_fstat			__NR_fstat
#else
#define __SNR_fstat			__PNR_fstat
#endif

#ifdef __NR_fstat64
#define __SNR_fstat64			__NR_fstat64
#else
#define __SNR_fstat64			__PNR_fstat64
#endif

#ifdef __NR_fstatat64
#define __SNR_fstatat64			__NR_fstatat64
#else
#define __SNR_fstatat64			__PNR_fstatat64
#endif

#ifdef __NR_fstatfs
#define __SNR_fstatfs			__NR_fstatfs
#else
#define __SNR_fstatfs			__PNR_fstatfs
#endif

#ifdef __NR_fstatfs64
#define __SNR_fstatfs64			__NR_fstatfs64
#else
#define __SNR_fstatfs64			__PNR_fstatfs64
#endif

#define __SNR_fsync			__NR_fsync

#ifdef __NR_ftime
#define __SNR_ftime			__NR_ftime
#else
#define __SNR_ftime			__PNR_ftime
#endif

#ifdef __NR_ftruncate
#define __SNR_ftruncate			__NR_ftruncate
#else
#define __SNR_ftruncate			__PNR_ftruncate
#endif

#ifdef __NR_ftruncate64
#define __SNR_ftruncate64		__NR_ftruncate64
#else
#define __SNR_ftruncate64		__PNR_ftruncate64
#endif

#define __SNR_futex			__NR_futex

#ifdef __NR_futex_time64
#define __SNR_futex_time64		__NR_futex_time64
#else
#define __SNR_futex_time64		__PNR_futex_time64
#endif

#ifdef __NR_futimesat
#define __SNR_futimesat			__NR_futimesat
#else
#define __SNR_futimesat			__PNR_futimesat
#endif

#ifdef __NR_get_kernel_syms
#define __SNR_get_kernel_syms		__NR_get_kernel_syms
#else
#define __SNR_get_kernel_syms		__PNR_get_kernel_syms
#endif

#ifdef __NR_get_mempolicy
#define __SNR_get_mempolicy		__NR_get_mempolicy
#else
#define __SNR_get_mempolicy		__PNR_get_mempolicy
#endif

#define __SNR_get_robust_list		__NR_get_robust_list

#ifdef __NR_get_thread_area
#define __SNR_get_thread_area		__NR_get_thread_area
#else
#define __SNR_get_thread_area		__PNR_get_thread_area
#endif

#ifdef __NR_get_tls
#ifdef __ARM_NR_get_tls
#define __SNR_get_tls			__ARM_NR_get_tls
#else
#define __SNR_get_tls			__NR_get_tls
#endif
#else
#define __SNR_get_tls			__PNR_get_tls
#endif

#define __SNR_getcpu			__NR_getcpu

#define __SNR_getcwd			__NR_getcwd

#ifdef __NR_getdents
#define __SNR_getdents			__NR_getdents
#else
#define __SNR_getdents			__PNR_getdents
#endif

#define __SNR_getdents64		__NR_getdents64

#ifdef __NR_getegid
#define __SNR_getegid			__NR_getegid
#else
#define __SNR_getegid			__PNR_getegid
#endif

#ifdef __NR_getegid32
#define __SNR_getegid32			__NR_getegid32
#else
#define __SNR_getegid32			__PNR_getegid32
#endif

#ifdef __NR_geteuid
#define __SNR_geteuid			__NR_geteuid
#else
#define __SNR_geteuid			__PNR_geteuid
#endif

#ifdef __NR_geteuid32
#define __SNR_geteuid32			__NR_geteuid32
#else
#define __SNR_geteuid32			__PNR_geteuid32
#endif

#ifdef __NR_getgid
#define __SNR_getgid			__NR_getgid
#else
#define __SNR_getgid			__PNR_getgid
#endif

#ifdef __NR_getgid32
#define __SNR_getgid32			__NR_getgid32
#else
#define __SNR_getgid32			__PNR_getgid32
#endif

#ifdef __NR_getgroups
#define __SNR_getgroups			__NR_getgroups
#else
#define __SNR_getgroups			__PNR_getgroups
#endif

#ifdef __NR_getgroups32
#define __SNR_getgroups32		__NR_getgroups32
#else
#define __SNR_getgroups32		__PNR_getgroups32
#endif

#define __SNR_getitimer			__NR_getitimer

#ifdef __NR_getpeername
#define __SNR_getpeername		__NR_getpeername
#else
#define __SNR_getpeername		__PNR_getpeername
#endif

#define __SNR_getpgid			__NR_getpgid

#ifdef __NR_getpgrp
#define __SNR_getpgrp			__NR_getpgrp
#else
#define __SNR_getpgrp			__PNR_getpgrp
#endif

#define __SNR_getpid			__NR_getpid

#ifdef __NR_getpmsg
#define __SNR_getpmsg			__NR_getpmsg
#else
#define __SNR_getpmsg			__PNR_getpmsg
#endif

#define __SNR_getppid			__NR_getppid

#define __SNR_getpriority		__NR_getpriority

#ifdef __NR_getrandom
#define __SNR_getrandom			__NR_getrandom
#else
#define __SNR_getrandom			__PNR_getrandom
#endif

#ifdef __NR_getresgid
#define __SNR_getresgid			__NR_getresgid
#else
#define __SNR_getresgid			__PNR_getresgid
#endif

#ifdef __NR_getresgid32
#define __SNR_getresgid32		__NR_getresgid32
#else
#define __SNR_getresgid32		__PNR_getresgid32
#endif

#ifdef __NR_getresuid
#define __SNR_getresuid			__NR_getresuid
#else
#define __SNR_getresuid			__PNR_getresuid
#endif

#ifdef __NR_getresuid32
#define __SNR_getresuid32		__NR_getresuid32
#else
#define __SNR_getresuid32		__PNR_getresuid32
#endif

#ifdef __NR_getrlimit
#define __SNR_getrlimit			__NR_getrlimit
#else
#define __SNR_getrlimit			__PNR_getrlimit
#endif

#define __SNR_getrusage			__NR_getrusage

#define __SNR_getsid			__NR_getsid

#ifdef __NR_getsockname
#define __SNR_getsockname		__NR_getsockname
#else
#define __SNR_getsockname		__PNR_getsockname
#endif

#ifdef __NR_getsockopt
#define __SNR_getsockopt		__NR_getsockopt
#else
#define __SNR_getsockopt		__PNR_getsockopt
#endif

#define __SNR_gettid			__NR_gettid

#define __SNR_gettimeofday		__NR_gettimeofday

#ifdef __NR_getuid
#define __SNR_getuid			__NR_getuid
#else
#define __SNR_getuid			__PNR_getuid
#endif

#ifdef __NR_getuid32
#define __SNR_getuid32			__NR_getuid32
#else
#define __SNR_getuid32			__PNR_getuid32
#endif

#define __SNR_getxattr			__NR_getxattr

#ifdef __NR_gtty
#define __SNR_gtty			__NR_gtty
#else
#define __SNR_gtty			__PNR_gtty
#endif

#ifdef __NR_idle
#define __SNR_idle			__NR_idle
#else
#define __SNR_idle			__PNR_idle
#endif

#define __SNR_init_module		__NR_init_module

#define __SNR_inotify_add_watch		__NR_inotify_add_watch

#ifdef __NR_inotify_init
#define __SNR_inotify_init		__NR_inotify_init
#else
#define __SNR_inotify_init		__PNR_inotify_init
#endif

#define __SNR_inotify_init1		__NR_inotify_init1

#define __SNR_inotify_rm_watch		__NR_inotify_rm_watch

#define __SNR_io_cancel			__NR_io_cancel

#define __SNR_io_destroy		__NR_io_destroy

#define __SNR_io_getevents		__NR_io_getevents

#ifdef __NR_io_pgetevents
#define __SNR_io_pgetevents		__NR_io_pgetevents
#else
#define __SNR_io_pgetevents		__PNR_io_pgetevents
#endif

#ifdef __NR_io_pgetevents_time64
#define __SNR_io_pgetevents_time64	__NR_io_pgetevents_time64
#else
#define __SNR_io_pgetevents_time64	__PNR_io_pgetevents_time64
#endif

#define __SNR_io_setup			__NR_io_setup

#define __SNR_io_submit			__NR_io_submit

#define __SNR_io_uring_setup		__NR_io_uring_setup

#define __SNR_io_uring_enter		__NR_io_uring_enter

#define __SNR_io_uring_register		__NR_io_uring_register

#define __SNR_ioctl			__NR_ioctl

#ifdef __NR_ioperm
#define __SNR_ioperm			__NR_ioperm
#else
#define __SNR_ioperm			__PNR_ioperm
#endif

#ifdef __NR_iopl
#define __SNR_iopl			__NR_iopl
#else
#define __SNR_iopl			__PNR_iopl
#endif

#define __SNR_ioprio_get		__NR_ioprio_get

#define __SNR_ioprio_set		__NR_ioprio_set

#ifdef __NR_ipc
#define __SNR_ipc			__NR_ipc
#else
#define __SNR_ipc			__PNR_ipc
#endif

#define __SNR_kcmp			__NR_kcmp

#ifdef __NR_kexec_file_load
#define __SNR_kexec_file_load		__NR_kexec_file_load
#else
#define __SNR_kexec_file_load		__PNR_kexec_file_load
#endif

#define __SNR_kexec_load		__NR_kexec_load

#define __SNR_keyctl			__NR_keyctl

#define __SNR_kill			__NR_kill

#define __SNR_landlock_add_rule		__NR_landlock_add_rule
#define __SNR_landlock_create_ruleset	__NR_landlock_create_ruleset
#define __SNR_landlock_restrict_self	__NR_landlock_restrict_self

#ifdef __NR_lchown
#define __SNR_lchown			__NR_lchown
#else
#define __SNR_lchown			__PNR_lchown
#endif

#ifdef __NR_lchown32
#define __SNR_lchown32			__NR_lchown32
#else
#define __SNR_lchown32			__PNR_lchown32
#endif

#define __SNR_lgetxattr			__NR_lgetxattr

#ifdef __NR_link
#define __SNR_link			__NR_link
#else
#define __SNR_link			__PNR_link
#endif

#define __SNR_linkat			__NR_linkat

#ifdef __NR_listen
#define __SNR_listen			__NR_listen
#else
#define __SNR_listen			__PNR_listen
#endif

#define __SNR_listxattr			__NR_listxattr

#define __SNR_llistxattr		__NR_llistxattr

#ifdef __NR_lock
#define __SNR_lock			__NR_lock
#else
#define __SNR_lock			__PNR_lock
#endif

#define __SNR_lookup_dcookie		__NR_lookup_dcookie

#define __SNR_lremovexattr		__NR_lremovexattr

#define __SNR_lseek			__NR_lseek

#define __SNR_lsetxattr			__NR_lsetxattr

#ifdef __NR_lstat
#define __SNR_lstat			__NR_lstat
#else
#define __SNR_lstat			__PNR_lstat
#endif

#ifdef __NR_lstat64
#define __SNR_lstat64			__NR_lstat64
#else
#define __SNR_lstat64			__PNR_lstat64
#endif

#define __SNR_madvise			__NR_madvise

#ifdef __NR_mbind
#define __SNR_mbind			__NR_mbind
#else
#define __SNR_mbind			__PNR_mbind
#endif

#ifdef __NR_membarrier
#define __SNR_membarrier		__NR_membarrier
#else
#define __SNR_membarrier		__PNR_membarrier
#endif

#ifdef __NR_memfd_create
#define __SNR_memfd_create		__NR_memfd_create
#else
#define __SNR_memfd_create		__PNR_memfd_create
#endif

#ifdef __NR_memfd_secret
#define __SNR_memfd_secret		__NR_memfd_secret
#else
#define __SNR_memfd_secret		__PNR_memfd_secret
#endif

#ifdef __NR_migrate_pages
#define __SNR_migrate_pages		__NR_migrate_pages
#else
#define __SNR_migrate_pages		__PNR_migrate_pages
#endif

#define __SNR_mincore			__NR_mincore

#ifdef __NR_mkdir
#define __SNR_mkdir			__NR_mkdir
#else
#define __SNR_mkdir			__PNR_mkdir
#endif

#define __SNR_mkdirat			__NR_mkdirat

#ifdef __NR_mknod
#define __SNR_mknod			__NR_mknod
#else
#define __SNR_mknod			__PNR_mknod
#endif

#define __SNR_mknodat			__NR_mknodat

#define __SNR_mlock			__NR_mlock

#define __SNR_mlock2			__NR_mlock2

#define __SNR_mlockall			__NR_mlockall

#ifdef __NR_mmap
#define __SNR_mmap			__NR_mmap
#else
#define __SNR_mmap			__PNR_mmap
#endif

#ifdef __NR_mmap2
#define __SNR_mmap2			__NR_mmap2
#else
#define __SNR_mmap2			__PNR_mmap2
#endif

#ifdef __NR_modify_ldt
#define __SNR_modify_ldt		__NR_modify_ldt
#else
#define __SNR_modify_ldt		__PNR_modify_ldt
#endif

#define __SNR_mount			__NR_mount

#define __SNR_mount_setattr		__NR_mount_setattr

#ifdef __NR_move_mount
#define __SNR_move_mount		__NR_move_mount
#else
#define __SNR_move_mount		__PNR_move_mount
#endif

#ifdef __NR_move_pages
#define __SNR_move_pages		__NR_move_pages
#else
#define __SNR_move_pages		__PNR_move_pages
#endif

#define __SNR_mprotect			__NR_mprotect

#ifdef __NR_mpx
#define __SNR_mpx			__NR_mpx
#else
#define __SNR_mpx			__PNR_mpx
#endif

#define __SNR_mq_getsetattr		__NR_mq_getsetattr

#define __SNR_mq_notify			__NR_mq_notify

#define __SNR_mq_open			__NR_mq_open

#define __SNR_mq_timedreceive		__NR_mq_timedreceive

#ifdef __NR_mq_timedreceive_time64
#define __SNR_mq_timedreceive_time64	__NR_mq_timedreceive_time64
#else
#define __SNR_mq_timedreceive_time64	__PNR_mq_timedreceive_time64
#endif

#define __SNR_mq_timedsend		__NR_mq_timedsend

#ifdef __NR_mq_timedsend_time64
#define __SNR_mq_timedsend_time64	__NR_mq_timedsend_time64
#else
#define __SNR_mq_timedsend_time64	__PNR_mq_timedsend_time64
#endif

#define __SNR_mq_unlink			__NR_mq_unlink

#define __SNR_mremap			__NR_mremap

#ifdef __NR_msgctl
#define __SNR_msgctl			__NR_msgctl
#else
#define __SNR_msgctl			__PNR_msgctl
#endif

#ifdef __NR_msgget
#define __SNR_msgget			__NR_msgget
#else
#define __SNR_msgget			__PNR_msgget
#endif

#ifdef __NR_msgrcv
#define __SNR_msgrcv			__NR_msgrcv
#else
#define __SNR_msgrcv			__PNR_msgrcv
#endif

#ifdef __NR_msgsnd
#define __SNR_msgsnd			__NR_msgsnd
#else
#define __SNR_msgsnd			__PNR_msgsnd
#endif

#define __SNR_msync			__NR_msync

#ifdef __NR_multiplexer
#define __SNR_multiplexer		__NR_multiplexer
#else
#define __SNR_multiplexer		__PNR_multiplexer
#endif

#define __SNR_munlock			__NR_munlock

#define __SNR_munlockall			__NR_munlockall

#define __SNR_munmap			__NR_munmap

#define __SNR_name_to_handle_at			__NR_name_to_handle_at

#define __SNR_nanosleep			__NR_nanosleep

#ifdef __NR_newfstatat
#define __SNR_newfstatat		__NR_newfstatat
#else
#define __SNR_newfstatat		__PNR_newfstatat
#endif

#ifdef __NR_nfsservctl
#define __SNR_nfsservctl		__NR_nfsservctl
#else
#define __SNR_nfsservctl		__PNR_nfsservctl
#endif

#ifdef __NR_nice
#define __SNR_nice			__NR_nice
#else
#define __SNR_nice			__PNR_nice
#endif

#ifdef __NR_oldfstat
#define __SNR_oldfstat			__NR_oldfstat
#else
#define __SNR_oldfstat			__PNR_oldfstat
#endif

#ifdef __NR_oldlstat
#define __SNR_oldlstat			__NR_oldlstat
#else
#define __SNR_oldlstat			__PNR_oldlstat
#endif

#ifdef __NR_oldolduname
#define __SNR_oldolduname		__NR_oldolduname
#else
#define __SNR_oldolduname		__PNR_oldolduname
#endif

#ifdef __NR_oldstat
#define __SNR_oldstat			__NR_oldstat
#else
#define __SNR_oldstat			__PNR_oldstat
#endif

#ifdef __NR_olduname
#define __SNR_olduname			__NR_olduname
#else
#define __SNR_olduname			__PNR_olduname
#endif

#ifdef __NR_open
#define __SNR_open			__NR_open
#else
#define __SNR_open			__PNR_open
#endif

#define __SNR_open_by_handle_at		__NR_open_by_handle_at

#ifdef __NR_open_tree
#define __SNR_open_tree			__NR_open_tree
#else
#define __SNR_open_tree			__PNR_open_tree
#endif

#define __SNR_openat			__NR_openat

#define __SNR_openat2			__NR_openat2

#ifdef __NR_pause
#define __SNR_pause			__NR_pause
#else
#define __SNR_pause			__PNR_pause
#endif

#ifdef __NR_pciconfig_iobase
#define __SNR_pciconfig_iobase		__NR_pciconfig_iobase
#else
#define __SNR_pciconfig_iobase		__PNR_pciconfig_iobase
#endif

#ifdef __NR_pciconfig_read
#define __SNR_pciconfig_read		__NR_pciconfig_read
#else
#define __SNR_pciconfig_read		__PNR_pciconfig_read
#endif

#ifdef __NR_pciconfig_write
#define __SNR_pciconfig_write		__NR_pciconfig_write
#else
#define __SNR_pciconfig_write		__PNR_pciconfig_write
#endif

#define __SNR_perf_event_open		__NR_perf_event_open

#define __SNR_personality		__NR_personality

#define __SNR_pidfd_getfd		__NR_pidfd_getfd

#ifdef __NR_pidfd_open
#define __SNR_pidfd_open		__NR_pidfd_open
#else
#define __SNR_pidfd_open		__PNR_pidfd_open
#endif

#ifdef __NR_pidfd_send_signal
#define __SNR_pidfd_send_signal		__NR_pidfd_send_signal
#else
#define __SNR_pidfd_send_signal		__PNR_pidfd_send_signal
#endif

#ifdef __NR_pipe
#define __SNR_pipe			__NR_pipe
#else
#define __SNR_pipe			__PNR_pipe
#endif

#define __SNR_pipe2			__NR_pipe2

#define __SNR_pivot_root		__NR_pivot_root

#ifdef __NR_pkey_alloc
#define __SNR_pkey_alloc		__NR_pkey_alloc
#else
#define __SNR_pkey_alloc		__PNR_pkey_alloc
#endif

#ifdef __NR_pkey_free
#define __SNR_pkey_free			__NR_pkey_free
#else
#define __SNR_pkey_free			__PNR_pkey_free
#endif

#ifdef __NR_pkey_mprotect
#define __SNR_pkey_mprotect		__NR_pkey_mprotect
#else
#define __SNR_pkey_mprotect		__PNR_pkey_mprotect
#endif

#ifdef __NR_poll
#define __SNR_poll			__NR_poll
#else
#define __SNR_poll			__PNR_poll
#endif

#ifdef __NR_ppoll
#define __SNR_ppoll			__NR_ppoll
#else
#define __SNR_ppoll			__PNR_ppoll
#endif

#ifdef __NR_ppoll_time64
#define __SNR_ppoll_time64		__NR_ppoll_time64
#else
#define __SNR_ppoll_time64		__PNR_ppoll_time64
#endif

#define __SNR_prctl			__NR_prctl

#define __SNR_pread64			__NR_pread64

#define __SNR_preadv			__NR_preadv

#define __SNR_preadv2			__NR_preadv2

#define __SNR_prlimit64			__NR_prlimit64

#define __SNR_process_madvise		__NR_process_madvise

#define __SNR_process_mrelease		__NR_process_mrelease

#define __SNR_process_vm_readv		__NR_process_vm_readv

#define __SNR_process_vm_writev		__NR_process_vm_writev

#ifdef __NR_prof
#define __SNR_prof			__NR_prof
#else
#define __SNR_prof			__PNR_prof
#endif

#ifdef __NR_profil
#define __SNR_profil			__NR_profil
#else
#define __SNR_profil			__PNR_profil
#endif

#define __SNR_pselect6			__NR_pselect6

#ifdef __NR_pselect6_time64
#define __SNR_pselect6_time64		__NR_pselect6_time64
#else
#define __SNR_pselect6_time64		__PNR_pselect6_time64
#endif

#define __SNR_ptrace			__NR_ptrace

#ifdef __NR_putpmsg
#define __SNR_putpmsg			__NR_putpmsg
#else
#define __SNR_putpmsg			__PNR_putpmsg
#endif

#define __SNR_pwrite64			__NR_pwrite64

#define __SNR_pwritev			__NR_pwritev

#define __SNR_pwritev2			__NR_pwritev2

#ifdef __NR_query_module
#define __SNR_query_module		__NR_query_module
#else
#define __SNR_query_module		__PNR_query_module
#endif

#define __SNR_quotactl			__NR_quotactl

#define __SNR_quotactl_fd		__NR_quotactl_fd

#ifdef __NR_read
#define __SNR_read			__NR_read
#else
#define __SNR_read			__PNR_read
#endif

#define __SNR_readahead			__NR_readahead

#ifdef __NR_readdir
#define __SNR_readdir			__NR_readdir
#else
#define __SNR_readdir			__PNR_readdir
#endif

#ifdef __NR_readlink
#define __SNR_readlink			__NR_readlink
#else
#define __SNR_readlink			__PNR_readlink
#endif

#define __SNR_readlinkat		__NR_readlinkat

#define __SNR_readv			__NR_readv

#define __SNR_reboot			__NR_reboot

#ifdef __NR_recv
#define __SNR_recv			__NR_recv
#else
#define __SNR_recv			__PNR_recv
#endif

#ifdef __NR_recvfrom
#define __SNR_recvfrom			__NR_recvfrom
#else
#define __SNR_recvfrom			__PNR_recvfrom
#endif

#ifdef __NR_recvmmsg
#define __SNR_recvmmsg			__NR_recvmmsg
#else
#define __SNR_recvmmsg			__PNR_recvmmsg
#endif

#ifdef __NR_recvmmsg_time64
#define __SNR_recvmmsg_time64		__NR_recvmmsg_time64
#else
#define __SNR_recvmmsg_time64		__PNR_recvmmsg_time64
#endif

#ifdef __NR_recvmsg
#define __SNR_recvmsg			__NR_recvmsg
#else
#define __SNR_recvmsg			__PNR_recvmsg
#endif

#define __SNR_remap_file_pages		__NR_remap_file_pages

#define __SNR_removexattr		__NR_removexattr

#ifdef __NR_rename
#define __SNR_rename			__NR_rename
#else
#define __SNR_rename			__PNR_rename
#endif

#ifdef __NR_renameat
#define __SNR_renameat			__NR_renameat
#else
#define __SNR_renameat			__PNR_renameat
#endif

#define __SNR_renameat2			__NR_renameat2

#define __SNR_request_key		__NR_request_key

#define __SNR_restart_syscall		__NR_restart_syscall

#ifdef __NR_riscv_flush_icache
#define __SNR_riscv_flush_icache	__NR_riscv_flush_icache
#else
#define __SNR_riscv_flush_icache	__PNR_riscv_flush_icache
#endif

#ifdef __NR_rmdir
#define __SNR_rmdir			__NR_rmdir
#else
#define __SNR_rmdir			__PNR_rmdir
#endif

#ifdef __NR_rseq
#define __SNR_rseq			__NR_rseq
#else
#define __SNR_rseq			__PNR_rseq
#endif

#define __SNR_rt_sigaction		__NR_rt_sigaction

#define __SNR_rt_sigpending		__NR_rt_sigpending

#define __SNR_rt_sigprocmask		__NR_rt_sigprocmask

#define __SNR_rt_sigqueueinfo		__NR_rt_sigqueueinfo

#define __SNR_rt_sigreturn		__NR_rt_sigreturn

#define __SNR_rt_sigsuspend		__NR_rt_sigsuspend

#define __SNR_rt_sigtimedwait		__NR_rt_sigtimedwait

#ifdef __NR_rt_sigtimedwait_time64
#define __SNR_rt_sigtimedwait_time64	__NR_rt_sigtimedwait_time64
#else
#define __SNR_rt_sigtimedwait_time64	__PNR_rt_sigtimedwait_time64
#endif

#define __SNR_rt_tgsigqueueinfo		__NR_rt_tgsigqueueinfo

#ifdef __NR_rtas
#define __SNR_rtas			__NR_rtas
#else
#define __SNR_rtas			__PNR_rtas
#endif

#ifdef __NR_s390_guarded_storage
#define __SNR_s390_guarded_storage	__NR_s390_guarded_storage
#else
#define __SNR_s390_guarded_storage	__PNR_s390_guarded_storage
#endif

#ifdef __NR_s390_pci_mmio_read
#define __SNR_s390_pci_mmio_read	__NR_s390_pci_mmio_read
#else
#define __SNR_s390_pci_mmio_read	__PNR_s390_pci_mmio_read
#endif

#ifdef __NR_s390_pci_mmio_write
#define __SNR_s390_pci_mmio_write	__NR_s390_pci_mmio_write
#else
#define __SNR_s390_pci_mmio_write	__PNR_s390_pci_mmio_write
#endif

#ifdef __NR_s390_runtime_instr
#define __SNR_s390_runtime_instr	__NR_s390_runtime_instr
#else
#define __SNR_s390_runtime_instr	__PNR_s390_runtime_instr
#endif

#ifdef __NR_s390_sthyi
#define __SNR_s390_sthyi		__NR_s390_sthyi
#else
#define __SNR_s390_sthyi		__PNR_s390_sthyi
#endif

#define __SNR_sched_get_priority_max	__NR_sched_get_priority_max

#define __SNR_sched_get_priority_min	__NR_sched_get_priority_min

#define __SNR_sched_getaffinity		__NR_sched_getaffinity

#define __SNR_sched_getattr		__NR_sched_getattr

#define __SNR_sched_getparam		__NR_sched_getparam

#define __SNR_sched_getscheduler	__NR_sched_getscheduler

#define __SNR_sched_rr_get_interval	__NR_sched_rr_get_interval

#ifdef __NR_sched_rr_get_interval_time64
#define __SNR_sched_rr_get_interval_time64	__NR_sched_rr_get_interval_time64
#else
#define __SNR_sched_rr_get_interval_time64	__PNR_sched_rr_get_interval_time64
#endif

#define __SNR_sched_setaffinity		__NR_sched_setaffinity

#define __SNR_sched_setattr		__NR_sched_setattr

#define __SNR_sched_setparam		__NR_sched_setparam

#define __SNR_sched_setscheduler	__NR_sched_setscheduler

#define __SNR_sched_yield		__NR_sched_yield

#define __SNR_seccomp			__NR_seccomp

#ifdef __NR_security
#define __SNR_security			__NR_security
#else
#define __SNR_security			__PNR_security
#endif

#ifdef __NR_select
#define __SNR_select			__NR_select
#else
#define __SNR_select			__PNR_select
#endif

#ifdef __NR_semctl
#define __SNR_semctl			__NR_semctl
#else
#define __SNR_semctl			__PNR_semctl
#endif

#ifdef __NR_semget
#define __SNR_semget			__NR_semget
#else
#define __SNR_semget			__PNR_semget
#endif

#ifdef __NR_semop
#define __SNR_semop			__NR_semop
#else
#define __SNR_semop			__PNR_semop
#endif

#ifdef __NR_semtimedop
#define __SNR_semtimedop		__NR_semtimedop
#else
#define __SNR_semtimedop		__PNR_semtimedop
#endif

#ifdef __NR_semtimedop_time64
#define __SNR_semtimedop_time64		__NR_semtimedop_time64
#else
#define __SNR_semtimedop_time64		__PNR_semtimedop_time64
#endif

#ifdef __NR_send
#define __SNR_send			__NR_send
#else
#define __SNR_send			__PNR_send
#endif

#ifdef __NR_sendfile
#define __SNR_sendfile			__NR_sendfile
#else
#define __SNR_sendfile			__PNR_sendfile
#endif

#ifdef __NR_sendfile64
#define __SNR_sendfile64		__NR_sendfile64
#else
#define __SNR_sendfile64		__PNR_sendfile64
#endif

#ifdef __NR_sendmmsg
#define __SNR_sendmmsg			__NR_sendmmsg
#else
#define __SNR_sendmmsg			__PNR_sendmmsg
#endif

#ifdef __NR_sendmsg
#define __SNR_sendmsg			__NR_sendmsg
#else
#define __SNR_sendmsg			__PNR_sendmsg
#endif

#ifdef __NR_sendto
#define __SNR_sendto			__NR_sendto
#else
#define __SNR_sendto			__PNR_sendto
#endif

#ifdef __NR_set_mempolicy
#define __SNR_set_mempolicy		__NR_set_mempolicy
#else
#define __SNR_set_mempolicy		__PNR_set_mempolicy
#endif

#define __SNR_set_robust_list		__NR_set_robust_list

#ifdef __NR_set_thread_area
#define __SNR_set_thread_area		__NR_set_thread_area
#else
#define __SNR_set_thread_area		__PNR_set_thread_area
#endif

#define __SNR_set_tid_address		__NR_set_tid_address

#ifdef __NR_set_tls
#ifdef __ARM_NR_set_tls
#define __SNR_set_tls			__ARM_NR_set_tls
#else
#define __SNR_set_tls			__NR_set_tls
#endif
#else
#define __SNR_set_tls			__PNR_set_tls
#endif

#define __SNR_setdomainname		__NR_setdomainname

#ifdef __NR_setfsgid
#define __SNR_setfsgid			__NR_setfsgid
#else
#define __SNR_setfsgid			__PNR_setfsgid
#endif

#ifdef __NR_setfsgid32
#define __SNR_setfsgid32		__NR_setfsgid32
#else
#define __SNR_setfsgid32		__PNR_setfsgid32
#endif

#ifdef __NR_setfsuid
#define __SNR_setfsuid			__NR_setfsuid
#else
#define __SNR_setfsuid			__PNR_setfsuid
#endif

#ifdef __NR_setfsuid32
#define __SNR_setfsuid32		__NR_setfsuid32
#else
#define __SNR_setfsuid32		__PNR_setfsuid32
#endif

#ifdef __NR_setgid
#define __SNR_setgid			__NR_setgid
#else
#define __SNR_setgid			__PNR_setgid
#endif

#ifdef __NR_setgid32
#define __SNR_setgid32			__NR_setgid32
#else
#define __SNR_setgid32			__PNR_setgid32
#endif

#ifdef __NR_setgroups
#define __SNR_setgroups			__NR_setgroups
#else
#define __SNR_setgroups			__PNR_setgroups
#endif

#ifdef __NR_setgroups32
#define __SNR_setgroups32		__NR_setgroups32
#else
#define __SNR_setgroups32		__PNR_setgroups32
#endif

#define __SNR_sethostname		__NR_sethostname

#define __SNR_setitimer			__NR_setitimer

#define __SNR_setns			__NR_setns

#define __SNR_setpgid			__NR_setpgid

#define __SNR_setpriority		__NR_setpriority

#ifdef __NR_setregid
#define __SNR_setregid			__NR_setregid
#else
#define __SNR_setregid			__PNR_setregid
#endif

#ifdef __NR_setregid32
#define __SNR_setregid32		__NR_setregid32
#else
#define __SNR_setregid32		__PNR_setregid32
#endif

#ifdef __NR_setresgid
#define __SNR_setresgid			__NR_setresgid
#else
#define __SNR_setresgid			__PNR_setresgid
#endif

#ifdef __NR_setresgid32
#define __SNR_setresgid32		__NR_setresgid32
#else
#define __SNR_setresgid32		__PNR_setresgid32
#endif

#ifdef __NR_setresuid
#define __SNR_setresuid			__NR_setresuid
#else
#define __SNR_setresuid			__PNR_setresuid
#endif

#ifdef __NR_setresuid32
#define __SNR_setresuid32		__NR_setresuid32
#else
#define __SNR_setresuid32		__PNR_setresuid32
#endif

#ifdef __NR_setreuid
#define __SNR_setreuid			__NR_setreuid
#else
#define __SNR_setreuid			__PNR_setreuid
#endif

#ifdef __NR_setreuid32
#define __SNR_setreuid32		__NR_setreuid32
#else
#define __SNR_setreuid32		__PNR_setreuid32
#endif

#ifdef __NR_setrlimit
#define __SNR_setrlimit			__NR_setrlimit
#else
#define __SNR_setrlimit			__PNR_setrlimit
#endif

#define __SNR_setsid			__NR_setsid

#ifdef __NR_setsockopt
#define __SNR_setsockopt		__NR_setsockopt
#else
#define __SNR_setsockopt		__PNR_setsockopt
#endif

#define __SNR_settimeofday		__NR_settimeofday

#ifdef __NR_setuid
#define __SNR_setuid			__NR_setuid
#else
#define __SNR_setuid			__PNR_setuid
#endif

#ifdef __NR_setuid32
#define __SNR_setuid32			__NR_setuid32
#else
#define __SNR_setuid32			__PNR_setuid32
#endif

#define __SNR_setxattr			__NR_setxattr

#ifdef __NR_sgetmask
#define __SNR_sgetmask			__NR_sgetmask
#else
#define __SNR_sgetmask			__PNR_sgetmask
#endif

#ifdef __NR_shmat
#define __SNR_shmat			__NR_shmat
#else
#define __SNR_shmat			__PNR_shmat
#endif

#ifdef __NR_shmctl
#define __SNR_shmctl			__NR_shmctl
#else
#define __SNR_shmctl			__PNR_shmctl
#endif

#ifdef __NR_shmdt
#define __SNR_shmdt			__NR_shmdt
#else
#define __SNR_shmdt			__PNR_shmdt
#endif

#ifdef __NR_shmget
#define __SNR_shmget			__NR_shmget
#else
#define __SNR_shmget			__PNR_shmget
#endif

#ifdef __NR_shutdown
#define __SNR_shutdown			__NR_shutdown
#else
#define __SNR_shutdown			__PNR_shutdown
#endif

#ifdef __NR_sigaction
#define __SNR_sigaction			__NR_sigaction
#else
#define __SNR_sigaction			__PNR_sigaction
#endif

#define __SNR_sigaltstack		__NR_sigaltstack

#ifdef __NR_signal
#define __SNR_signal			__NR_signal
#else
#define __SNR_signal			__PNR_signal
#endif

#ifdef __NR_signalfd
#define __SNR_signalfd			__NR_signalfd
#else
#define __SNR_signalfd			__PNR_signalfd
#endif

#define __SNR_signalfd4			__NR_signalfd4

#ifdef __NR_sigpending
#define __SNR_sigpending		__NR_sigpending
#else
#define __SNR_sigpending		__PNR_sigpending
#endif

#ifdef __NR_sigprocmask
#define __SNR_sigprocmask		__NR_sigprocmask
#else
#define __SNR_sigprocmask		__PNR_sigprocmask
#endif

#ifdef __NR_sigreturn
#define __SNR_sigreturn			__NR_sigreturn
#else
#define __SNR_sigreturn			__PNR_sigreturn
#endif

#ifdef __NR_sigsuspend
#define __SNR_sigsuspend		__NR_sigsuspend
#else
#define __SNR_sigsuspend		__PNR_sigsuspend
#endif

#ifdef __NR_socket
#define __SNR_socket			__NR_socket
#else
#define __SNR_socket			__PNR_socket
#endif

#ifdef __NR_socketcall
#define __SNR_socketcall		__NR_socketcall
#else
#define __SNR_socketcall		__PNR_socketcall
#endif

#ifdef __NR_socketpair
#define __SNR_socketpair		__NR_socketpair
#else
#define __SNR_socketpair		__PNR_socketpair
#endif

#define __SNR_splice			__NR_splice

#ifdef __NR_spu_create
#define __SNR_spu_create		__NR_spu_create
#else
#define __SNR_spu_create		__PNR_spu_create
#endif

#ifdef __NR_spu_run
#define __SNR_spu_run			__NR_spu_run
#else
#define __SNR_spu_run			__PNR_spu_run
#endif

#ifdef __NR_ssetmask
#define __SNR_ssetmask			__NR_ssetmask
#else
#define __SNR_ssetmask			__PNR_ssetmask
#endif

#ifdef __NR_stat
#define __SNR_stat			__NR_stat
#else
#define __SNR_stat			__PNR_stat
#endif

#ifdef __NR_stat64
#define __SNR_stat64			__NR_stat64
#else
#define __SNR_stat64			__PNR_stat64
#endif

#ifdef __NR_statfs
#define __SNR_statfs			__NR_statfs
#else
#define __SNR_statfs			__PNR_statfs
#endif

#ifdef __NR_statfs64
#define __SNR_statfs64			__NR_statfs64
#else
#define __SNR_statfs64			__PNR_statfs64
#endif

#ifdef __NR_statx
#define __SNR_statx			__NR_statx
#else
#define __SNR_statx			__PNR_statx
#endif

#ifdef __NR_stime
#define __SNR_stime			__NR_stime
#else
#define __SNR_stime			__PNR_stime
#endif

#ifdef __NR_stty
#define __SNR_stty			__NR_stty
#else
#define __SNR_stty			__PNR_stty
#endif

#ifdef __NR_subpage_prot
#define __SNR_subpage_prot		__NR_subpage_prot
#else
#define __SNR_subpage_prot		__PNR_subpage_prot
#endif

#ifdef __NR_swapcontext
#define __SNR_swapcontext		__NR_swapcontext
#else
#define __SNR_swapcontext		__PNR_swapcontext
#endif

#define __SNR_swapoff			__NR_swapoff

#define __SNR_swapon			__NR_swapon

#ifdef __NR_switch_endian
#define __SNR_switch_endian		__NR_switch_endian
#else
#define __SNR_switch_endian		__PNR_switch_endian
#endif

#ifdef __NR_symlink
#define __SNR_symlink			__NR_symlink
#else
#define __SNR_symlink			__PNR_symlink
#endif

#define __SNR_symlinkat			__NR_symlinkat

#ifdef __NR_sync
#define __SNR_sync			__NR_sync
#else
#define __SNR_sync			__PNR_sync
#endif

#ifdef __NR_sync_file_range
#define __SNR_sync_file_range		__NR_sync_file_range
#else
#define __SNR_sync_file_range		__PNR_sync_file_range
#endif

#ifdef __NR_sync_file_range2
#define __SNR_sync_file_range2		__NR_sync_file_range2
#else
#define __SNR_sync_file_range2		__PNR_sync_file_range2
#endif

#define __SNR_syncfs			__NR_syncfs

#ifdef __NR_syscall
#define __SNR_syscall			__NR_syscall
#else
#define __SNR_syscall			__PNR_syscall
#endif

#ifdef __NR_sys_debug_setcontext
#define __SNR_sys_debug_setcontext	__NR_sys_debug_setcontext
#else
#define __SNR_sys_debug_setcontext	__PNR_sys_debug_setcontext
#endif

#ifdef __NR_sysfs
#define __SNR_sysfs			__NR_sysfs
#else
#define __SNR_sysfs			__PNR_sysfs
#endif

#define __SNR_sysinfo			__NR_sysinfo

#define __SNR_syslog			__NR_syslog

#ifdef __NR_sysmips
#define __SNR_sysmips			__NR_sysmips
#else
#define __SNR_sysmips			__PNR_sysmips
#endif

#define __SNR_tee			__NR_tee

#define __SNR_tgkill			__NR_tgkill

#ifdef __NR_time
#define __SNR_time			__NR_time
#else
#define __SNR_time			__PNR_time
#endif

#define __SNR_timer_create		__NR_timer_create

#define __SNR_timer_delete		__NR_timer_delete

#define __SNR_timer_getoverrun		__NR_timer_getoverrun

#define __SNR_timer_gettime		__NR_timer_gettime

#ifdef __NR_timer_gettime64
#define __SNR_timer_gettime64		__NR_timer_gettime64
#else
#define __SNR_timer_gettime64		__PNR_timer_gettime64
#endif

#define __SNR_timer_settime		__NR_timer_settime

#ifdef __NR_timer_settime64
#define __SNR_timer_settime64		__NR_timer_settime64
#else
#define __SNR_timer_settime64		__PNR_timer_settime64
#endif

#ifdef __NR_timerfd
#define __SNR_timerfd			__NR_timerfd
#else
#define __SNR_timerfd			__PNR_timerfd
#endif

#define __SNR_timerfd_create		__NR_timerfd_create

#define __SNR_timerfd_gettime		__NR_timerfd_gettime

#ifdef __NR_timerfd_gettime64
#define __SNR_timerfd_gettime64		__NR_timerfd_gettime64
#else
#define __SNR_timerfd_gettime64		__PNR_timerfd_gettime64
#endif

#define __SNR_timerfd_settime		__NR_timerfd_settime

#ifdef __NR_timerfd_settime64
#define __SNR_timerfd_settime64		__NR_timerfd_settime64
#else
#define __SNR_timerfd_settime64		__PNR_timerfd_settime64
#endif

#define __SNR_times			__NR_times

#define __SNR_tkill			__NR_tkill

#ifdef __NR_truncate
#define __SNR_truncate			__NR_truncate
#else
#define __SNR_truncate			__PNR_truncate
#endif

#ifdef __NR_truncate64
#define __SNR_truncate64		__NR_truncate64
#else
#define __SNR_truncate64		__PNR_truncate64
#endif

#ifdef __NR_tuxcall
#define __SNR_tuxcall			__NR_tuxcall
#else
#define __SNR_tuxcall			__PNR_tuxcall
#endif

#ifdef __NR_ugetrlimit
#define __SNR_ugetrlimit		__NR_ugetrlimit
#else
#define __SNR_ugetrlimit		__PNR_ugetrlimit
#endif

#ifdef __NR_ulimit
#define __SNR_ulimit			__NR_ulimit
#else
#define __SNR_ulimit			__PNR_ulimit
#endif

#define __SNR_umask			__NR_umask

#ifdef __NR_umount
#define __SNR_umount			__NR_umount
#else
#define __SNR_umount			__PNR_umount
#endif

#define __SNR_umount2			__NR_umount2

#define __SNR_uname			__NR_uname

#ifdef __NR_unlink
#define __SNR_unlink			__NR_unlink
#else
#define __SNR_unlink			__PNR_unlink
#endif

#define __SNR_unlinkat			__NR_unlinkat

#define __SNR_unshare			__NR_unshare

#ifdef __NR_uselib
#define __SNR_uselib			__NR_uselib
#else
#define __SNR_uselib			__PNR_uselib
#endif

#ifdef __NR_userfaultfd
#define __SNR_userfaultfd		__NR_userfaultfd
#else
#define __SNR_userfaultfd		__PNR_userfaultfd
#endif

#ifdef __NR_usr26
#ifdef __ARM_NR_usr26
#define __SNR_usr26			__NR_usr26
#else
#define __SNR_usr26			__NR_usr26
#endif
#else
#define __SNR_usr26			__PNR_usr26
#endif

#ifdef __NR_usr32
#ifdef __ARM_NR_usr32
#define __SNR_usr32			__NR_usr32
#else
#define __SNR_usr32			__NR_usr32
#endif
#else
#define __SNR_usr32			__PNR_usr32
#endif

#ifdef __NR_ustat
#define __SNR_ustat			__NR_ustat
#else
#define __SNR_ustat			__PNR_ustat
#endif

#ifdef __NR_utime
#define __SNR_utime			__NR_utime
#else
#define __SNR_utime			__PNR_utime
#endif

#define __SNR_utimensat			__NR_utimensat

#ifdef __NR_utimensat_time64
#define __SNR_utimensat_time64		__NR_utimensat_time64
#else
#define __SNR_utimensat_time64		__PNR_utimensat_time64
#endif

#ifdef __NR_utimes
#define __SNR_utimes			__NR_utimes
#else
#define __SNR_utimes			__PNR_utimes
#endif

#ifdef __NR_vfork
#define __SNR_vfork			__NR_vfork
#else
#define __SNR_vfork			__PNR_vfork
#endif

#define __SNR_vhangup			__NR_vhangup

#ifdef __NR_vm86
#define __SNR_vm86			__NR_vm86
#else
#define __SNR_vm86			__PNR_vm86
#endif

#ifdef __NR_vm86old
#define __SNR_vm86old			__NR_vm86old
#else
#define __SNR_vm86old			__PNR_vm86old
#endif

#define __SNR_vmsplice			__NR_vmsplice

#ifdef __NR_vserver
#define __SNR_vserver			__NR_vserver
#else
#define __SNR_vserver			__PNR_vserver
#endif

#define __SNR_wait4			__NR_wait4

#define __SNR_waitid			__NR_waitid

#ifdef __NR_waitpid
#define __SNR_waitpid			__NR_waitpid
#else
#define __SNR_waitpid			__PNR_waitpid
#endif

#define __SNR_write			__NR_write

#define __SNR_writev			__NR_writev
