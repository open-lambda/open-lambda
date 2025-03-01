require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_THREAD_SHARED_TYPES_H)) {
    eval 'sub _THREAD_SHARED_TYPES_H () {1;}' unless defined(&_THREAD_SHARED_TYPES_H);
    require 'bits/pthreadtypes-arch.ph';
    require 'bits/atomic_wide_counter.ph';
    require 'bits/struct_mutex.ph';
    require 'bits/struct_rwlock.ph';
    eval 'sub __ONCE_FLAG_INIT () {{ 0};}' unless defined(&__ONCE_FLAG_INIT);
}
1;
