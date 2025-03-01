require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_SYS_TIME_H)) {
    eval 'sub _SYS_TIME_H () {1;}' unless defined(&_SYS_TIME_H);
    require 'features.ph';
    require 'bits/types.ph';
    require 'bits/types/time_t.ph';
    require 'bits/types/struct_timeval.ph';
    unless(defined(&__suseconds_t_defined)) {
	eval 'sub __suseconds_t_defined () {1;}' unless defined(&__suseconds_t_defined);
    }
    require 'sys/select.ph';
    if(defined(&__USE_GNU)) {
	eval 'sub TIMEVAL_TO_TIMESPEC {
	    my($tv, $ts) = @_;
    	    eval q({ ($ts)-> &tv_sec = ($tv)-> &tv_sec; ($ts)-> &tv_nsec = ($tv)-> &tv_usec * 1000; });
	}' unless defined(&TIMEVAL_TO_TIMESPEC);
	eval 'sub TIMESPEC_TO_TIMEVAL {
	    my($tv, $ts) = @_;
    	    eval q({ ($tv)-> &tv_sec = ($ts)-> &tv_sec; ($tv)-> &tv_usec = ($ts)-> &tv_nsec / 1000; });
	}' unless defined(&TIMESPEC_TO_TIMEVAL);
    }
    if(defined(&__USE_MISC)) {
    }
    unless(defined(&__USE_TIME_BITS64)) {
    } else {
	if(defined(&__REDIRECT_NTH)) {
	} else {
	    eval 'sub gettimeofday () { &__gettimeofday64;}' unless defined(&gettimeofday);
	}
    }
    if(defined(&__USE_MISC)) {
	unless(defined(&__USE_TIME_BITS64)) {
	} else {
	    if(defined(&__REDIRECT_NTH)) {
	    } else {
		eval 'sub settimeofday () { &__settimeofday64;}' unless defined(&settimeofday);
		eval 'sub adjtime () { &__adjtime64;}' unless defined(&adjtime);
	    }
	}
    }
    eval("sub ITIMER_REAL () { 0; }") unless defined(&ITIMER_REAL);
    eval("sub ITIMER_VIRTUAL () { 1; }") unless defined(&ITIMER_VIRTUAL);
    eval("sub ITIMER_PROF () { 2; }") unless defined(&ITIMER_PROF);
    if(defined (&__USE_GNU)  && !defined (&__cplusplus)) {
    } else {
    }
    unless(defined(&__USE_TIME_BITS64)) {
    } else {
	if(defined(&__REDIRECT_NTH)) {
	} else {
	    eval 'sub getitimer () { &__getitimer64;}' unless defined(&getitimer);
	    eval 'sub setitimer () { &__setitimer64;}' unless defined(&setitimer);
	    eval 'sub utimes () { &__utimes64;}' unless defined(&utimes);
	}
    }
    if(defined(&__USE_MISC)) {
	unless(defined(&__USE_TIME_BITS64)) {
	} else {
	    if(defined(&__REDIRECT_NTH)) {
	    } else {
		eval 'sub lutimes () { &__lutimes64;}' unless defined(&lutimes);
		eval 'sub futimes () { &__futimes64;}' unless defined(&futimes);
	    }
	}
    }
    if(defined(&__USE_GNU)) {
	unless(defined(&__USE_TIME_BITS64)) {
	} else {
	    if(defined(&__REDIRECT_NTH)) {
	    } else {
		eval 'sub futimesat () { &__futimesat64;}' unless defined(&futimesat);
	    }
	}
    }
    if(defined(&__USE_MISC)) {
	eval 'sub timerisset {
	    my($tvp) = @_;
    	    eval q((($tvp)-> &tv_sec || ($tvp)-> &tv_usec));
	}' unless defined(&timerisset);
	eval 'sub timerclear {
	    my($tvp) = @_;
    	    eval q((($tvp)-> &tv_sec = ($tvp)-> &tv_usec = 0));
	}' unless defined(&timerclear);
	eval 'sub timercmp {
	    my($a, $b, $CMP) = @_;
    	    eval q(((($a)-> &tv_sec == ($b)-> &tv_sec) ? (($a)-> &tv_usec $CMP ($b)-> &tv_usec) : (($a)-> &tv_sec $CMP ($b)-> &tv_sec)));
	}' unless defined(&timercmp);
	eval 'sub timeradd {
	    my($a, $b, $result) = @_;
    	    eval q( &do { ($result)-> &tv_sec = ($a)-> &tv_sec + ($b)-> &tv_sec; ($result)-> &tv_usec = ($a)-> &tv_usec + ($b)-> &tv_usec;  &if (($result)-> &tv_usec >= 1000000) { ++($result)-> &tv_sec; ($result)-> &tv_usec -= 1000000; } }  &while (0));
	}' unless defined(&timeradd);
	eval 'sub timersub {
	    my($a, $b, $result) = @_;
    	    eval q( &do { ($result)-> &tv_sec = ($a)-> &tv_sec - ($b)-> &tv_sec; ($result)-> &tv_usec = ($a)-> &tv_usec - ($b)-> &tv_usec;  &if (($result)-> &tv_usec < 0) { --($result)-> &tv_sec; ($result)-> &tv_usec += 1000000; } }  &while (0));
	}' unless defined(&timersub);
    }
}
1;
