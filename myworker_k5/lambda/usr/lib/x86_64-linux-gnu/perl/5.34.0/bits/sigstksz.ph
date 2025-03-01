require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_SIGNAL_H)) {
    die("Never include <bits/sigstksz.h> directly; use <signal.h> instead.");
}
if(defined (&__USE_DYNAMIC_STACK_SIZE)  && (defined(&__USE_DYNAMIC_STACK_SIZE) ? &__USE_DYNAMIC_STACK_SIZE : undef)) {
    require 'unistd.ph';
    undef(&SIGSTKSZ) if defined(&SIGSTKSZ);
    eval 'sub SIGSTKSZ () { &sysconf ( &_SC_SIGSTKSZ);}' unless defined(&SIGSTKSZ);
    undef(&MINSIGSTKSZ) if defined(&MINSIGSTKSZ);
    eval 'sub MINSIGSTKSZ () { &SIGSTKSZ;}' unless defined(&MINSIGSTKSZ);
}
1;
