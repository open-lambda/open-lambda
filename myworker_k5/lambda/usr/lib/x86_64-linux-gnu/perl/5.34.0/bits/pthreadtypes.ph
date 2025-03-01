require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_BITS_PTHREADTYPES_COMMON_H)) {
    eval 'sub _BITS_PTHREADTYPES_COMMON_H () {1;}' unless defined(&_BITS_PTHREADTYPES_COMMON_H);
    require 'bits/thread-shared-types.ph';
    unless(defined(&__have_pthread_attr_t)) {
	eval 'sub __have_pthread_attr_t () {1;}' unless defined(&__have_pthread_attr_t);
    }
    if(defined (&__USE_UNIX98) || defined (&__USE_XOPEN2K)) {
    }
    if(defined(&__USE_XOPEN2K)) {
    }
}
1;
