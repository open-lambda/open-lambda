/**
 * Seccomp Library
 *
 * Copyright (c) 2019 Cisco Systems <pmoore2@cisco.com>
 * Copyright (c) 2012,2013 Red Hat <pmoore@redhat.com>
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
#define _SECCOMP_H

#include <elf.h>
#include <inttypes.h>
#include <asm/unistd.h>
#include <linux/audit.h>
#include <linux/types.h>
#include <linux/seccomp.h>

#ifdef __cplusplus
extern "C" {
#endif

/*
 * version information
 */

#define SCMP_VER_MAJOR		2
#define SCMP_VER_MINOR		5
#define SCMP_VER_MICRO		3

struct scmp_version {
	unsigned int major;
	unsigned int minor;
	unsigned int micro;
};

/*
 * types
 */

/**
 * Filter context/handle
 */
typedef void *scmp_filter_ctx;

/**
 * Filter attributes
 */
enum scmp_filter_attr {
	_SCMP_FLTATR_MIN = 0,
	SCMP_FLTATR_ACT_DEFAULT = 1,	/**< default filter action */
	SCMP_FLTATR_ACT_BADARCH = 2,	/**< bad architecture action */
	SCMP_FLTATR_CTL_NNP = 3,	/**< set NO_NEW_PRIVS on filter load */
	SCMP_FLTATR_CTL_TSYNC = 4,	/**< sync threads on filter load */
	SCMP_FLTATR_API_TSKIP = 5,	/**< allow rules with a -1 syscall */
	SCMP_FLTATR_CTL_LOG = 6,	/**< log not-allowed actions */
	SCMP_FLTATR_CTL_SSB = 7,	/**< disable SSB mitigation */
	SCMP_FLTATR_CTL_OPTIMIZE = 8,	/**< filter optimization level:
					 * 0 - currently unused
					 * 1 - rules weighted by priority and
					 *     complexity (DEFAULT)
					 * 2 - binary tree sorted by syscall
					 *     number
					 */
	SCMP_FLTATR_API_SYSRAWRC = 9,	/**< return the system return codes */
	_SCMP_FLTATR_MAX,
};

/**
 * Comparison operators
 */
enum scmp_compare {
	_SCMP_CMP_MIN = 0,
	SCMP_CMP_NE = 1,		/**< not equal */
	SCMP_CMP_LT = 2,		/**< less than */
	SCMP_CMP_LE = 3,		/**< less than or equal */
	SCMP_CMP_EQ = 4,		/**< equal */
	SCMP_CMP_GE = 5,		/**< greater than or equal */
	SCMP_CMP_GT = 6,		/**< greater than */
	SCMP_CMP_MASKED_EQ = 7,		/**< masked equality */
	_SCMP_CMP_MAX,
};

/**
 * Argument datum
 */
typedef uint64_t scmp_datum_t;

/**
 * Argument / Value comparison definition
 */
struct scmp_arg_cmp {
	unsigned int arg;	/**< argument number, starting at 0 */
	enum scmp_compare op;	/**< the comparison op, e.g. SCMP_CMP_* */
	scmp_datum_t datum_a;
	scmp_datum_t datum_b;
};

/*
 * macros/defines
 */

/**
 * The native architecture token
 */
#define SCMP_ARCH_NATIVE	0

/**
 * The x86 (32-bit) architecture token
 */
#define SCMP_ARCH_X86		AUDIT_ARCH_I386

/**
 * The x86-64 (64-bit) architecture token
 */
#define SCMP_ARCH_X86_64	AUDIT_ARCH_X86_64

/**
 * The x32 (32-bit x86_64) architecture token
 *
 * NOTE: this is different from the value used by the kernel because we need to
 * be able to distinguish between x32 and x86_64
 */
#define SCMP_ARCH_X32		(EM_X86_64|__AUDIT_ARCH_LE)

/**
 * The ARM architecture tokens
 */
#define SCMP_ARCH_ARM		AUDIT_ARCH_ARM
/* AArch64 support for audit was merged in 3.17-rc1 */
#ifndef AUDIT_ARCH_AARCH64
#ifndef EM_AARCH64
#define EM_AARCH64		183
#endif /* EM_AARCH64 */
#define AUDIT_ARCH_AARCH64	(EM_AARCH64|__AUDIT_ARCH_64BIT|__AUDIT_ARCH_LE)
#endif /* AUDIT_ARCH_AARCH64 */
#define SCMP_ARCH_AARCH64	AUDIT_ARCH_AARCH64

/**
 * The MIPS architecture tokens
 */
#ifndef __AUDIT_ARCH_CONVENTION_MIPS64_N32
#define __AUDIT_ARCH_CONVENTION_MIPS64_N32	0x20000000
#endif
#ifndef EM_MIPS
#define EM_MIPS			8
#endif
#ifndef AUDIT_ARCH_MIPS
#define AUDIT_ARCH_MIPS		(EM_MIPS)
#endif
#ifndef AUDIT_ARCH_MIPS64
#define AUDIT_ARCH_MIPS64	(EM_MIPS|__AUDIT_ARCH_64BIT)
#endif
/* MIPS64N32 support was merged in 3.15 */
#ifndef AUDIT_ARCH_MIPS64N32
#define AUDIT_ARCH_MIPS64N32	(EM_MIPS|__AUDIT_ARCH_64BIT|\
				 __AUDIT_ARCH_CONVENTION_MIPS64_N32)
#endif
/* MIPSEL64N32 support was merged in 3.15 */
#ifndef AUDIT_ARCH_MIPSEL64N32
#define AUDIT_ARCH_MIPSEL64N32	(EM_MIPS|__AUDIT_ARCH_64BIT|__AUDIT_ARCH_LE|\
				 __AUDIT_ARCH_CONVENTION_MIPS64_N32)
#endif
#define SCMP_ARCH_MIPS		AUDIT_ARCH_MIPS
#define SCMP_ARCH_MIPS64	AUDIT_ARCH_MIPS64
#define SCMP_ARCH_MIPS64N32	AUDIT_ARCH_MIPS64N32
#define SCMP_ARCH_MIPSEL	AUDIT_ARCH_MIPSEL
#define SCMP_ARCH_MIPSEL64	AUDIT_ARCH_MIPSEL64
#define SCMP_ARCH_MIPSEL64N32	AUDIT_ARCH_MIPSEL64N32

/**
 * The PowerPC architecture tokens
 */
#define SCMP_ARCH_PPC		AUDIT_ARCH_PPC
#define SCMP_ARCH_PPC64		AUDIT_ARCH_PPC64
#ifndef AUDIT_ARCH_PPC64LE
#define AUDIT_ARCH_PPC64LE	(EM_PPC64|__AUDIT_ARCH_64BIT|__AUDIT_ARCH_LE)
#endif
#define SCMP_ARCH_PPC64LE	AUDIT_ARCH_PPC64LE

/**
 * The S390 architecture tokens
 */
#define SCMP_ARCH_S390		AUDIT_ARCH_S390
#define SCMP_ARCH_S390X		AUDIT_ARCH_S390X

/**
 * The PA-RISC hppa architecture tokens
 */
#define SCMP_ARCH_PARISC	AUDIT_ARCH_PARISC
#define SCMP_ARCH_PARISC64	AUDIT_ARCH_PARISC64

/**
 * The RISC-V architecture tokens
 */
/* RISC-V support for audit was merged in 5.0-rc1 */
#ifndef AUDIT_ARCH_RISCV64
#ifndef EM_RISCV
#define EM_RISCV		243
#endif /* EM_RISCV */
#define AUDIT_ARCH_RISCV64	(EM_RISCV|__AUDIT_ARCH_64BIT|__AUDIT_ARCH_LE)
#endif /* AUDIT_ARCH_RISCV64 */
#define SCMP_ARCH_RISCV64	AUDIT_ARCH_RISCV64

/**
 * Convert a syscall name into the associated syscall number
 * @param x the syscall name
 */
#define SCMP_SYS(x)		(__SNR_##x)

/* Helpers for the argument comparison macros, DO NOT USE directly */
#define _SCMP_VA_NUM_ARGS(...)	_SCMP_VA_NUM_ARGS_IMPL(__VA_ARGS__,2,1)
#define _SCMP_VA_NUM_ARGS_IMPL(_1,_2,N,...)	N
#define _SCMP_MACRO_DISPATCHER(func, ...) \
	_SCMP_MACRO_DISPATCHER_IMPL1(func, _SCMP_VA_NUM_ARGS(__VA_ARGS__))
#define _SCMP_MACRO_DISPATCHER_IMPL1(func, nargs) \
	_SCMP_MACRO_DISPATCHER_IMPL2(func, nargs)
#define _SCMP_MACRO_DISPATCHER_IMPL2(func, nargs) \
	func ## nargs
#define _SCMP_CMP32_1(x, y, z) \
	SCMP_CMP64(x, y, (uint32_t)(z))
#define _SCMP_CMP32_2(x, y, z, q) \
	SCMP_CMP64(x, y, (uint32_t)(z), (uint32_t)(q))

/**
 * Specify a 64-bit argument comparison struct for use in declaring rules
 * @param arg the argument number, starting at 0
 * @param op the comparison operator, e.g. SCMP_CMP_*
 * @param datum_a dependent on comparison
 * @param datum_b dependent on comparison, optional
 */
#define SCMP_CMP64(...)		((struct scmp_arg_cmp){__VA_ARGS__})
#define SCMP_CMP		SCMP_CMP64

/**
 * Specify a 32-bit argument comparison struct for use in declaring rules
 * @param arg the argument number, starting at 0
 * @param op the comparison operator, e.g. SCMP_CMP_*
 * @param datum_a dependent on comparison (32-bits)
 * @param datum_b dependent on comparison, optional (32-bits)
 */
#define SCMP_CMP32(x, y, ...) \
	_SCMP_MACRO_DISPATCHER(_SCMP_CMP32_, __VA_ARGS__)(x, y, __VA_ARGS__)

/**
 * Specify a 64-bit argument comparison struct for argument 0
 */
#define SCMP_A0_64(...)		SCMP_CMP64(0, __VA_ARGS__)
#define SCMP_A0			SCMP_A0_64

/**
 * Specify a 32-bit argument comparison struct for argument 0
 */
#define SCMP_A0_32(x, ...)	SCMP_CMP32(0, x, __VA_ARGS__)

/**
 * Specify a 64-bit argument comparison struct for argument 1
 */
#define SCMP_A1_64(...)		SCMP_CMP64(1, __VA_ARGS__)
#define SCMP_A1			SCMP_A1_64

/**
 * Specify a 32-bit argument comparison struct for argument 1
 */
#define SCMP_A1_32(x, ...)	SCMP_CMP32(1, x, __VA_ARGS__)

/**
 * Specify a 64-bit argument comparison struct for argument 2
 */
#define SCMP_A2_64(...)		SCMP_CMP64(2, __VA_ARGS__)
#define SCMP_A2			SCMP_A2_64

/**
 * Specify a 32-bit argument comparison struct for argument 2
 */
#define SCMP_A2_32(x, ...)	SCMP_CMP32(2, x, __VA_ARGS__)

/**
 * Specify a 64-bit argument comparison struct for argument 3
 */
#define SCMP_A3_64(...)		SCMP_CMP64(3, __VA_ARGS__)
#define SCMP_A3			SCMP_A3_64

/**
 * Specify a 32-bit argument comparison struct for argument 3
 */
#define SCMP_A3_32(x, ...)	SCMP_CMP32(3, x, __VA_ARGS__)

/**
 * Specify a 64-bit argument comparison struct for argument 4
 */
#define SCMP_A4_64(...)		SCMP_CMP64(4, __VA_ARGS__)
#define SCMP_A4			SCMP_A4_64

/**
 * Specify a 32-bit argument comparison struct for argument 4
 */
#define SCMP_A4_32(x, ...)	SCMP_CMP32(4, x, __VA_ARGS__)

/**
 * Specify a 64-bit argument comparison struct for argument 5
 */
#define SCMP_A5_64(...)		SCMP_CMP64(5, __VA_ARGS__)
#define SCMP_A5			SCMP_A5_64

/**
 * Specify a 32-bit argument comparison struct for argument 5
 */
#define SCMP_A5_32(x, ...)	SCMP_CMP32(5, x, __VA_ARGS__)

/*
 * seccomp actions
 */

/**
 * Kill the process
 */
#define SCMP_ACT_KILL_PROCESS	0x80000000U
/**
 * Kill the thread
 */
#define SCMP_ACT_KILL_THREAD	0x00000000U
/**
 * Kill the thread, defined for backward compatibility
 */
#define SCMP_ACT_KILL		SCMP_ACT_KILL_THREAD
/**
 * Throw a SIGSYS signal
 */
#define SCMP_ACT_TRAP		0x00030000U
/**
 * Notifies userspace
 */
#define SCMP_ACT_NOTIFY	 	0x7fc00000U
/**
 * Return the specified error code
 */
#define SCMP_ACT_ERRNO(x)	(0x00050000U | ((x) & 0x0000ffffU))
/**
 * Notify a tracing process with the specified value
 */
#define SCMP_ACT_TRACE(x)	(0x7ff00000U | ((x) & 0x0000ffffU))
/**
 * Allow the syscall to be executed after the action has been logged
 */
#define SCMP_ACT_LOG		0x7ffc0000U
/**
 * Allow the syscall to be executed
 */
#define SCMP_ACT_ALLOW		0x7fff0000U

/* SECCOMP_RET_USER_NOTIF was added in kernel v5.0. */
#ifndef SECCOMP_RET_USER_NOTIF
#define SECCOMP_RET_USER_NOTIF	0x7fc00000U

struct seccomp_notif {
	__u64 id;
	__u32 pid;
	__u32 flags;
	struct seccomp_data data;
};

struct seccomp_notif_resp {
	__u64 id;
	__s64 val;
	__s32 error;
	__u32 flags;
};
#endif

/*
 * functions
 */

/**
 * Query the library version information
 *
 * This function returns a pointer to a populated scmp_version struct, the
 * caller does not need to free the structure when finished.
 *
 */
const struct scmp_version *seccomp_version(void);

/**
 * Query the library's level of API support
 *
 * This function returns an API level value indicating the current supported
 * functionality.  It is important to note that this level of support is
 * determined at runtime and therefore can change based on the running kernel
 * and system configuration (e.g. any previously loaded seccomp filters).  This
 * function can be called multiple times, but it only queries the system the
 * first time it is called, the API level is cached and used in subsequent
 * calls.
 *
 * The current API levels are described below:
 *  0 : reserved
 *  1 : base level
 *  2 : support for the SCMP_FLTATR_CTL_TSYNC filter attribute
 *      uses the seccomp(2) syscall instead of the prctl(2) syscall
 *  3 : support for the SCMP_FLTATR_CTL_LOG filter attribute
 *      support for the SCMP_ACT_LOG action
 *      support for the SCMP_ACT_KILL_PROCESS action
 *  4 : support for the SCMP_FLTATR_CTL_SSB filter attrbute
 *  5 : support for the SCMP_ACT_NOTIFY action and notify APIs
 *  6 : support the simultaneous use of SCMP_FLTATR_CTL_TSYNC and notify APIs
 *
 */
unsigned int seccomp_api_get(void);

/**
 * Set the library's level of API support
 *
 * This function forcibly sets the API level of the library at runtime.  Valid
 * API levels are discussed in the description of the seccomp_api_get()
 * function.  General use of this function is strongly discouraged.
 *
 */
int seccomp_api_set(unsigned int level);

/**
 * Initialize the filter state
 * @param def_action the default filter action
 *
 * This function initializes the internal seccomp filter state and should
 * be called before any other functions in this library to ensure the filter
 * state is initialized.  Returns a filter context on success, NULL on failure.
 *
 */
scmp_filter_ctx seccomp_init(uint32_t def_action);

/**
 * Reset the filter state
 * @param ctx the filter context
 * @param def_action the default filter action
 *
 * This function resets the given seccomp filter state and ensures the
 * filter state is reinitialized.  This function does not reset any seccomp
 * filters already loaded into the kernel.  Returns zero on success, negative
 * values on failure.
 *
 */
int seccomp_reset(scmp_filter_ctx ctx, uint32_t def_action);

/**
 * Destroys the filter state and releases any resources
 * @param ctx the filter context
 *
 * This functions destroys the given seccomp filter state and releases any
 * resources, including memory, associated with the filter state.  This
 * function does not reset any seccomp filters already loaded into the kernel.
 * The filter context can no longer be used after calling this function.
 *
 */
void seccomp_release(scmp_filter_ctx ctx);

/**
 * Merge two filters
 * @param ctx_dst the destination filter context
 * @param ctx_src the source filter context
 *
 * This function merges two filter contexts into a single filter context and
 * destroys the second filter context.  The two filter contexts must have the
 * same attribute values and not contain any of the same architectures; if they
 * do, the merge operation will fail.  On success, the source filter context
 * will be destroyed and should no longer be used; it is not necessary to
 * call seccomp_release() on the source filter context.  Returns zero on
 * success, negative values on failure.
 *
 */
int seccomp_merge(scmp_filter_ctx ctx_dst, scmp_filter_ctx ctx_src);

/**
 * Resolve the architecture name to a architecture token
 * @param arch_name the architecture name
 *
 * This function resolves the given architecture name to a token suitable for
 * use with libseccomp, returns zero on failure.
 *
 */
uint32_t seccomp_arch_resolve_name(const char *arch_name);

/**
 * Return the native architecture token
 *
 * This function returns the native architecture token value, e.g. SCMP_ARCH_*.
 *
 */
uint32_t seccomp_arch_native(void);

/**
 * Check to see if an existing architecture is present in the filter
 * @param ctx the filter context
 * @param arch_token the architecture token, e.g. SCMP_ARCH_*
 *
 * This function tests to see if a given architecture is included in the filter
 * context.  If the architecture token is SCMP_ARCH_NATIVE then the native
 * architecture will be assumed.  Returns zero if the architecture exists in
 * the filter, -EEXIST if it is not present, and other negative values on
 * failure.
 *
 */
int seccomp_arch_exist(const scmp_filter_ctx ctx, uint32_t arch_token);

/**
 * Adds an architecture to the filter
 * @param ctx the filter context
 * @param arch_token the architecture token, e.g. SCMP_ARCH_*
 *
 * This function adds a new architecture to the given seccomp filter context.
 * Any new rules added after this function successfully returns will be added
 * to this architecture but existing rules will not be added to this
 * architecture.  If the architecture token is SCMP_ARCH_NATIVE then the native
 * architecture will be assumed.  Returns zero on success, -EEXIST if
 * specified architecture is already present, other negative values on failure.
 *
 */
int seccomp_arch_add(scmp_filter_ctx ctx, uint32_t arch_token);

/**
 * Removes an architecture from the filter
 * @param ctx the filter context
 * @param arch_token the architecture token, e.g. SCMP_ARCH_*
 *
 * This function removes an architecture from the given seccomp filter context.
 * If the architecture token is SCMP_ARCH_NATIVE then the native architecture
 * will be assumed.  Returns zero on success, negative values on failure.
 *
 */
int seccomp_arch_remove(scmp_filter_ctx ctx, uint32_t arch_token);

/**
 * Loads the filter into the kernel
 * @param ctx the filter context
 *
 * This function loads the given seccomp filter context into the kernel.  If
 * the filter was loaded correctly, the kernel will be enforcing the filter
 * when this function returns.  Returns zero on success, negative values on
 * error.
 *
 */
int seccomp_load(const scmp_filter_ctx ctx);

/**
 * Get the value of a filter attribute
 * @param ctx the filter context
 * @param attr the filter attribute name
 * @param value the filter attribute value
 *
 * This function fetches the value of the given attribute name and returns it
 * via @value.  Returns zero on success, negative values on failure.
 *
 */
int seccomp_attr_get(const scmp_filter_ctx ctx,
		     enum scmp_filter_attr attr, uint32_t *value);

/**
 * Set the value of a filter attribute
 * @param ctx the filter context
 * @param attr the filter attribute name
 * @param value the filter attribute value
 *
 * This function sets the value of the given attribute.  Returns zero on
 * success, negative values on failure.
 *
 */
int seccomp_attr_set(scmp_filter_ctx ctx,
		     enum scmp_filter_attr attr, uint32_t value);

/**
 * Resolve a syscall number to a name
 * @param arch_token the architecture token, e.g. SCMP_ARCH_*
 * @param num the syscall number
 *
 * Resolve the given syscall number to the syscall name for the given
 * architecture; it is up to the caller to free the returned string.  Returns
 * the syscall name on success, NULL on failure.
 *
 */
char *seccomp_syscall_resolve_num_arch(uint32_t arch_token, int num);

/**
 * Resolve a syscall name to a number
 * @param arch_token the architecture token, e.g. SCMP_ARCH_*
 * @param name the syscall name
 *
 * Resolve the given syscall name to the syscall number for the given
 * architecture.  Returns the syscall number on success, including negative
 * pseudo syscall numbers (e.g. __PNR_*); returns __NR_SCMP_ERROR on failure.
 *
 */
int seccomp_syscall_resolve_name_arch(uint32_t arch_token, const char *name);

/**
 * Resolve a syscall name to a number and perform any rewriting necessary
 * @param arch_token the architecture token, e.g. SCMP_ARCH_*
 * @param name the syscall name
 *
 * Resolve the given syscall name to the syscall number for the given
 * architecture and do any necessary syscall rewriting needed by the
 * architecture.  Returns the syscall number on success, including negative
 * pseudo syscall numbers (e.g. __PNR_*); returns __NR_SCMP_ERROR on failure.
 *
 */
int seccomp_syscall_resolve_name_rewrite(uint32_t arch_token, const char *name);

/**
 * Resolve a syscall name to a number
 * @param name the syscall name
 *
 * Resolve the given syscall name to the syscall number.  Returns the syscall
 * number on success, including negative pseudo syscall numbers (e.g. __PNR_*);
 * returns __NR_SCMP_ERROR on failure.
 *
 */
int seccomp_syscall_resolve_name(const char *name);

/**
 * Set the priority of a given syscall
 * @param ctx the filter context
 * @param syscall the syscall number
 * @param priority priority value, higher value == higher priority
 *
 * This function sets the priority of the given syscall; this value is used
 * when generating the seccomp filter code such that higher priority syscalls
 * will incur less filter code overhead than the lower priority syscalls in the
 * filter.  Returns zero on success, negative values on failure.
 *
 */
int seccomp_syscall_priority(scmp_filter_ctx ctx,
			     int syscall, uint8_t priority);

/**
 * Add a new rule to the filter
 * @param ctx the filter context
 * @param action the filter action
 * @param syscall the syscall number
 * @param arg_cnt the number of argument filters in the argument filter chain
 * @param ... scmp_arg_cmp structs (use of SCMP_ARG_CMP() recommended)
 *
 * This function adds a series of new argument/value checks to the seccomp
 * filter for the given syscall; multiple argument/value checks can be
 * specified and they will be chained together (AND'd together) in the filter.
 * If the specified rule needs to be adjusted due to architecture specifics it
 * will be adjusted without notification.  Returns zero on success, negative
 * values on failure.
 *
 */
int seccomp_rule_add(scmp_filter_ctx ctx,
		     uint32_t action, int syscall, unsigned int arg_cnt, ...);


/**
 * Add a new rule to the filter
 * @param ctx the filter context
 * @param action the filter action
 * @param syscall the syscall number
 * @param arg_cnt the number of elements in the arg_array parameter
 * @param arg_array array of scmp_arg_cmp structs
 *
 * This function adds a series of new argument/value checks to the seccomp
 * filter for the given syscall; multiple argument/value checks can be
 * specified and they will be chained together (AND'd together) in the filter.
 * If the specified rule needs to be adjusted due to architecture specifics it
 * will be adjusted without notification.  Returns zero on success, negative
 * values on failure.
 *
 */
int seccomp_rule_add_array(scmp_filter_ctx ctx,
			   uint32_t action, int syscall, unsigned int arg_cnt,
			   const struct scmp_arg_cmp *arg_array);

/**
 * Add a new rule to the filter
 * @param ctx the filter context
 * @param action the filter action
 * @param syscall the syscall number
 * @param arg_cnt the number of argument filters in the argument filter chain
 * @param ... scmp_arg_cmp structs (use of SCMP_ARG_CMP() recommended)
 *
 * This function adds a series of new argument/value checks to the seccomp
 * filter for the given syscall; multiple argument/value checks can be
 * specified and they will be chained together (AND'd together) in the filter.
 * If the specified rule can not be represented on the architecture the
 * function will fail.  Returns zero on success, negative values on failure.
 *
 */
int seccomp_rule_add_exact(scmp_filter_ctx ctx, uint32_t action,
			   int syscall, unsigned int arg_cnt, ...);

/**
 * Add a new rule to the filter
 * @param ctx the filter context
 * @param action the filter action
 * @param syscall the syscall number
 * @param arg_cnt  the number of elements in the arg_array parameter
 * @param arg_array array of scmp_arg_cmp structs
 *
 * This function adds a series of new argument/value checks to the seccomp
 * filter for the given syscall; multiple argument/value checks can be
 * specified and they will be chained together (AND'd together) in the filter.
 * If the specified rule can not be represented on the architecture the
 * function will fail.  Returns zero on success, negative values on failure.
 *
 */
int seccomp_rule_add_exact_array(scmp_filter_ctx ctx,
				 uint32_t action, int syscall,
				 unsigned int arg_cnt,
				 const struct scmp_arg_cmp *arg_array);

/**
 * Allocate a pair of notification request/response structures
 * @param req the request location
 * @param resp the response location
 *
 * This function allocates a pair of request/response structure by computing
 * the correct sized based on the currently running kernel. It returns zero on
 * success, and negative values on failure.
 *
 */
int seccomp_notify_alloc(struct seccomp_notif **req,
			 struct seccomp_notif_resp **resp);

/**
 * Free a pair of notification request/response structures.
 * @param req the request location
 * @param resp the response location
 */
void seccomp_notify_free(struct seccomp_notif *req,
			 struct seccomp_notif_resp *resp);

/**
 * Receive a notification from a seccomp notification fd
 * @param fd the notification fd
 * @param req the request buffer to save into
 *
 * Blocks waiting for a notification on this fd. This function is thread safe
 * (synchronization is performed in the kernel). Returns zero on success,
 * negative values on error.
 *
 */
int seccomp_notify_receive(int fd, struct seccomp_notif *req);

/**
 * Send a notification response to a seccomp notification fd
 * @param fd the notification fd
 * @param resp the response buffer to use
 *
 * Sends a notification response on this fd. This function is thread safe
 * (synchronization is performed in the kernel). Returns zero on success,
 * negative values on error.
 *
 */
int seccomp_notify_respond(int fd, struct seccomp_notif_resp *resp);

/**
 * Check if a notification id is still valid
 * @param fd the notification fd
 * @param id the id to test
 *
 * Checks to see if a notification id is still valid. Returns 0 on success, and
 * negative values on failure.
 *
 */
int seccomp_notify_id_valid(int fd, uint64_t id);

/**
 * Return the notification fd from a filter that has already been loaded
 * @param ctx the filter context
 *
 * This returns the listener fd that was generated when the seccomp policy was
 * loaded. This is only valid after seccomp_load() with a filter that makes
 * use of SCMP_ACT_NOTIFY.
 *
 */
int seccomp_notify_fd(const scmp_filter_ctx ctx);

/**
 * Generate seccomp Pseudo Filter Code (PFC) and export it to a file
 * @param ctx the filter context
 * @param fd the destination fd
 *
 * This function generates seccomp Pseudo Filter Code (PFC) and writes it to
 * the given fd.  Returns zero on success, negative values on failure.
 *
 */
int seccomp_export_pfc(const scmp_filter_ctx ctx, int fd);

/**
 * Generate seccomp Berkley Packet Filter (BPF) code and export it to a file
 * @param ctx the filter context
 * @param fd the destination fd
 *
 * This function generates seccomp Berkley Packer Filter (BPF) code and writes
 * it to the given fd.  Returns zero on success, negative values on failure.
 *
 */
int seccomp_export_bpf(const scmp_filter_ctx ctx, int fd);

/*
 * pseudo syscall definitions
 */

/* NOTE - pseudo syscall values {-1..-99} are reserved */
#define __NR_SCMP_ERROR		-1
#define __NR_SCMP_UNDEF		-2

#include <seccomp-syscalls.h>

#ifdef __cplusplus
}
#endif

#endif
