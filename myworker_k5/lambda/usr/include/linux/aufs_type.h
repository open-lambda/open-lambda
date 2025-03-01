/* SPDX-License-Identifier: GPL-2.0 WITH Linux-syscall-note */
/*
 * Copyright (C) 2005-2021 Junjiro R. Okajima
 *
 * This program, aufs is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; either version 2 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

#ifndef __AUFS_TYPE_H__
#define __AUFS_TYPE_H__

#define AUFS_NAME	"aufs"

#include <stdint.h>
#include <sys/types.h>
#include <limits.h>

#define AUFS_VERSION	"5.15.5-20211129"

/* todo? move this to linux-2.6.19/include/magic.h */
#define AUFS_SUPER_MAGIC	('a' << 24 | 'u' << 16 | 'f' << 8 | 's')

/* ---------------------------------------------------------------------- */


/* ---------------------------------------------------------------------- */

#define AUFS_FSTYPE		AUFS_NAME

#define AUFS_ROOT_INO		2
#define AUFS_FIRST_INO		11

#define AUFS_WH_PFX		".wh."
#define AUFS_WH_PFX_LEN		((int)sizeof(AUFS_WH_PFX) - 1)
#define AUFS_WH_TMP_LEN		4
/* a limit for rmdir/rename a dir and copyup */
#define AUFS_MAX_NAMELEN	(NAME_MAX \
				- AUFS_WH_PFX_LEN * 2	/* doubly whiteouted */\
				- 1			/* dot */\
				- AUFS_WH_TMP_LEN)	/* hex */
#define AUFS_XINO_FNAME		"." AUFS_NAME ".xino"
#define AUFS_XINO_DEFPATH	"/tmp/" AUFS_XINO_FNAME
#define AUFS_XINO_DEF_SEC	30 /* seconds */
#define AUFS_XINO_DEF_TRUNC	45 /* percentage */
#define AUFS_DIRWH_DEF		3
#define AUFS_RDCACHE_DEF	10 /* seconds */
#define AUFS_RDCACHE_MAX	3600 /* seconds */
#define AUFS_RDBLK_DEF		512 /* bytes */
#define AUFS_RDHASH_DEF		32
#define AUFS_WKQ_NAME		AUFS_NAME "d"
#define AUFS_MFS_DEF_SEC	30 /* seconds */
#define AUFS_MFS_MAX_SEC	3600 /* seconds */
#define AUFS_FHSM_CACHE_DEF_SEC	30 /* seconds */
#define AUFS_PLINK_WARN		50 /* number of plinks in a single bucket */

/* pseudo-link maintenace under /proc */
#define AUFS_PLINK_MAINT_NAME	"plink_maint"
#define AUFS_PLINK_MAINT_DIR	"fs/" AUFS_NAME
#define AUFS_PLINK_MAINT_PATH	AUFS_PLINK_MAINT_DIR "/" AUFS_PLINK_MAINT_NAME

/* dirren, renamed dir */
#define AUFS_DR_INFO_PFX	AUFS_WH_PFX ".dr."
#define AUFS_DR_BRHINO_NAME	AUFS_WH_PFX "hino"
/* whiteouted doubly */
#define AUFS_WH_DR_INFO_PFX	AUFS_WH_PFX AUFS_DR_INFO_PFX
#define AUFS_WH_DR_BRHINO	AUFS_WH_PFX AUFS_DR_BRHINO_NAME

#define AUFS_DIROPQ_NAME	AUFS_WH_PFX ".opq" /* whiteouted doubly */
#define AUFS_WH_DIROPQ		AUFS_WH_PFX AUFS_DIROPQ_NAME

#define AUFS_BASE_NAME		AUFS_WH_PFX AUFS_NAME
#define AUFS_PLINKDIR_NAME	AUFS_WH_PFX "plnk"
#define AUFS_ORPHDIR_NAME	AUFS_WH_PFX "orph"

/* doubly whiteouted */
#define AUFS_WH_BASE		AUFS_WH_PFX AUFS_BASE_NAME
#define AUFS_WH_PLINKDIR	AUFS_WH_PFX AUFS_PLINKDIR_NAME
#define AUFS_WH_ORPHDIR		AUFS_WH_PFX AUFS_ORPHDIR_NAME

/* branch permissions and attributes */
#define AUFS_BRPERM_RW		"rw"
#define AUFS_BRPERM_RO		"ro"
#define AUFS_BRPERM_RR		"rr"
#define AUFS_BRATTR_COO_REG	"coo_reg"
#define AUFS_BRATTR_COO_ALL	"coo_all"
#define AUFS_BRATTR_FHSM	"fhsm"
#define AUFS_BRATTR_UNPIN	"unpin"
#define AUFS_BRATTR_ICEX	"icex"
#define AUFS_BRATTR_ICEX_SEC	"icexsec"
#define AUFS_BRATTR_ICEX_SYS	"icexsys"
#define AUFS_BRATTR_ICEX_TR	"icextr"
#define AUFS_BRATTR_ICEX_USR	"icexusr"
#define AUFS_BRATTR_ICEX_OTH	"icexoth"
#define AUFS_BRRATTR_WH		"wh"
#define AUFS_BRWATTR_NLWH	"nolwh"
#define AUFS_BRWATTR_MOO	"moo"

#define AuBrPerm_RW		1		/* writable, hardlinkable wh */
#define AuBrPerm_RO		(1 << 1)	/* readonly */
#define AuBrPerm_RR		(1 << 2)	/* natively readonly */
#define AuBrPerm_Mask		(AuBrPerm_RW | AuBrPerm_RO | AuBrPerm_RR)

#define AuBrAttr_COO_REG	(1 << 3)	/* copy-up on open */
#define AuBrAttr_COO_ALL	(1 << 4)
#define AuBrAttr_COO_Mask	(AuBrAttr_COO_REG | AuBrAttr_COO_ALL)

#define AuBrAttr_FHSM		(1 << 5)	/* file-based hsm */
#define AuBrAttr_UNPIN		(1 << 6)	/* rename-able top dir of
						   branch. meaningless since
						   linux-3.18-rc1 */

/* ignore error in copying XATTR */
#define AuBrAttr_ICEX_SEC	(1 << 7)
#define AuBrAttr_ICEX_SYS	(1 << 8)
#define AuBrAttr_ICEX_TR	(1 << 9)
#define AuBrAttr_ICEX_USR	(1 << 10)
#define AuBrAttr_ICEX_OTH	(1 << 11)
#define AuBrAttr_ICEX		(AuBrAttr_ICEX_SEC	\
				 | AuBrAttr_ICEX_SYS	\
				 | AuBrAttr_ICEX_TR	\
				 | AuBrAttr_ICEX_USR	\
				 | AuBrAttr_ICEX_OTH)

#define AuBrRAttr_WH		(1 << 12)	/* whiteout-able */
#define AuBrRAttr_Mask		AuBrRAttr_WH

#define AuBrWAttr_NoLinkWH	(1 << 13)	/* un-hardlinkable whiteouts */
#define AuBrWAttr_MOO		(1 << 14)	/* move-up on open */
#define AuBrWAttr_Mask		(AuBrWAttr_NoLinkWH | AuBrWAttr_MOO)

#define AuBrAttr_CMOO_Mask	(AuBrAttr_COO_Mask | AuBrWAttr_MOO)

/* #warning test userspace */

/* the longest combination */
/* AUFS_BRATTR_ICEX and AUFS_BRATTR_ICEX_TR don't affect here */
#define AuBrPermStrSz	sizeof(AUFS_BRPERM_RW			\
			       "+" AUFS_BRATTR_COO_REG		\
			       "+" AUFS_BRATTR_FHSM		\
			       "+" AUFS_BRATTR_UNPIN		\
			       "+" AUFS_BRATTR_ICEX_SEC		\
			       "+" AUFS_BRATTR_ICEX_SYS		\
			       "+" AUFS_BRATTR_ICEX_USR		\
			       "+" AUFS_BRATTR_ICEX_OTH		\
			       "+" AUFS_BRWATTR_NLWH)

typedef struct {
	char a[AuBrPermStrSz];
} au_br_perm_str_t;

static __inline__ int au_br_writable(int brperm)
{
	return brperm & AuBrPerm_RW;
}

static __inline__ int au_br_whable(int brperm)
{
	return brperm & (AuBrPerm_RW | AuBrRAttr_WH);
}

static __inline__ int au_br_wh_linkable(int brperm)
{
	return !(brperm & AuBrWAttr_NoLinkWH);
}

static __inline__ int au_br_cmoo(int brperm)
{
	return brperm & AuBrAttr_CMOO_Mask;
}

static __inline__ int au_br_fhsm(int brperm)
{
	return brperm & AuBrAttr_FHSM;
}

/* ---------------------------------------------------------------------- */

/* ioctl */
enum {
	/* readdir in userspace */
	AuCtl_RDU,
	AuCtl_RDU_INO,

	AuCtl_WBR_FD,	/* pathconf wrapper */
	AuCtl_IBUSY,	/* busy inode */
	AuCtl_MVDOWN,	/* move-down */
	AuCtl_BR,	/* info about branches */
	AuCtl_FHSM_FD	/* connection for fhsm */
};

/* borrowed from linux/include/linux/kernel.h */
#ifndef ALIGN
#ifdef _GNU_SOURCE
#define ALIGN(x, a)		__ALIGN_MASK(x, (typeof(x))(a)-1)
#else
#define ALIGN(x, a)		(((x) + (a) - 1) & ~((a) - 1))
#endif
#define __ALIGN_MASK(x, mask)	(((x)+(mask))&~(mask))
#endif

/* borrowed from linux/include/linux/compiler-gcc3.h */
#ifndef __aligned
#define __aligned(x)			__attribute__((aligned(x)))
#endif


struct au_rdu_cookie {
	uint64_t	h_pos;
	int16_t		bindex;
	uint8_t		flags;
	uint8_t		pad;
	uint32_t	generation;
} __aligned(8);

struct au_rdu_ent {
	uint64_t	ino;
	int16_t		bindex;
	uint8_t		type;
	uint8_t		nlen;
	uint8_t		wh;
	char		name[];
} __aligned(8);

static __inline__ int au_rdu_len(int nlen)
{
	/* include the terminating NULL */
	return ALIGN(sizeof(struct au_rdu_ent) + nlen + 1,
		     sizeof(uint64_t));
}

union au_rdu_ent_ul {
	struct au_rdu_ent *e;
	uint64_t			ul;
};

enum {
	AufsCtlRduV_SZ,
	AufsCtlRduV_End
};

struct aufs_rdu {
	/* input */
	union {
		uint64_t	sz;	/* AuCtl_RDU */
		uint64_t	nent;	/* AuCtl_RDU_INO */
	};
	union au_rdu_ent_ul	ent;
	uint16_t		verify[AufsCtlRduV_End];

	/* input/output */
	uint32_t		blk;

	/* output */
	union au_rdu_ent_ul	tail;
	/* number of entries which were added in a single call */
	uint64_t		rent;
	uint8_t			full;
	uint8_t			shwh;

	struct au_rdu_cookie	cookie;
} __aligned(8);

/* ---------------------------------------------------------------------- */

/* dirren. the branch is identified by the filename who contains this */
struct au_drinfo {
	uint64_t ino;
	union {
		uint8_t oldnamelen;
		uint64_t _padding;
	};
	uint8_t oldname[];
} __aligned(8);

struct au_drinfo_fdata {
	uint32_t magic;
	struct au_drinfo drinfo;
} __aligned(8);

#define AUFS_DRINFO_MAGIC_V1	('a' << 24 | 'd' << 16 | 'r' << 8 | 0x01)
/* future */
#define AUFS_DRINFO_MAGIC_V2	('a' << 24 | 'd' << 16 | 'r' << 8 | 0x02)

/* ---------------------------------------------------------------------- */

struct aufs_wbr_fd {
	uint32_t	oflags;
	int16_t		brid;
} __aligned(8);

/* ---------------------------------------------------------------------- */

struct aufs_ibusy {
	uint64_t	ino, h_ino;
	int16_t		bindex;
} __aligned(8);

/* ---------------------------------------------------------------------- */

/* error code for move-down */
/* the actual message strings are implemented in aufs-util.git */
enum {
	EAU_MVDOWN_OPAQUE = 1,
	EAU_MVDOWN_WHITEOUT,
	EAU_MVDOWN_UPPER,
	EAU_MVDOWN_BOTTOM,
	EAU_MVDOWN_NOUPPER,
	EAU_MVDOWN_NOLOWERBR,
	EAU_Last
};

/* flags for move-down */
#define AUFS_MVDOWN_DMSG	1
#define AUFS_MVDOWN_OWLOWER	(1 << 1)	/* overwrite lower */
#define AUFS_MVDOWN_KUPPER	(1 << 2)	/* keep upper */
#define AUFS_MVDOWN_ROLOWER	(1 << 3)	/* do even if lower is RO */
#define AUFS_MVDOWN_ROLOWER_R	(1 << 4)	/* did on lower RO */
#define AUFS_MVDOWN_ROUPPER	(1 << 5)	/* do even if upper is RO */
#define AUFS_MVDOWN_ROUPPER_R	(1 << 6)	/* did on upper RO */
#define AUFS_MVDOWN_BRID_UPPER	(1 << 7)	/* upper brid */
#define AUFS_MVDOWN_BRID_LOWER	(1 << 8)	/* lower brid */
#define AUFS_MVDOWN_FHSM_LOWER	(1 << 9)	/* find fhsm attr for lower */
#define AUFS_MVDOWN_STFS	(1 << 10)	/* req. stfs */
#define AUFS_MVDOWN_STFS_FAILED	(1 << 11)	/* output: stfs is unusable */
#define AUFS_MVDOWN_BOTTOM	(1 << 12)	/* output: no more lowers */

/* index for move-down */
enum {
	AUFS_MVDOWN_UPPER,
	AUFS_MVDOWN_LOWER,
	AUFS_MVDOWN_NARRAY
};

/*
 * additional info of move-down
 * number of free blocks and inodes.
 * subset of struct kstatfs, but smaller and always 64bit.
 */
struct aufs_stfs {
	uint64_t	f_blocks;
	uint64_t	f_bavail;
	uint64_t	f_files;
	uint64_t	f_ffree;
};

struct aufs_stbr {
	int16_t			brid;	/* optional input */
	int16_t			bindex;	/* output */
	struct aufs_stfs	stfs;	/* output when AUFS_MVDOWN_STFS set */
} __aligned(8);

struct aufs_mvdown {
	uint32_t		flags;			/* input/output */
	struct aufs_stbr	stbr[AUFS_MVDOWN_NARRAY]; /* input/output */
	int8_t			au_errno;		/* output */
} __aligned(8);

/* ---------------------------------------------------------------------- */

union aufs_brinfo {
	/* PATH_MAX may differ between kernel-space and user-space */
	char	_spacer[4096];
	struct {
		int16_t	id;
		int	perm;
		char	path[];
	};
} __aligned(8);

/* ---------------------------------------------------------------------- */

#define AuCtlType		'A'
#define AUFS_CTL_RDU		_IOWR(AuCtlType, AuCtl_RDU, struct aufs_rdu)
#define AUFS_CTL_RDU_INO	_IOWR(AuCtlType, AuCtl_RDU_INO, struct aufs_rdu)
#define AUFS_CTL_WBR_FD		_IOW(AuCtlType, AuCtl_WBR_FD, \
				     struct aufs_wbr_fd)
#define AUFS_CTL_IBUSY		_IOWR(AuCtlType, AuCtl_IBUSY, struct aufs_ibusy)
#define AUFS_CTL_MVDOWN		_IOWR(AuCtlType, AuCtl_MVDOWN, \
				      struct aufs_mvdown)
#define AUFS_CTL_BRINFO		_IOW(AuCtlType, AuCtl_BR, union aufs_brinfo)
#define AUFS_CTL_FHSM_FD	_IOW(AuCtlType, AuCtl_FHSM_FD, int)

#endif /* __AUFS_TYPE_H__ */
