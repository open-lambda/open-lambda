require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_BITS_SS_FLAGS_H)) {
    eval 'sub _BITS_SS_FLAGS_H () {1;}' unless defined(&_BITS_SS_FLAGS_H);
    if(!defined (&_SIGNAL_H)  && !defined (&_SYS_UCONTEXT_H)) {
	die("Never include this file directly.  Use <signal.h> instead");
    }
    eval("sub SS_ONSTACK () { 1; }") unless defined(&SS_ONSTACK);
    eval("sub SS_DISABLE () { 2; }") unless defined(&SS_DISABLE);
}
1;
