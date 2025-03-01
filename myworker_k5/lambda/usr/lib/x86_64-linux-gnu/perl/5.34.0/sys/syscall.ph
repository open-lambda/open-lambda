require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_SYSCALL_H)) {
    eval 'sub _SYSCALL_H () {1;}' unless defined(&_SYSCALL_H);
    require 'asm/unistd.ph';
    require 'bits/syscall.ph';
}
1;
