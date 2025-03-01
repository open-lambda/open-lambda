require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_ASM_X86_POSIX_TYPES_X32_H)) {
    eval 'sub _ASM_X86_POSIX_TYPES_X32_H () {1;}' unless defined(&_ASM_X86_POSIX_TYPES_X32_H);
    eval 'sub __kernel_long_t () {\'__kernel_long_t\';}' unless defined(&__kernel_long_t);
    require 'asm/posix_types_64.ph';
}
1;
