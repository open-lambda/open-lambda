require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_BITS_SIGINFO_ARCH_H)) {
    eval 'sub _BITS_SIGINFO_ARCH_H () {1;}' unless defined(&_BITS_SIGINFO_ARCH_H);
    if(defined (&__x86_64__)  && (defined(&__WORDSIZE) ? &__WORDSIZE : undef) == 32) {
	eval 'sub __SI_ALIGNMENT () { &__attribute__ (( &__aligned__ (8)));}' unless defined(&__SI_ALIGNMENT);
	eval 'sub __SI_CLOCK_T () { &__sigchld_clock_t;}' unless defined(&__SI_CLOCK_T);
    }
}
1;
