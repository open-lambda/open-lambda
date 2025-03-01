require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_BITS_SIGINFO_CONSTS_H)) {
    eval 'sub _BITS_SIGINFO_CONSTS_H () {1;}' unless defined(&_BITS_SIGINFO_CONSTS_H);
    unless(defined(&_SIGNAL_H)) {
	die("Don't include <bits/siginfo-consts.h> directly; use <signal.h> instead.");
    }
    require 'bits/siginfo-arch.ph';
    unless(defined(&__SI_ASYNCIO_AFTER_SIGIO)) {
	eval 'sub __SI_ASYNCIO_AFTER_SIGIO () {1;}' unless defined(&__SI_ASYNCIO_AFTER_SIGIO);
    }
    eval("sub SI_ASYNCNL () { -60; }") unless defined(&SI_ASYNCNL);
    eval("sub SI_DETHREAD () { -7; }") unless defined(&SI_DETHREAD);
    eval("sub SI_TKILL () { -6; }") unless defined(&SI_TKILL);
    eval("sub SI_SIGIO () { -5; }") unless defined(&SI_SIGIO);
    eval("sub SI_QUEUE () { -4; }") unless defined(&SI_QUEUE);
    eval("sub SI_USER () { -3; }") unless defined(&SI_USER);
    eval("sub SI_KERNEL () { 0x80; }") unless defined(&SI_KERNEL);
    if(defined (&__USE_XOPEN_EXTENDED) || defined (&__USE_XOPEN2K8)) {
	eval("sub ILL_ILLOPC () { 1; }") unless defined(&ILL_ILLOPC);
	eval("sub ILL_ILLOPN () { 2; }") unless defined(&ILL_ILLOPN);
	eval("sub ILL_ILLADR () { 3; }") unless defined(&ILL_ILLADR);
	eval("sub ILL_ILLTRP () { 4; }") unless defined(&ILL_ILLTRP);
	eval("sub ILL_PRVOPC () { 5; }") unless defined(&ILL_PRVOPC);
	eval("sub ILL_PRVREG () { 6; }") unless defined(&ILL_PRVREG);
	eval("sub ILL_COPROC () { 7; }") unless defined(&ILL_COPROC);
	eval("sub ILL_BADSTK () { 8; }") unless defined(&ILL_BADSTK);
	eval("sub ILL_BADIADDR () { 9; }") unless defined(&ILL_BADIADDR);
	eval("sub FPE_INTDIV () { 1; }") unless defined(&FPE_INTDIV);
	eval("sub FPE_INTOVF () { 2; }") unless defined(&FPE_INTOVF);
	eval("sub FPE_FLTDIV () { 3; }") unless defined(&FPE_FLTDIV);
	eval("sub FPE_FLTOVF () { 4; }") unless defined(&FPE_FLTOVF);
	eval("sub FPE_FLTUND () { 5; }") unless defined(&FPE_FLTUND);
	eval("sub FPE_FLTRES () { 6; }") unless defined(&FPE_FLTRES);
	eval("sub FPE_FLTINV () { 7; }") unless defined(&FPE_FLTINV);
	eval("sub FPE_FLTSUB () { 8; }") unless defined(&FPE_FLTSUB);
	eval("sub FPE_FLTUNK () { 14; }") unless defined(&FPE_FLTUNK);
	eval("sub FPE_CONDTRAP () { 15; }") unless defined(&FPE_CONDTRAP);
	eval("sub SEGV_MAPERR () { 1; }") unless defined(&SEGV_MAPERR);
	eval("sub SEGV_ACCERR () { 2; }") unless defined(&SEGV_ACCERR);
	eval("sub SEGV_BNDERR () { 3; }") unless defined(&SEGV_BNDERR);
	eval("sub SEGV_PKUERR () { 4; }") unless defined(&SEGV_PKUERR);
	eval("sub SEGV_ACCADI () { 5; }") unless defined(&SEGV_ACCADI);
	eval("sub SEGV_ADIDERR () { 6; }") unless defined(&SEGV_ADIDERR);
	eval("sub SEGV_ADIPERR () { 7; }") unless defined(&SEGV_ADIPERR);
	eval("sub SEGV_MTEAERR () { 8; }") unless defined(&SEGV_MTEAERR);
	eval("sub SEGV_MTESERR () { 9; }") unless defined(&SEGV_MTESERR);
	eval("sub BUS_ADRALN () { 1; }") unless defined(&BUS_ADRALN);
	eval("sub BUS_ADRERR () { 2; }") unless defined(&BUS_ADRERR);
	eval("sub BUS_OBJERR () { 3; }") unless defined(&BUS_OBJERR);
	eval("sub BUS_MCEERR_AR () { 4; }") unless defined(&BUS_MCEERR_AR);
	eval("sub BUS_MCEERR_AO () { 5; }") unless defined(&BUS_MCEERR_AO);
    }
    if(defined(&__USE_XOPEN_EXTENDED)) {
	eval("sub TRAP_BRKPT () { 1; }") unless defined(&TRAP_BRKPT);
	eval("sub TRAP_TRACE () { 2; }") unless defined(&TRAP_TRACE);
	eval("sub TRAP_BRANCH () { 3; }") unless defined(&TRAP_BRANCH);
	eval("sub TRAP_HWBKPT () { 4; }") unless defined(&TRAP_HWBKPT);
	eval("sub TRAP_UNK () { 5; }") unless defined(&TRAP_UNK);
    }
    if(defined (&__USE_XOPEN_EXTENDED) || defined (&__USE_XOPEN2K8)) {
	eval("sub CLD_EXITED () { 1; }") unless defined(&CLD_EXITED);
	eval("sub CLD_KILLED () { 2; }") unless defined(&CLD_KILLED);
	eval("sub CLD_DUMPED () { 3; }") unless defined(&CLD_DUMPED);
	eval("sub CLD_TRAPPED () { 4; }") unless defined(&CLD_TRAPPED);
	eval("sub CLD_STOPPED () { 5; }") unless defined(&CLD_STOPPED);
	eval("sub CLD_CONTINUED () { 6; }") unless defined(&CLD_CONTINUED);
	eval("sub POLL_IN () { 1; }") unless defined(&POLL_IN);
	eval("sub POLL_OUT () { 2; }") unless defined(&POLL_OUT);
	eval("sub POLL_MSG () { 3; }") unless defined(&POLL_MSG);
	eval("sub POLL_ERR () { 4; }") unless defined(&POLL_ERR);
	eval("sub POLL_PRI () { 5; }") unless defined(&POLL_PRI);
	eval("sub POLL_HUP () { 6; }") unless defined(&POLL_HUP);
    }
    if(defined(&__USE_GNU)) {
	require 'bits/siginfo-consts-arch.ph';
    }
}
1;
