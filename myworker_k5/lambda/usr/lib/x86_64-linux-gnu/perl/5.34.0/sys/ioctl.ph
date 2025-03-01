require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_SYS_IOCTL_H)) {
    eval 'sub _SYS_IOCTL_H () {1;}' unless defined(&_SYS_IOCTL_H);
    require 'features.ph';
    require 'bits/ioctls.ph';
    require 'bits/ioctl-types.ph';
    require 'sys/ttydefaults.ph';
    unless(defined(&__USE_TIME_BITS64)) {
    } else {
	if(defined(&__REDIRECT)) {
	} else {
	    eval 'sub ioctl () { &__ioctl_time64;}' unless defined(&ioctl);
	}
    }
}
1;
