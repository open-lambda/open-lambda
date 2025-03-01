require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_BITS_SIGNUM_ARCH_H)) {
    eval 'sub _BITS_SIGNUM_ARCH_H () {1;}' unless defined(&_BITS_SIGNUM_ARCH_H);
    unless(defined(&_SIGNAL_H)) {
	die("Never include <bits/signum-arch.h> directly; use <signal.h> instead.");
    }
    eval 'sub SIGSTKFLT () {16;}' unless defined(&SIGSTKFLT);
    eval 'sub SIGPWR () {30;}' unless defined(&SIGPWR);
    eval 'sub SIGBUS () {7;}' unless defined(&SIGBUS);
    eval 'sub SIGSYS () {31;}' unless defined(&SIGSYS);
    eval 'sub SIGURG () {23;}' unless defined(&SIGURG);
    eval 'sub SIGSTOP () {19;}' unless defined(&SIGSTOP);
    eval 'sub SIGTSTP () {20;}' unless defined(&SIGTSTP);
    eval 'sub SIGCONT () {18;}' unless defined(&SIGCONT);
    eval 'sub SIGCHLD () {17;}' unless defined(&SIGCHLD);
    eval 'sub SIGTTIN () {21;}' unless defined(&SIGTTIN);
    eval 'sub SIGTTOU () {22;}' unless defined(&SIGTTOU);
    eval 'sub SIGPOLL () {29;}' unless defined(&SIGPOLL);
    eval 'sub SIGXFSZ () {25;}' unless defined(&SIGXFSZ);
    eval 'sub SIGXCPU () {24;}' unless defined(&SIGXCPU);
    eval 'sub SIGVTALRM () {26;}' unless defined(&SIGVTALRM);
    eval 'sub SIGPROF () {27;}' unless defined(&SIGPROF);
    eval 'sub SIGUSR1 () {10;}' unless defined(&SIGUSR1);
    eval 'sub SIGUSR2 () {12;}' unless defined(&SIGUSR2);
    eval 'sub SIGWINCH () {28;}' unless defined(&SIGWINCH);
    eval 'sub SIGIO () { &SIGPOLL;}' unless defined(&SIGIO);
    eval 'sub SIGIOT () { &SIGABRT;}' unless defined(&SIGIOT);
    eval 'sub SIGCLD () { &SIGCHLD;}' unless defined(&SIGCLD);
    eval 'sub __SIGRTMIN () {32;}' unless defined(&__SIGRTMIN);
    eval 'sub __SIGRTMAX () {64;}' unless defined(&__SIGRTMAX);
}
1;
