# This Makefile fragment (since dpkg 1.16.1) defines the following
# vendor-related variables:
#
#   DEB_VENDOR: output of «dpkg-vendor --query Vendor».
#   DEB_PARENT_VENDOR: output of «dpkg-vendor --query Parent» (can be empty).
#
# This Makefile fragment also defines a set of "dpkg_vendor_derives_from"
# macros that can be used to verify if the current vendor derives from
# another vendor. The unversioned variant defaults to the v0 version if
# undefined, which can be defined explicitly to one of the versions or the
# versioned macros can be used directly. The following are example usages:
#
# - dpkg_vendor_derives_from (since dpkg 1.16.1)
#
#   ifeq ($(shell $(call dpkg_vendor_derives_from,ubuntu)),yes)
#     ...
#   endif
#
# - dpkg_vendor_derives_from_v0 (since dpkg 1.19.3)
#
#   ifeq ($(shell $(call dpkg_vendor_derives_from_v0,ubuntu)),yes)
#     ...
#   endif
#
# - dpkg_vendor_derives_from_v1 (since dpkg 1.19.3)
#
#   dpkg_vendor_derives_from = $(dpkg_vendor_derives_from_v1)
#   ifeq ($(call dpkg_vendor_derives_from,ubuntu),yes)
#     ...
#   endif
#   ifeq ($(call dpkg_vendor_derives_from_v1,ubuntu),yes)
#     ...
#   endif

dpkg_late_eval ?= $(or $(value DPKG_CACHE_$(1)),$(eval DPKG_CACHE_$(1) := $(shell $(2)))$(value DPKG_CACHE_$(1)))

DEB_VENDOR = $(call dpkg_late_eval,DEB_VENDOR,dpkg-vendor --query Vendor)
DEB_PARENT_VENDOR = $(call dpkg_late_eval,DEB_PARENT_VENDOR,dpkg-vendor --query Parent)

dpkg_vendor_derives_from_v0 = dpkg-vendor --derives-from $(1) && echo yes || echo no
dpkg_vendor_derives_from_v1 = $(shell $(dpkg_vendor_derives_from_v0))

dpkg_vendor_derives_from ?= $(dpkg_vendor_derives_from_v0)
