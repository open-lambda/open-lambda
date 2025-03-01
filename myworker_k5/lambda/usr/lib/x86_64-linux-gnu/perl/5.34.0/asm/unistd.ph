require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_ASM_X86_UNISTD_H)) {
    eval 'sub _ASM_X86_UNISTD_H () {1;}' unless defined(&_ASM_X86_UNISTD_H);
    eval 'sub __X32_SYSCALL_BIT () {0x40000000;}' unless defined(&__X32_SYSCALL_BIT);
    if(defined(&__i386__)) {
	require 'asm/unistd_32.ph';
    }
 elsif(defined(&__ILP32__)) {
	require 'asm/unistd_x32.ph';
    } else {
	require 'asm/unistd_64.ph';
    }
}
1;
