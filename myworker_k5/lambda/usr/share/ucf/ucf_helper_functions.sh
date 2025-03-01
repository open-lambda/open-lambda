#!/bin/sh

UCF="ucf --three-way --debconf-ok"

# rename ucf-conffile. This was mostly stolen from cacti.postinst after
# a short discussion on debian-mentors, see
# http://lists.debian.org/debian-mentors/2013/07/msg00027.html
# and the following thread. Thanks to Paul Gevers
rename_ucf_file() {
    local oldname
    local newname

    # override UCF_FORCE_CONFFNEW with an empty local variable
    local UCF_FORCE_CONFFNEW

    oldname="$1"
    newname="$2"
    if [ ! -e "$newname" ] ; then
        if [ -e "$oldname" ]; then
            mv "$oldname" "$newname"
        fi
        # ucf doesn't offer a documented way to do this, so we need to
        # peddle around with undocumented ucf internals.
        sed -i "s|$oldname|$newname|" /var/lib/ucf/hashfile
        ucfr --purge "$PKGNAME" "$oldname"
        ucfr "$PKGNAME" "$newname"
        # else: Don't do anything, leave old file in place
    fi
    ucfr "$PKGNAME" "$newname"
}

generate_directory_structure() {
    local pkgdir
    local locdir
    pkgdir="$1"
    locdir="$2"

    # generate empty directory structure

    (cd "$pkgdir" && find . -type d -print0 ) | \
      (cd "$locdir" && xargs -0 mkdir -p --)
}

# handle a single ucf_conffile by first checking whether the file might be
# accociated with a different package. If so, we keep our hands off the file
# so that a different package can safely hijack our conffiles.
# to hijack a file, simply ucfr it to a package before the ucf processing
# code.
# If the file is either unassociated or already associated with us, call ucf
# proper and register the file as ours.
handle_single_ucf_file()
{
    local pkgfile
    local locfile
    if [ -n "${UCF_HELPER_FUNCTIONS_DEBUG:-}" ]; then
    	set -x
    fi

    pkgfile="$1"
    locfile="$2"
    export DEBIAN_FRONTEND

    PKG="$(ucfq --with-colons "$locdir/$file" | head -n 1 | cut --delimiter=: --fields=2 )"
    # skip conffile if it is associated with a different package.
    # This allows other packages to safely hijack our conffiles.
    if [ -z "$PKG" ] || [ "$PKG" = "$PKGNAME" ]; then
        $UCF "$pkgfile" "$locdir/$file"
        ucfr "$PKGNAME" "$locdir/$file"
    fi
    set +x
}

# checks whether a file was deleted in the package and handle it on the local
# system appropriately: If the local file differs from what we had previously,
# we just unregister it and leave it on the system (preserving local changes),
# otherwise we remove it.
# this also removes conffiles that are zero-size after the
# ucf run, which might happen if the local admin has
# deleted a conffile that has changed in the package.
handle_deleted_ucf_file()
{
    local locfile
    local locdir
    local pkgdir
    if [ -n "${UCF_HELPER_FUNCTIONS_DEBUG:-}" ]; then
    	set -x
    fi
    locfile="$1"
    pkgdir="$2"
    locdir="$3"

    # compute the name of the reference file in $pkgdir
    reffile="$(echo "$locfile" | sed "s|$locdir|$pkgdir|")"
    if ! [ -e "$reffile" ]; then
        # if the reference file does not exist, then it was removed in the package
        # do as if the file was replaced with an empty file
        $UCF /dev/null "$locfile"
        if [ -s "$locfile" ]; then
            # the file has non-zero size after the ucf run. local admin must
            # have decided to keep the file with contents. Done here.
            :
        else
            # the file has zero size and can be removed
            # remove the file itself ('') and all possible backup/reference extensions
            for ext in '' '~' '%' .bak .dpkg-tmp .dpkg-new .dpkg-old .dpkg-dist .ucf-new .ucf-old .ucf-dist;  do
              rm -f "${locfile}$ext"
            done
        fi
        # unregister the file anyhow since the package doesn't know about it any more
        ucf --purge "${locfile}"
        ucfr --purge "$PKGNAME" "${locfile}"
    fi
    set +x
}

handle_all_ucf_files() {
    local pkgdir
    local locdir
    pkgdir="$1"
    locdir="$2"

    generate_directory_structure "$pkgdir" "$locdir"

    # handle regular ucf-conffiles by iterating through all conffiles
    # that come with the package
    for file in $(find "$pkgdir" -type f -printf '%P\n' ); do
        handle_single_ucf_file "$pkgdir/$file" "$locdir/$file"
    done

    # handle ucf-conffiles that were deleted in our package by iterating
    # through all ucf-conffiles that are registered for the package
    for locfile in $(ucfq --with-colons "$PKGNAME" | cut --delimiter=: --fields=1); do
        handle_deleted_ucf_file "$locfile" "$pkgdir" "$locdir"
    done
}

# vim:sw=4:sts=4:et:
