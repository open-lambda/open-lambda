require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_THREAD_MUTEX_INTERNAL_H)) {
    eval 'sub _THREAD_MUTEX_INTERNAL_H () {1;}' unless defined(&_THREAD_MUTEX_INTERNAL_H);
    if(defined(&__x86_64__)) {
    }
    if(defined(&__x86_64__)) {
	eval 'sub __PTHREAD_MUTEX_HAVE_PREV () {1;}' unless defined(&__PTHREAD_MUTEX_HAVE_PREV);
    } else {
	eval 'sub __spins () { ($__elision_data->{__espins});}' unless defined(&__spins);
	eval 'sub __elision () { ($__elision_data->{__eelision});}' unless defined(&__elision);
	eval 'sub __PTHREAD_MUTEX_HAVE_PREV () {0;}' unless defined(&__PTHREAD_MUTEX_HAVE_PREV);
    }
    if(defined(&__x86_64__)) {
	eval 'sub __PTHREAD_MUTEX_INITIALIZER {
	    my($__kind) = @_;
    	    eval q(0, 0, 0, 0, $__kind, 0, 0, { 0, 0});
	}' unless defined(&__PTHREAD_MUTEX_INITIALIZER);
    } else {
	eval 'sub __PTHREAD_MUTEX_INITIALIZER {
	    my($__kind) = @_;
    	    eval q(0, 0, 0, $__kind, 0, { { 0, 0} });
	}' unless defined(&__PTHREAD_MUTEX_INITIALIZER);
    }
}
1;
