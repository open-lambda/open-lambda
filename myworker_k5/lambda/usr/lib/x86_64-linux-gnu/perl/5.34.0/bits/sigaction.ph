require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_BITS_SIGACTION_H)) {
    eval 'sub _BITS_SIGACTION_H () {1;}' unless defined(&_BITS_SIGACTION_H);
    unless(defined(&_SIGNAL_H)) {
	die("Never include <bits/sigaction.h> directly; use <signal.h> instead.");
    }
    if(defined (&__USE_POSIX199309) || defined (&__USE_XOPEN_EXTENDED)) {
	eval 'sub sa_handler () { ($__sigaction_handler->{sa_handler});}' unless defined(&sa_handler);
	eval 'sub sa_sigaction () { ($__sigaction_handler->{sa_sigaction});}' unless defined(&sa_sigaction);
    } else {
    }
    eval 'sub SA_NOCLDSTOP () {1;}' unless defined(&SA_NOCLDSTOP);
    eval 'sub SA_NOCLDWAIT () {2;}' unless defined(&SA_NOCLDWAIT);
    eval 'sub SA_SIGINFO () {4;}' unless defined(&SA_SIGINFO);
    if(defined (&__USE_XOPEN_EXTENDED) || defined (&__USE_MISC)) {
	eval 'sub SA_ONSTACK () {0x8000000;}' unless defined(&SA_ONSTACK);
    }
    if(defined (&__USE_XOPEN_EXTENDED) || defined (&__USE_XOPEN2K8)) {
	eval 'sub SA_RESTART () {0x10000000;}' unless defined(&SA_RESTART);
	eval 'sub SA_NODEFER () {0x40000000;}' unless defined(&SA_NODEFER);
	eval 'sub SA_RESETHAND () {0x80000000;}' unless defined(&SA_RESETHAND);
    }
    if(defined(&__USE_MISC)) {
	eval 'sub SA_INTERRUPT () {0x20000000;}' unless defined(&SA_INTERRUPT);
	eval 'sub SA_NOMASK () { &SA_NODEFER;}' unless defined(&SA_NOMASK);
	eval 'sub SA_ONESHOT () { &SA_RESETHAND;}' unless defined(&SA_ONESHOT);
	eval 'sub SA_STACK () { &SA_ONSTACK;}' unless defined(&SA_STACK);
    }
    eval 'sub SIG_BLOCK () {0;}' unless defined(&SIG_BLOCK);
    eval 'sub SIG_UNBLOCK () {1;}' unless defined(&SIG_UNBLOCK);
    eval 'sub SIG_SETMASK () {2;}' unless defined(&SIG_SETMASK);
}
1;
