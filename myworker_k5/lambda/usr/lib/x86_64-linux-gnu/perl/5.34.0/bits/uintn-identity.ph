require '_h2ph_pre.ph';

no warnings qw(redefine misc);

if(!defined (&_NETINET_IN_H)  && !defined (&_ENDIAN_H)) {
    die("Never use <bits/uintn-identity.h> directly; include <netinet/in.h> or <endian.h> instead.");
}
unless(defined(&_BITS_UINTN_IDENTITY_H)) {
    eval 'sub _BITS_UINTN_IDENTITY_H () {1;}' unless defined(&_BITS_UINTN_IDENTITY_H);
    require 'bits/types.ph';
    eval 'sub __uint32_identity {
        my($__x) = @_;
	    eval q({ $__x; });
    }' unless defined(&__uint32_identity);
    eval 'sub __uint64_identity {
        my($__x) = @_;
	    eval q({ $__x; });
    }' unless defined(&__uint64_identity);
}
1;
