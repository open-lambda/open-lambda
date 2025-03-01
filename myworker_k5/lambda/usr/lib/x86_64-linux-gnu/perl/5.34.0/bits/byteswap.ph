require '_h2ph_pre.ph';

no warnings qw(redefine misc);

if(!defined (&_BYTESWAP_H)  && !defined (&_NETINET_IN_H)  && !defined (&_ENDIAN_H)) {
    die("Never use <bits/byteswap.h> directly; include <byteswap.h> instead.");
}
unless(defined(&_BITS_BYTESWAP_H)) {
    eval 'sub _BITS_BYTESWAP_H () {1;}' unless defined(&_BITS_BYTESWAP_H);
    require 'features.ph';
    require 'bits/types.ph';
    eval 'sub __bswap_constant_16 {
        my($x) = @_;
	    eval q((( &__uint16_t) (((($x) >> 8) & 0xff) | ((($x) & 0xff) << 8))));
    }' unless defined(&__bswap_constant_16);
# some #ifdef were dropped here -- fill in the blanks
    eval 'sub __bswap_16 {
        my($__bsx) = @_;
	    eval q({ });
    }' unless defined(&__bswap_16);
    eval 'sub __bswap_constant_32 {
        my($x) = @_;
	    eval q((((($x) & 0xff000000) >> 24) | ((($x) & 0xff0000) >> 8) | ((($x) & 0xff00) << 8) | ((($x) & 0xff) << 24)));
    }' unless defined(&__bswap_constant_32);
# some #ifdef were dropped here -- fill in the blanks
    eval 'sub __bswap_32 {
        my($__bsx) = @_;
	    eval q({ });
    }' unless defined(&__bswap_32);
    eval 'sub __bswap_constant_64 {
        my($x) = @_;
	    eval q((((($x) & 0xff00000000000000) >> 56) | ((($x) & 0xff000000000000) >> 40) | ((($x) & 0xff0000000000) >> 24) | ((($x) & 0xff00000000) >> 8) | ((($x) & 0xff000000) << 8) | ((($x) & 0xff0000) << 24) | ((($x) & 0xff00) << 40) | ((($x) & 0xff) << 56)));
    }' unless defined(&__bswap_constant_64);
# some #ifdef were dropped here -- fill in the blanks
    eval 'sub __bswap_64 {
        my($__bsx) = @_;
	    eval q({ });
    }' unless defined(&__bswap_64);
}
1;
