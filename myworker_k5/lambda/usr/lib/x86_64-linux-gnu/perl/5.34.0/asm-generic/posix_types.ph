require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&__ASM_GENERIC_POSIX_TYPES_H)) {
    eval 'sub __ASM_GENERIC_POSIX_TYPES_H () {1;}' unless defined(&__ASM_GENERIC_POSIX_TYPES_H);
    require 'asm/bitsperlong.ph';
    unless(defined(&__kernel_long_t)) {
    }
    unless(defined(&__kernel_ino_t)) {
    }
    unless(defined(&__kernel_mode_t)) {
    }
    unless(defined(&__kernel_pid_t)) {
    }
    unless(defined(&__kernel_ipc_pid_t)) {
    }
    unless(defined(&__kernel_uid_t)) {
    }
    unless(defined(&__kernel_suseconds_t)) {
    }
    unless(defined(&__kernel_daddr_t)) {
    }
    unless(defined(&__kernel_uid32_t)) {
    }
    unless(defined(&__kernel_old_uid_t)) {
    }
    unless(defined(&__kernel_old_dev_t)) {
    }
    unless(defined(&__kernel_size_t)) {
	if((defined(&__BITS_PER_LONG) ? &__BITS_PER_LONG : undef) != 64) {
	} else {
	}
    }
    unless(defined(&__kernel_fsid_t)) {
    }
}
1;
