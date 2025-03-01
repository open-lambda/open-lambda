#!/bin/sh

if  test $# = 0							\
    && test x"$LDCONFIG_NOTRIGGER" = x				\
 && test x"$DPKG_MAINTSCRIPT_PACKAGE" != x			\
 && dpkg-trigger --check-supported 2>/dev/null
then
	if dpkg-trigger --no-await ldconfig; then
		if test x"$LDCONFIG_TRIGGER_DEBUG" != x; then
			echo "ldconfig: wrapper deferring update (trigger activated)"
		fi
		exit 0
	fi	
fi

exec /sbin/ldconfig.real "$@"
