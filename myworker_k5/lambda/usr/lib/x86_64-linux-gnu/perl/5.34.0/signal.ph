require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_SIGNAL_H)) {
    eval 'sub _SIGNAL_H () {1;}' unless defined(&_SIGNAL_H);
    require 'features.ph';
    require 'bits/types.ph';
    require 'bits/signum-generic.ph';
    require 'bits/types/sig_atomic_t.ph';
    if(defined (&__USE_POSIX)) {
	require 'bits/types/sigset_t.ph';
    }
    if(defined (&__USE_XOPEN) || defined (&__USE_XOPEN2K)) {
	unless(defined(&__pid_t_defined)) {
	    eval 'sub __pid_t_defined () {1;}' unless defined(&__pid_t_defined);
	}
	if(defined(&__USE_XOPEN)) {
	}
	unless(defined(&__uid_t_defined)) {
	    eval 'sub __uid_t_defined () {1;}' unless defined(&__uid_t_defined);
	}
    }
    if(defined(&__USE_POSIX199309)) {
	require 'bits/types/struct_timespec.ph';
    }
    if(defined (&__USE_POSIX199309) || defined (&__USE_XOPEN_EXTENDED)) {
	require 'bits/types/siginfo_t.ph';
	require 'bits/siginfo-consts.ph';
    }
    if(defined(&__USE_MISC)) {
	require 'bits/types/sigval_t.ph';
    }
    if(defined(&__USE_POSIX199309)) {
	require 'bits/types/sigevent_t.ph';
	require 'bits/sigevent-consts.ph';
    }
    if(defined(&__USE_GNU)) {
    }
    if(defined(&__USE_MISC)) {
    } else {
	if(defined(&__REDIRECT_NTH)) {
	} else {
	    eval 'sub signal () { &__sysv_signal;}' unless defined(&signal);
	}
    }
    if(defined (&__USE_XOPEN_EXTENDED)  && !defined (&__USE_XOPEN2K8)) {
    }
    if(defined(&__USE_POSIX)) {
    }
    if(defined (&__USE_MISC) || defined (&__USE_XOPEN_EXTENDED)) {
    }
    if(defined(&__USE_MISC)) {
    }
    if(defined(&__USE_XOPEN2K8)) {
    }
    if(defined(&__USE_XOPEN_EXTENDED)) {
	if(defined(&__GNUC__)) {
	} else {
	    eval 'sub sigpause {
	        my($sig) = @_;
    		eval q( &__sigpause (($sig), 1));
	    }' unless defined(&sigpause);
	}
    }
    if(defined(&__USE_MISC)) {
	eval 'sub sigmask {
	    my($sig) = @_;
    	    eval q( &__glibc_macro_warning (\\"sigmask is deprecated\\") ((1 << (($sig) - 1))));
	}' unless defined(&sigmask);
    }
    if(defined(&__USE_MISC)) {
	eval 'sub NSIG () { &_NSIG;}' unless defined(&NSIG);
    }
    if(defined(&__USE_GNU)) {
    }
    if(defined(&__USE_MISC)) {
    }
    if(defined(&__USE_POSIX)) {
	if(defined(&__USE_GNU)) {
	}
	require 'bits/sigaction.ph';
	if(defined(&__USE_POSIX199506)) {
	}
	if(defined(&__USE_POSIX199309)) {
	    unless(defined(&__USE_TIME_BITS64)) {
	    } else {
		if(defined(&__REDIRECT)) {
		} else {
		    eval 'sub sigtimedwait () { &__sigtimedwait64;}' unless defined(&sigtimedwait);
		}
	    }
	}
    }
    if(defined(&__USE_MISC)) {
	require 'bits/sigcontext.ph';
    }
    if(defined (&__USE_XOPEN_EXTENDED) || defined (&__USE_XOPEN2K8)) {
	eval 'sub __need_size_t () {1;}' unless defined(&__need_size_t);
	require 'stddef.ph';
	require 'bits/types/stack_t.ph';
	if(defined (&__USE_XOPEN) || defined (&__USE_XOPEN2K8)) {
	    require 'sys/ucontext.ph';
	}
    }
    if(defined (&__USE_XOPEN_EXTENDED) || defined (&__USE_MISC)) {
	require 'bits/sigstack.ph';
	require 'bits/sigstksz.ph';
	require 'bits/ss_flags.ph';
    }
    if(((defined (&__USE_XOPEN_EXTENDED)  && !defined (&__USE_XOPEN2K8)) || defined (&__USE_MISC))) {
	require 'bits/types/struct_sigstack.ph';
    }
    if(((defined (&__USE_XOPEN_EXTENDED)  && !defined (&__USE_XOPEN2K)) || defined (&__USE_MISC))) {
    }
    if(defined(&__USE_XOPEN_EXTENDED)) {
    }
    if(defined (&__USE_POSIX199506) || defined (&__USE_UNIX98)) {
	require 'bits/pthreadtypes.ph';
	require 'bits/sigthread.ph';
    }
    eval 'sub SIGRTMIN () {( &__libc_current_sigrtmin ());}' unless defined(&SIGRTMIN);
    eval 'sub SIGRTMAX () {( &__libc_current_sigrtmax ());}' unless defined(&SIGRTMAX);
    require 'bits/signal_ext.ph';
}
1;
