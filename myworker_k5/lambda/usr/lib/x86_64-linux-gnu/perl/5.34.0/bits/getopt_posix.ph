require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_GETOPT_POSIX_H)) {
    eval 'sub _GETOPT_POSIX_H () {1;}' unless defined(&_GETOPT_POSIX_H);
    if(!defined (&_UNISTD_H)  && !defined (&_STDIO_H)) {
	die("Never include getopt_posix.h directly; use unistd.h instead.");
    }
    require 'bits/getopt_core.ph';
    if(defined (&__USE_POSIX2)  && !defined (&__USE_POSIX_IMPLICITLY)  && !defined (&__USE_GNU)  && !defined (&_GETOPT_H)) {
	if(defined(&__REDIRECT)) {
	} else {
	    eval 'sub getopt () { &__posix_getopt;}' unless defined(&getopt);
	}
    }
}
1;
