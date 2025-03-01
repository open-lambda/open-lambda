require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_SYS_WAIT_H)) {
    eval 'sub _SYS_WAIT_H () {1;}' unless defined(&_SYS_WAIT_H);
    require 'features.ph';
    require 'bits/types.ph';
    unless(defined(&__pid_t_defined)) {
	eval 'sub __pid_t_defined () {1;}' unless defined(&__pid_t_defined);
    }
    if(defined (&__USE_XOPEN_EXTENDED) || defined (&__USE_XOPEN2K8)) {
	require 'signal.ph';
    }
    if(defined (&__USE_XOPEN_EXTENDED)  && !defined (&__USE_XOPEN2K8)) {
	require 'bits/types/struct_rusage.ph';
    }
    if(!defined (&_STDLIB_H) || (!defined (&__USE_XOPEN)  && !defined (&__USE_XOPEN2K8))) {
	require 'bits/waitflags.ph';
	require 'bits/waitstatus.ph';
	eval 'sub WEXITSTATUS {
	    my($status) = @_;
    	    eval q( &__WEXITSTATUS ($status));
	}' unless defined(&WEXITSTATUS);
	eval 'sub WTERMSIG {
	    my($status) = @_;
    	    eval q( &__WTERMSIG ($status));
	}' unless defined(&WTERMSIG);
	eval 'sub WSTOPSIG {
	    my($status) = @_;
    	    eval q( &__WSTOPSIG ($status));
	}' unless defined(&WSTOPSIG);
	eval 'sub WIFEXITED {
	    my($status) = @_;
    	    eval q( &__WIFEXITED ($status));
	}' unless defined(&WIFEXITED);
	eval 'sub WIFSIGNALED {
	    my($status) = @_;
    	    eval q( &__WIFSIGNALED ($status));
	}' unless defined(&WIFSIGNALED);
	eval 'sub WIFSTOPPED {
	    my($status) = @_;
    	    eval q( &__WIFSTOPPED ($status));
	}' unless defined(&WIFSTOPPED);
	if(defined(&__WIFCONTINUED)) {
	    eval 'sub WIFCONTINUED {
	        my($status) = @_;
    		eval q( &__WIFCONTINUED ($status));
	    }' unless defined(&WIFCONTINUED);
	}
    }
    if(defined(&__USE_MISC)) {
	eval 'sub WCOREFLAG () { &__WCOREFLAG;}' unless defined(&WCOREFLAG);
	eval 'sub WCOREDUMP {
	    my($status) = @_;
    	    eval q( &__WCOREDUMP ($status));
	}' unless defined(&WCOREDUMP);
	eval 'sub W_EXITCODE {
	    my($ret, $sig) = @_;
    	    eval q( &__W_EXITCODE ($ret, $sig));
	}' unless defined(&W_EXITCODE);
	eval 'sub W_STOPCODE {
	    my($sig) = @_;
    	    eval q( &__W_STOPCODE ($sig));
	}' unless defined(&W_STOPCODE);
    }
    if(defined (&__USE_XOPEN_EXTENDED) || defined (&__USE_XOPEN2K8)) {
	eval("sub P_ALL () { 0; }") unless defined(&P_ALL);
	eval("sub P_PID () { 1; }") unless defined(&P_PID);
	eval("sub P_PGID () { 2; }") unless defined(&P_PGID);
    }
    if(defined(&__USE_MISC)) {
	eval 'sub WAIT_ANY () {(-1);}' unless defined(&WAIT_ANY);
	eval 'sub WAIT_MYPGRP () {0;}' unless defined(&WAIT_MYPGRP);
    }
    if(defined (&__USE_XOPEN_EXTENDED) || defined (&__USE_XOPEN2K8)) {
	unless(defined(&__id_t_defined)) {
	    eval 'sub __id_t_defined () {1;}' unless defined(&__id_t_defined);
	}
	require 'bits/types/siginfo_t.ph';
    }
    if(defined (&__USE_MISC) || (defined (&__USE_XOPEN_EXTENDED)  && !defined (&__USE_XOPEN2K))) {
	unless(defined(&__USE_TIME_BITS64)) {
	} else {
	    if(defined(&__REDIRECT_NTHNL)) {
	    } else {
		eval 'sub wait3 () { &__wait3_time64;}' unless defined(&wait3);
	    }
	}
    }
    if(defined(&__USE_MISC)) {
	unless(defined(&__USE_TIME_BITS64)) {
	} else {
	    if(defined(&__REDIRECT_NTHNL)) {
	    } else {
		eval 'sub wait4 () { &__wait4_time64;}' unless defined(&wait4);
	    }
	}
    }
}
1;
