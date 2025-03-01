require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_LINUX_IOCTL_H)) {
    eval 'sub _LINUX_IOCTL_H () {1;}' unless defined(&_LINUX_IOCTL_H);
    require 'asm/ioctl.ph';
}
1;
