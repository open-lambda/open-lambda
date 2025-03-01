require '_h2ph_pre.ph';

no warnings qw(redefine misc);

if(defined (&__x86_64__)  && !defined (&__ILP32__)) {
    eval 'sub __WORDSIZE () {64;}' unless defined(&__WORDSIZE);
} else {
    eval 'sub __WORDSIZE () {32;}' unless defined(&__WORDSIZE);
    eval 'sub __WORDSIZE32_SIZE_ULONG () {0;}' unless defined(&__WORDSIZE32_SIZE_ULONG);
    eval 'sub __WORDSIZE32_PTRDIFF_LONG () {0;}' unless defined(&__WORDSIZE32_PTRDIFF_LONG);
}
if(defined(&__x86_64__)) {
    eval 'sub __WORDSIZE_TIME64_COMPAT32 () {1;}' unless defined(&__WORDSIZE_TIME64_COMPAT32);
    eval 'sub __SYSCALL_WORDSIZE () {64;}' unless defined(&__SYSCALL_WORDSIZE);
} else {
    eval 'sub __WORDSIZE_TIME64_COMPAT32 () {0;}' unless defined(&__WORDSIZE_TIME64_COMPAT32);
}
1;
