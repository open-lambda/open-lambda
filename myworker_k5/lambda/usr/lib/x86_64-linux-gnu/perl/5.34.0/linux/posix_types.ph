require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_LINUX_POSIX_TYPES_H)) {
    eval 'sub _LINUX_POSIX_TYPES_H () {1;}' unless defined(&_LINUX_POSIX_TYPES_H);
    require 'linux/stddef.ph';
    undef(&__FD_SETSIZE) if defined(&__FD_SETSIZE);
    eval 'sub __FD_SETSIZE () {1024;}' unless defined(&__FD_SETSIZE);
    require 'asm/posix_types.ph';
}
1;
