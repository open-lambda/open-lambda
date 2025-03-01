require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_UNISTD_H)) {
    die("Never include this file directly.  Use <unistd.h> instead");
}
require 'bits/wordsize.ph';
if((defined(&__WORDSIZE) ? &__WORDSIZE : undef) == 64) {
    eval 'sub _POSIX_V7_LPBIG_OFFBIG () {-1;}' unless defined(&_POSIX_V7_LPBIG_OFFBIG);
    eval 'sub _POSIX_V6_LPBIG_OFFBIG () {-1;}' unless defined(&_POSIX_V6_LPBIG_OFFBIG);
    eval 'sub _XBS5_LPBIG_OFFBIG () {-1;}' unless defined(&_XBS5_LPBIG_OFFBIG);
    eval 'sub _POSIX_V7_LP64_OFF64 () {1;}' unless defined(&_POSIX_V7_LP64_OFF64);
    eval 'sub _POSIX_V6_LP64_OFF64 () {1;}' unless defined(&_POSIX_V6_LP64_OFF64);
    eval 'sub _XBS5_LP64_OFF64 () {1;}' unless defined(&_XBS5_LP64_OFF64);
} else {
    eval 'sub _POSIX_V7_ILP32_OFFBIG () {1;}' unless defined(&_POSIX_V7_ILP32_OFFBIG);
    eval 'sub _POSIX_V6_ILP32_OFFBIG () {1;}' unless defined(&_POSIX_V6_ILP32_OFFBIG);
    eval 'sub _XBS5_ILP32_OFFBIG () {1;}' unless defined(&_XBS5_ILP32_OFFBIG);
    unless(defined(&__x86_64__)) {
	eval 'sub _POSIX_V7_ILP32_OFF32 () {1;}' unless defined(&_POSIX_V7_ILP32_OFF32);
	eval 'sub _POSIX_V6_ILP32_OFF32 () {1;}' unless defined(&_POSIX_V6_ILP32_OFF32);
	eval 'sub _XBS5_ILP32_OFF32 () {1;}' unless defined(&_XBS5_ILP32_OFF32);
    }
}
eval 'sub __ILP32_OFF32_CFLAGS () {"-m32";}' unless defined(&__ILP32_OFF32_CFLAGS);
eval 'sub __ILP32_OFF32_LDFLAGS () {"-m32";}' unless defined(&__ILP32_OFF32_LDFLAGS);
if(defined (&__x86_64__)  && defined (&__ILP32__)) {
    eval 'sub __ILP32_OFFBIG_CFLAGS () {"-mx32";}' unless defined(&__ILP32_OFFBIG_CFLAGS);
    eval 'sub __ILP32_OFFBIG_LDFLAGS () {"-mx32";}' unless defined(&__ILP32_OFFBIG_LDFLAGS);
} else {
    eval 'sub __ILP32_OFFBIG_CFLAGS () {"-m32 -D_LARGEFILE_SOURCE -D_FILE_OFFSET_BITS=64";}' unless defined(&__ILP32_OFFBIG_CFLAGS);
    eval 'sub __ILP32_OFFBIG_LDFLAGS () {"-m32";}' unless defined(&__ILP32_OFFBIG_LDFLAGS);
}
eval 'sub __LP64_OFF64_CFLAGS () {"-m64";}' unless defined(&__LP64_OFF64_CFLAGS);
eval 'sub __LP64_OFF64_LDFLAGS () {"-m64";}' unless defined(&__LP64_OFF64_LDFLAGS);
1;
