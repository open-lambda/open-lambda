require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_BITS_SIGINFO_CONSTS_ARCH_H)) {
    eval 'sub _BITS_SIGINFO_CONSTS_ARCH_H () {1;}' unless defined(&_BITS_SIGINFO_CONSTS_ARCH_H);
}
1;
