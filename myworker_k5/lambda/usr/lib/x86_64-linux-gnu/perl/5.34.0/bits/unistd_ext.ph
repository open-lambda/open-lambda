require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_UNISTD_H)) {
    die("Never include <bits/unistd_ext.h> directly; use <unistd.h> instead.");
}
if(defined(&__USE_GNU)) {
    if(defined(&__has_include)) {
	if( &__has_include ("linux/close_range.h")) {
	    require 'linux/close_range.ph';
	}
    }
    unless(defined(&CLOSE_RANGE_UNSHARE)) {
	eval 'sub CLOSE_RANGE_UNSHARE () {(1 << 1);}' unless defined(&CLOSE_RANGE_UNSHARE);
    }
    unless(defined(&CLOSE_RANGE_CLOEXEC)) {
	eval 'sub CLOSE_RANGE_CLOEXEC () {(1 << 2);}' unless defined(&CLOSE_RANGE_CLOEXEC);
    }
}
1;
