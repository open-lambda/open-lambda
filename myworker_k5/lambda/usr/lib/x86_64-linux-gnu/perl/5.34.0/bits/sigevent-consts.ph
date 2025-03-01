require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_BITS_SIGEVENT_CONSTS_H)) {
    eval 'sub _BITS_SIGEVENT_CONSTS_H () {1;}' unless defined(&_BITS_SIGEVENT_CONSTS_H);
    if(!defined (&_SIGNAL_H)  && !defined (&_AIO_H)) {
	die("Don't include <bits/sigevent-consts.h> directly; use <signal.h> instead.");
    }
    eval("sub SIGEV_SIGNAL () { 0; }") unless defined(&SIGEV_SIGNAL);
    eval("sub SIGEV_NONE () { 1; }") unless defined(&SIGEV_NONE);
    eval("sub SIGEV_THREAD () { 2; }") unless defined(&SIGEV_THREAD);
    eval("sub SIGEV_THREAD_ID () { 4; }") unless defined(&SIGEV_THREAD_ID);
}
1;
