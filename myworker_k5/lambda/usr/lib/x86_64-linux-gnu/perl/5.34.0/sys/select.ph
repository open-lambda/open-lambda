require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_SYS_SELECT_H)) {
    eval 'sub _SYS_SELECT_H () {1;}' unless defined(&_SYS_SELECT_H);
    require 'features.ph';
    require 'bits/types.ph';
    require 'bits/select.ph';
    require 'bits/types/sigset_t.ph';
    require 'bits/types/time_t.ph';
    require 'bits/types/struct_timeval.ph';
    if(defined(&__USE_XOPEN2K)) {
	require 'bits/types/struct_timespec.ph';
    }
    unless(defined(&__suseconds_t_defined)) {
	eval 'sub __suseconds_t_defined () {1;}' unless defined(&__suseconds_t_defined);
    }
    undef(&__NFDBITS) if defined(&__NFDBITS);
    eval 'sub __NFDBITS () {(8* $sizeof{ &__fd_mask});}' unless defined(&__NFDBITS);
    eval 'sub __FD_ELT {
        my($d) = @_;
	    eval q((($d) /  &__NFDBITS));
    }' unless defined(&__FD_ELT);
    eval 'sub __FD_MASK {
        my($d) = @_;
	    eval q((( &__fd_mask) (1 << (($d) %  &__NFDBITS))));
    }' unless defined(&__FD_MASK);
    if(defined(&__USE_XOPEN)) {
	eval 'sub __FDS_BITS {
	    my($set) = @_;
    	    eval q((($set)-> &fds_bits));
	}' unless defined(&__FDS_BITS);
    } else {
	eval 'sub __FDS_BITS {
	    my($set) = @_;
    	    eval q((($set)-> &__fds_bits));
	}' unless defined(&__FDS_BITS);
    }
    eval 'sub FD_SETSIZE () { &__FD_SETSIZE;}' unless defined(&FD_SETSIZE);
    if(defined(&__USE_MISC)) {
	eval 'sub NFDBITS () { &__NFDBITS;}' unless defined(&NFDBITS);
    }
    eval 'sub FD_SET {
        my($fd, $fdsetp) = @_;
	    eval q( &__FD_SET ($fd, $fdsetp));
    }' unless defined(&FD_SET);
    eval 'sub FD_CLR {
        my($fd, $fdsetp) = @_;
	    eval q( &__FD_CLR ($fd, $fdsetp));
    }' unless defined(&FD_CLR);
    eval 'sub FD_ISSET {
        my($fd, $fdsetp) = @_;
	    eval q( &__FD_ISSET ($fd, $fdsetp));
    }' unless defined(&FD_ISSET);
    eval 'sub FD_ZERO {
        my($fdsetp) = @_;
	    eval q( &__FD_ZERO ($fdsetp));
    }' unless defined(&FD_ZERO);
    unless(defined(&__USE_TIME_BITS64)) {
    } else {
	if(defined(&__REDIRECT)) {
	} else {
	    eval 'sub select () { &__select64;}' unless defined(&select);
	}
    }
    if(defined(&__USE_XOPEN2K)) {
	unless(defined(&__USE_TIME_BITS64)) {
	} else {
	    if(defined(&__REDIRECT)) {
	    } else {
		eval 'sub pselect () { &__pselect64;}' unless defined(&pselect);
	    }
	}
    }
    if((defined(&__USE_FORTIFY_LEVEL) ? &__USE_FORTIFY_LEVEL : undef) > 0 && defined (&__GNUC__)) {
	require 'bits/select2.ph';
    }
}
1;
