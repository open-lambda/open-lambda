/*
 * Copyright © 2014-2018 Broadcom
 *
 * Permission is hereby granted, free of charge, to any person obtaining a
 * copy of this software and associated documentation files (the "Software"),
 * to deal in the Software without restriction, including without limitation
 * the rights to use, copy, modify, merge, publish, distribute, sublicense,
 * and/or sell copies of the Software, and to permit persons to whom the
 * Software is furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice (including the next
 * paragraph) shall be included in all copies or substantial portions of the
 * Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.  IN NO EVENT SHALL
 * THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
 * FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS
 * IN THE SOFTWARE.
 */

#ifndef _V3D_DRM_H_
#define _V3D_DRM_H_

#include "drm.h"

#if defined(__cplusplus)
extern "C" {
#endif

#define DRM_V3D_SUBMIT_CL                         0x00
#define DRM_V3D_WAIT_BO                           0x01
#define DRM_V3D_CREATE_BO                         0x02
#define DRM_V3D_MMAP_BO                           0x03
#define DRM_V3D_GET_PARAM                         0x04
#define DRM_V3D_GET_BO_OFFSET                     0x05
#define DRM_V3D_SUBMIT_TFU                        0x06
#define DRM_V3D_SUBMIT_CSD                        0x07
#define DRM_V3D_PERFMON_CREATE                    0x08
#define DRM_V3D_PERFMON_DESTROY                   0x09
#define DRM_V3D_PERFMON_GET_VALUES                0x0a

#define DRM_IOCTL_V3D_SUBMIT_CL           DRM_IOWR(DRM_COMMAND_BASE + DRM_V3D_SUBMIT_CL, struct drm_v3d_submit_cl)
#define DRM_IOCTL_V3D_WAIT_BO             DRM_IOWR(DRM_COMMAND_BASE + DRM_V3D_WAIT_BO, struct drm_v3d_wait_bo)
#define DRM_IOCTL_V3D_CREATE_BO           DRM_IOWR(DRM_COMMAND_BASE + DRM_V3D_CREATE_BO, struct drm_v3d_create_bo)
#define DRM_IOCTL_V3D_MMAP_BO             DRM_IOWR(DRM_COMMAND_BASE + DRM_V3D_MMAP_BO, struct drm_v3d_mmap_bo)
#define DRM_IOCTL_V3D_GET_PARAM           DRM_IOWR(DRM_COMMAND_BASE + DRM_V3D_GET_PARAM, struct drm_v3d_get_param)
#define DRM_IOCTL_V3D_GET_BO_OFFSET       DRM_IOWR(DRM_COMMAND_BASE + DRM_V3D_GET_BO_OFFSET, struct drm_v3d_get_bo_offset)
#define DRM_IOCTL_V3D_SUBMIT_TFU          DRM_IOW(DRM_COMMAND_BASE + DRM_V3D_SUBMIT_TFU, struct drm_v3d_submit_tfu)
#define DRM_IOCTL_V3D_SUBMIT_CSD          DRM_IOW(DRM_COMMAND_BASE + DRM_V3D_SUBMIT_CSD, struct drm_v3d_submit_csd)
#define DRM_IOCTL_V3D_PERFMON_CREATE      DRM_IOWR(DRM_COMMAND_BASE + DRM_V3D_PERFMON_CREATE, \
						   struct drm_v3d_perfmon_create)
#define DRM_IOCTL_V3D_PERFMON_DESTROY     DRM_IOWR(DRM_COMMAND_BASE + DRM_V3D_PERFMON_DESTROY, \
						   struct drm_v3d_perfmon_destroy)
#define DRM_IOCTL_V3D_PERFMON_GET_VALUES  DRM_IOWR(DRM_COMMAND_BASE + DRM_V3D_PERFMON_GET_VALUES, \
						   struct drm_v3d_perfmon_get_values)

#define DRM_V3D_SUBMIT_CL_FLUSH_CACHE             0x01

/**
 * struct drm_v3d_submit_cl - ioctl argument for submitting commands to the 3D
 * engine.
 *
 * This asks the kernel to have the GPU execute an optional binner
 * command list, and a render command list.
 *
 * The L1T, slice, L2C, L2T, and GCA caches will be flushed before
 * each CL executes.  The VCD cache should be flushed (if necessary)
 * by the submitted CLs.  The TLB writes are guaranteed to have been
 * flushed by the time the render done IRQ happens, which is the
 * trigger for out_sync.  Any dirtying of cachelines by the job (only
 * possible using TMU writes) must be flushed by the caller using the
 * DRM_V3D_SUBMIT_CL_FLUSH_CACHE_FLAG flag.
 */
struct drm_v3d_submit_cl {
	/* Pointer to the binner command list.
	 *
	 * This is the first set of commands executed, which runs the
	 * coordinate shader to determine where primitives land on the screen,
	 * then writes out the state updates and draw calls necessary per tile
	 * to the tile allocation BO.
	 *
	 * This BCL will block on any previous BCL submitted on the
	 * same FD, but not on any RCL or BCLs submitted by other
	 * clients -- that is left up to the submitter to control
	 * using in_sync_bcl if necessary.
	 */
	__u32 bcl_start;

	/** End address of the BCL (first byte after the BCL) */
	__u32 bcl_end;

	/* Offset of the render command list.
	 *
	 * This is the second set of commands executed, which will either
	 * execute the tiles that have been set up by the BCL, or a fixed set
	 * of tiles (in the case of RCL-only blits).
	 *
	 * This RCL will block on this submit's BCL, and any previous
	 * RCL submitted on the same FD, but not on any RCL or BCLs
	 * submitted by other clients -- that is left up to the
	 * submitter to control using in_sync_rcl if necessary.
	 */
	__u32 rcl_start;

	/** End address of the RCL (first byte after the RCL) */
	__u32 rcl_end;

	/** An optional sync object to wait on before starting the BCL. */
	__u32 in_sync_bcl;
	/** An optional sync object to wait on before starting the RCL. */
	__u32 in_sync_rcl;
	/** An optional sync object to place the completion fence in. */
	__u32 out_sync;

	/* Offset of the tile alloc memory
	 *
	 * This is optional on V3D 3.3 (where the CL can set the value) but
	 * required on V3D 4.1.
	 */
	__u32 qma;

	/** Size of the tile alloc memory. */
	__u32 qms;

	/** Offset of the tile state data array. */
	__u32 qts;

	/* Pointer to a u32 array of the BOs that are referenced by the job.
	 */
	__u64 bo_handles;

	/* Number of BO handles passed in (size is that times 4). */
	__u32 bo_handle_count;

	__u32 flags;

	/* ID of the perfmon to attach to this job. 0 means no perfmon. */
	__u32 perfmon_id;

	__u32 pad;
};

/**
 * struct drm_v3d_wait_bo - ioctl argument for waiting for
 * completion of the last DRM_V3D_SUBMIT_CL on a BO.
 *
 * This is useful for cases where multiple processes might be
 * rendering to a BO and you want to wait for all rendering to be
 * completed.
 */
struct drm_v3d_wait_bo {
	__u32 handle;
	__u32 pad;
	__u64 timeout_ns;
};

/**
 * struct drm_v3d_create_bo - ioctl argument for creating V3D BOs.
 *
 * There are currently no values for the flags argument, but it may be
 * used in a future extension.
 */
struct drm_v3d_create_bo {
	__u32 size;
	__u32 flags;
	/** Returned GEM handle for the BO. */
	__u32 handle;
	/**
	 * Returned offset for the BO in the V3D address space.  This offset
	 * is private to the DRM fd and is valid for the lifetime of the GEM
	 * handle.
	 *
	 * This offset value will always be nonzero, since various HW
	 * units treat 0 specially.
	 */
	__u32 offset;
};

/**
 * struct drm_v3d_mmap_bo - ioctl argument for mapping V3D BOs.
 *
 * This doesn't actually perform an mmap.  Instead, it returns the
 * offset you need to use in an mmap on the DRM device node.  This
 * means that tools like valgrind end up knowing about the mapped
 * memory.
 *
 * There are currently no values for the flags argument, but it may be
 * used in a future extension.
 */
struct drm_v3d_mmap_bo {
	/** Handle for the object being mapped. */
	__u32 handle;
	__u32 flags;
	/** offset into the drm node to use for subsequent mmap call. */
	__u64 offset;
};

enum drm_v3d_param {
	DRM_V3D_PARAM_V3D_UIFCFG,
	DRM_V3D_PARAM_V3D_HUB_IDENT1,
	DRM_V3D_PARAM_V3D_HUB_IDENT2,
	DRM_V3D_PARAM_V3D_HUB_IDENT3,
	DRM_V3D_PARAM_V3D_CORE0_IDENT0,
	DRM_V3D_PARAM_V3D_CORE0_IDENT1,
	DRM_V3D_PARAM_V3D_CORE0_IDENT2,
	DRM_V3D_PARAM_SUPPORTS_TFU,
	DRM_V3D_PARAM_SUPPORTS_CSD,
	DRM_V3D_PARAM_SUPPORTS_CACHE_FLUSH,
	DRM_V3D_PARAM_SUPPORTS_PERFMON,
};

struct drm_v3d_get_param {
	__u32 param;
	__u32 pad;
	__u64 value;
};

/**
 * Returns the offset for the BO in the V3D address space for this DRM fd.
 * This is the same value returned by drm_v3d_create_bo, if that was called
 * from this DRM fd.
 */
struct drm_v3d_get_bo_offset {
	__u32 handle;
	__u32 offset;
};

struct drm_v3d_submit_tfu {
	__u32 icfg;
	__u32 iia;
	__u32 iis;
	__u32 ica;
	__u32 iua;
	__u32 ioa;
	__u32 ios;
	__u32 coef[4];
	/* First handle is the output BO, following are other inputs.
	 * 0 for unused.
	 */
	__u32 bo_handles[4];
	/* sync object to block on before running the TFU job.  Each TFU
	 * job will execute in the order submitted to its FD.  Synchronization
	 * against rendering jobs requires using sync objects.
	 */
	__u32 in_sync;
	/* Sync object to signal when the TFU job is done. */
	__u32 out_sync;
};

/* Submits a compute shader for dispatch.  This job will block on any
 * previous compute shaders submitted on this fd, and any other
 * synchronization must be performed with in_sync/out_sync.
 */
struct drm_v3d_submit_csd {
	__u32 cfg[7];
	__u32 coef[4];

	/* Pointer to a u32 array of the BOs that are referenced by the job.
	 */
	__u64 bo_handles;

	/* Number of BO handles passed in (size is that times 4). */
	__u32 bo_handle_count;

	/* sync object to block on before running the CSD job.  Each
	 * CSD job will execute in the order submitted to its FD.
	 * Synchronization against rendering/TFU jobs or CSD from
	 * other fds requires using sync objects.
	 */
	__u32 in_sync;
	/* Sync object to signal when the CSD job is done. */
	__u32 out_sync;

	/* ID of the perfmon to attach to this job. 0 means no perfmon. */
	__u32 perfmon_id;
};

enum {
	V3D_PERFCNT_FEP_VALID_PRIMTS_NO_PIXELS,
	V3D_PERFCNT_FEP_VALID_PRIMS,
	V3D_PERFCNT_FEP_EZ_NFCLIP_QUADS,
	V3D_PERFCNT_FEP_VALID_QUADS,
	V3D_PERFCNT_TLB_QUADS_STENCIL_FAIL,
	V3D_PERFCNT_TLB_QUADS_STENCILZ_FAIL,
	V3D_PERFCNT_TLB_QUADS_STENCILZ_PASS,
	V3D_PERFCNT_TLB_QUADS_ZERO_COV,
	V3D_PERFCNT_TLB_QUADS_NONZERO_COV,
	V3D_PERFCNT_TLB_QUADS_WRITTEN,
	V3D_PERFCNT_PTB_PRIM_VIEWPOINT_DISCARD,
	V3D_PERFCNT_PTB_PRIM_CLIP,
	V3D_PERFCNT_PTB_PRIM_REV,
	V3D_PERFCNT_QPU_IDLE_CYCLES,
	V3D_PERFCNT_QPU_ACTIVE_CYCLES_VERTEX_COORD_USER,
	V3D_PERFCNT_QPU_ACTIVE_CYCLES_FRAG,
	V3D_PERFCNT_QPU_CYCLES_VALID_INSTR,
	V3D_PERFCNT_QPU_CYCLES_TMU_STALL,
	V3D_PERFCNT_QPU_CYCLES_SCOREBOARD_STALL,
	V3D_PERFCNT_QPU_CYCLES_VARYINGS_STALL,
	V3D_PERFCNT_QPU_IC_HIT,
	V3D_PERFCNT_QPU_IC_MISS,
	V3D_PERFCNT_QPU_UC_HIT,
	V3D_PERFCNT_QPU_UC_MISS,
	V3D_PERFCNT_TMU_TCACHE_ACCESS,
	V3D_PERFCNT_TMU_TCACHE_MISS,
	V3D_PERFCNT_VPM_VDW_STALL,
	V3D_PERFCNT_VPM_VCD_STALL,
	V3D_PERFCNT_BIN_ACTIVE,
	V3D_PERFCNT_RDR_ACTIVE,
	V3D_PERFCNT_L2T_HITS,
	V3D_PERFCNT_L2T_MISSES,
	V3D_PERFCNT_CYCLE_COUNT,
	V3D_PERFCNT_QPU_CYCLES_STALLED_VERTEX_COORD_USER,
	V3D_PERFCNT_QPU_CYCLES_STALLED_FRAGMENT,
	V3D_PERFCNT_PTB_PRIMS_BINNED,
	V3D_PERFCNT_AXI_WRITES_WATCH_0,
	V3D_PERFCNT_AXI_READS_WATCH_0,
	V3D_PERFCNT_AXI_WRITE_STALLS_WATCH_0,
	V3D_PERFCNT_AXI_READ_STALLS_WATCH_0,
	V3D_PERFCNT_AXI_WRITE_BYTES_WATCH_0,
	V3D_PERFCNT_AXI_READ_BYTES_WATCH_0,
	V3D_PERFCNT_AXI_WRITES_WATCH_1,
	V3D_PERFCNT_AXI_READS_WATCH_1,
	V3D_PERFCNT_AXI_WRITE_STALLS_WATCH_1,
	V3D_PERFCNT_AXI_READ_STALLS_WATCH_1,
	V3D_PERFCNT_AXI_WRITE_BYTES_WATCH_1,
	V3D_PERFCNT_AXI_READ_BYTES_WATCH_1,
	V3D_PERFCNT_TLB_PARTIAL_QUADS,
	V3D_PERFCNT_TMU_CONFIG_ACCESSES,
	V3D_PERFCNT_L2T_NO_ID_STALL,
	V3D_PERFCNT_L2T_COM_QUE_STALL,
	V3D_PERFCNT_L2T_TMU_WRITES,
	V3D_PERFCNT_TMU_ACTIVE_CYCLES,
	V3D_PERFCNT_TMU_STALLED_CYCLES,
	V3D_PERFCNT_CLE_ACTIVE,
	V3D_PERFCNT_L2T_TMU_READS,
	V3D_PERFCNT_L2T_CLE_READS,
	V3D_PERFCNT_L2T_VCD_READS,
	V3D_PERFCNT_L2T_TMUCFG_READS,
	V3D_PERFCNT_L2T_SLC0_READS,
	V3D_PERFCNT_L2T_SLC1_READS,
	V3D_PERFCNT_L2T_SLC2_READS,
	V3D_PERFCNT_L2T_TMU_W_MISSES,
	V3D_PERFCNT_L2T_TMU_R_MISSES,
	V3D_PERFCNT_L2T_CLE_MISSES,
	V3D_PERFCNT_L2T_VCD_MISSES,
	V3D_PERFCNT_L2T_TMUCFG_MISSES,
	V3D_PERFCNT_L2T_SLC0_MISSES,
	V3D_PERFCNT_L2T_SLC1_MISSES,
	V3D_PERFCNT_L2T_SLC2_MISSES,
	V3D_PERFCNT_CORE_MEM_WRITES,
	V3D_PERFCNT_L2T_MEM_WRITES,
	V3D_PERFCNT_PTB_MEM_WRITES,
	V3D_PERFCNT_TLB_MEM_WRITES,
	V3D_PERFCNT_CORE_MEM_READS,
	V3D_PERFCNT_L2T_MEM_READS,
	V3D_PERFCNT_PTB_MEM_READS,
	V3D_PERFCNT_PSE_MEM_READS,
	V3D_PERFCNT_TLB_MEM_READS,
	V3D_PERFCNT_GMP_MEM_READS,
	V3D_PERFCNT_PTB_W_MEM_WORDS,
	V3D_PERFCNT_TLB_W_MEM_WORDS,
	V3D_PERFCNT_PSE_R_MEM_WORDS,
	V3D_PERFCNT_TLB_R_MEM_WORDS,
	V3D_PERFCNT_TMU_MRU_HITS,
	V3D_PERFCNT_COMPUTE_ACTIVE,
	V3D_PERFCNT_NUM,
};

#define DRM_V3D_MAX_PERF_COUNTERS                 32

struct drm_v3d_perfmon_create {
	__u32 id;
	__u32 ncounters;
	__u8 counters[DRM_V3D_MAX_PERF_COUNTERS];
};

struct drm_v3d_perfmon_destroy {
	__u32 id;
};

/*
 * Returns the values of the performance counters tracked by this
 * perfmon (as an array of ncounters u64 values).
 *
 * No implicit synchronization is performed, so the user has to
 * guarantee that any jobs using this perfmon have already been
 * completed  (probably by blocking on the seqno returned by the
 * last exec that used the perfmon).
 */
struct drm_v3d_perfmon_get_values {
	__u32 id;
	__u32 pad;
	__u64 values_ptr;
};

#if defined(__cplusplus)
}
#endif

#endif /* _V3D_DRM_H_ */
