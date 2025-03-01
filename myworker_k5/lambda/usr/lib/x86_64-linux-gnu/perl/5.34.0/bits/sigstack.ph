require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_BITS_SIGSTACK_H)) {
    eval 'sub _BITS_SIGSTACK_H () {1;}' unless defined(&_BITS_SIGSTACK_H);
    if(!defined (&_SIGNAL_H)  && !defined (&_SYS_UCONTEXT_H)) {
	die("Never include this file directly.  Use <signal.h> instead");
    }
    eval 'sub MINSIGSTKSZ () {2048;}' unless defined(&MINSIGSTKSZ);
    eval 'sub SIGSTKSZ () {8192;}' unless defined(&SIGSTKSZ);
}
1;
