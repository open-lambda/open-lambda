# This Makefile fragment (since dpkg 1.16.1) defines the following package
# information variables:
#
#   DEB_SOURCE: source package name.
#   DEB_VERSION: package's full version (epoch + upstream vers. + revision).
#   DEB_VERSION_EPOCH_UPSTREAM: package's version without the Debian revision.
#   DEB_VERSION_UPSTREAM_REVISION: package's version without the Debian epoch.
#   DEB_VERSION_UPSTREAM: package's upstream version.
#   DEB_DISTRIBUTION: distribution(s) listed in the current debian/changelog
#     entry.
#
#   SOURCE_DATE_EPOCH: source release date as seconds since the epoch, as
#     specified by <https://reproducible-builds.org/specs/source-date-epoch/>
#     (since dpkg 1.18.8).

dpkg_late_eval ?= $(or $(value DPKG_CACHE_$(1)),$(eval DPKG_CACHE_$(1) := $(shell $(2)))$(value DPKG_CACHE_$(1)))

DEB_SOURCE = $(call dpkg_late_eval,DEB_SOURCE,dpkg-parsechangelog -SSource)
DEB_VERSION = $(call dpkg_late_eval,DEB_VERSION,dpkg-parsechangelog -SVersion)
DEB_VERSION_EPOCH_UPSTREAM = $(call dpkg_late_eval,DEB_VERSION_EPOCH_UPSTREAM,echo '$(DEB_VERSION)' | sed -e 's/-[^-]*$$//')
DEB_VERSION_UPSTREAM_REVISION = $(call dpkg_late_eval,DEB_VERSION_UPSTREAM_REVISION,echo '$(DEB_VERSION)' | sed -e 's/^[0-9]*://')
DEB_VERSION_UPSTREAM = $(call dpkg_late_eval,DEB_VERSION_UPSTREAM,echo '$(DEB_VERSION_EPOCH_UPSTREAM)' | sed -e 's/^[0-9]*://')
DEB_DISTRIBUTION = $(call dpkg_late_eval,DEB_DISTRIBUTION,dpkg-parsechangelog -SDistribution)

SOURCE_DATE_EPOCH ?= $(call dpkg_late_eval,SOURCE_DATE_EPOCH,dpkg-parsechangelog -STimestamp)

export SOURCE_DATE_EPOCH
