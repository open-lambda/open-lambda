require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&__sigevent_t_defined)) {
    eval 'sub __sigevent_t_defined () {1;}' unless defined(&__sigevent_t_defined);
    require 'bits/wordsize.ph';
    require 'bits/types.ph';
    require 'bits/types/__sigval_t.ph';
    eval 'sub __SIGEV_MAX_SIZE () {64;}' unless defined(&__SIGEV_MAX_SIZE);
    if((defined(&__WORDSIZE) ? &__WORDSIZE : undef) == 64) {
	eval 'sub __SIGEV_PAD_SIZE () {(( &__SIGEV_MAX_SIZE / $sizeof{\'int\'}) - 4);}' unless defined(&__SIGEV_PAD_SIZE);
    } else {
	eval 'sub __SIGEV_PAD_SIZE () {(( &__SIGEV_MAX_SIZE / $sizeof{\'int\'}) - 3);}' unless defined(&__SIGEV_PAD_SIZE);
    }
    unless(defined(&__have_pthread_attr_t)) {
	eval 'sub __have_pthread_attr_t () {1;}' unless defined(&__have_pthread_attr_t);
    }
    eval 'sub sigev_notify_function () { ($_sigev_un->{_sigev_thread}->{_function});}' unless defined(&sigev_notify_function);
    eval 'sub sigev_notify_attributes () { ($_sigev_un->{_sigev_thread}->{_attribute});}' unless defined(&sigev_notify_attributes);
}
1;
