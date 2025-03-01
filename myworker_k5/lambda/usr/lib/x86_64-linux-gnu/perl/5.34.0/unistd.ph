require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_UNISTD_H)) {
    eval 'sub _UNISTD_H () {1;}' unless defined(&_UNISTD_H);
    require 'features.ph';
    if(defined(&__USE_XOPEN2K8)) {
	eval 'sub _POSIX_VERSION () {200809;}' unless defined(&_POSIX_VERSION);
    }
 elsif(defined (&__USE_XOPEN2K)) {
	eval 'sub _POSIX_VERSION () {200112;}' unless defined(&_POSIX_VERSION);
    }
 elsif(defined (&__USE_POSIX199506)) {
	eval 'sub _POSIX_VERSION () {199506;}' unless defined(&_POSIX_VERSION);
    }
 elsif(defined (&__USE_POSIX199309)) {
	eval 'sub _POSIX_VERSION () {199309;}' unless defined(&_POSIX_VERSION);
    } else {
	eval 'sub _POSIX_VERSION () {199009;}' unless defined(&_POSIX_VERSION);
    }
    if(defined(&__USE_XOPEN2K8)) {
	eval 'sub __POSIX2_THIS_VERSION () {200809;}' unless defined(&__POSIX2_THIS_VERSION);
    }
 elsif(defined (&__USE_XOPEN2K)) {
	eval 'sub __POSIX2_THIS_VERSION () {200112;}' unless defined(&__POSIX2_THIS_VERSION);
    }
 elsif(defined (&__USE_POSIX199506)) {
	eval 'sub __POSIX2_THIS_VERSION () {199506;}' unless defined(&__POSIX2_THIS_VERSION);
    } else {
	eval 'sub __POSIX2_THIS_VERSION () {199209;}' unless defined(&__POSIX2_THIS_VERSION);
    }
    eval 'sub _POSIX2_VERSION () { &__POSIX2_THIS_VERSION;}' unless defined(&_POSIX2_VERSION);
    eval 'sub _POSIX2_C_VERSION () { &__POSIX2_THIS_VERSION;}' unless defined(&_POSIX2_C_VERSION);
    eval 'sub _POSIX2_C_BIND () { &__POSIX2_THIS_VERSION;}' unless defined(&_POSIX2_C_BIND);
    eval 'sub _POSIX2_C_DEV () { &__POSIX2_THIS_VERSION;}' unless defined(&_POSIX2_C_DEV);
    eval 'sub _POSIX2_SW_DEV () { &__POSIX2_THIS_VERSION;}' unless defined(&_POSIX2_SW_DEV);
    eval 'sub _POSIX2_LOCALEDEF () { &__POSIX2_THIS_VERSION;}' unless defined(&_POSIX2_LOCALEDEF);
    if(defined(&__USE_XOPEN2K8)) {
	eval 'sub _XOPEN_VERSION () {700;}' unless defined(&_XOPEN_VERSION);
    }
 elsif(defined (&__USE_XOPEN2K)) {
	eval 'sub _XOPEN_VERSION () {600;}' unless defined(&_XOPEN_VERSION);
    }
 elsif(defined (&__USE_UNIX98)) {
	eval 'sub _XOPEN_VERSION () {500;}' unless defined(&_XOPEN_VERSION);
    } else {
	eval 'sub _XOPEN_VERSION () {4;}' unless defined(&_XOPEN_VERSION);
    }
    eval 'sub _XOPEN_XCU_VERSION () {4;}' unless defined(&_XOPEN_XCU_VERSION);
    eval 'sub _XOPEN_XPG2 () {1;}' unless defined(&_XOPEN_XPG2);
    eval 'sub _XOPEN_XPG3 () {1;}' unless defined(&_XOPEN_XPG3);
    eval 'sub _XOPEN_XPG4 () {1;}' unless defined(&_XOPEN_XPG4);
    eval 'sub _XOPEN_UNIX () {1;}' unless defined(&_XOPEN_UNIX);
    eval 'sub _XOPEN_ENH_I18N () {1;}' unless defined(&_XOPEN_ENH_I18N);
    eval 'sub _XOPEN_LEGACY () {1;}' unless defined(&_XOPEN_LEGACY);
    require 'bits/posix_opt.ph';
    if(defined (&__USE_UNIX98) || defined (&__USE_XOPEN2K)) {
	require 'bits/environments.ph';
    }
    eval 'sub STDIN_FILENO () {0;}' unless defined(&STDIN_FILENO);
    eval 'sub STDOUT_FILENO () {1;}' unless defined(&STDOUT_FILENO);
    eval 'sub STDERR_FILENO () {2;}' unless defined(&STDERR_FILENO);
    require 'bits/types.ph';
    unless(defined(&__ssize_t_defined)) {
	eval 'sub __ssize_t_defined () {1;}' unless defined(&__ssize_t_defined);
    }
    eval 'sub __need_size_t () {1;}' unless defined(&__need_size_t);
    eval 'sub __need_NULL () {1;}' unless defined(&__need_NULL);
    require 'stddef.ph';
    if(defined (&__USE_XOPEN) || defined (&__USE_XOPEN2K)) {
	unless(defined(&__gid_t_defined)) {
	    eval 'sub __gid_t_defined () {1;}' unless defined(&__gid_t_defined);
	}
	unless(defined(&__uid_t_defined)) {
	    eval 'sub __uid_t_defined () {1;}' unless defined(&__uid_t_defined);
	}
	unless(defined(&__off_t_defined)) {
	    unless(defined(&__USE_FILE_OFFSET64)) {
	    } else {
	    }
	    eval 'sub __off_t_defined () {1;}' unless defined(&__off_t_defined);
	}
	if(defined (&__USE_LARGEFILE64)  && !defined (&__off64_t_defined)) {
	    eval 'sub __off64_t_defined () {1;}' unless defined(&__off64_t_defined);
	}
	unless(defined(&__useconds_t_defined)) {
	    eval 'sub __useconds_t_defined () {1;}' unless defined(&__useconds_t_defined);
	}
	unless(defined(&__pid_t_defined)) {
	    eval 'sub __pid_t_defined () {1;}' unless defined(&__pid_t_defined);
	}
    }
    if(defined (&__USE_XOPEN_EXTENDED) || defined (&__USE_XOPEN2K)) {
	unless(defined(&__intptr_t_defined)) {
	    eval 'sub __intptr_t_defined () {1;}' unless defined(&__intptr_t_defined);
	}
    }
    if(defined (&__USE_MISC) || defined (&__USE_XOPEN)) {
	unless(defined(&__socklen_t_defined)) {
	    eval 'sub __socklen_t_defined () {1;}' unless defined(&__socklen_t_defined);
	}
    }
    eval 'sub R_OK () {4;}' unless defined(&R_OK);
    eval 'sub W_OK () {2;}' unless defined(&W_OK);
    eval 'sub X_OK () {1;}' unless defined(&X_OK);
    eval 'sub F_OK () {0;}' unless defined(&F_OK);
    if(defined(&__USE_GNU)) {
    }
    if(defined(&__USE_ATFILE)) {
    }
    unless(defined(&_STDIO_H)) {
	eval 'sub SEEK_SET () {0;}' unless defined(&SEEK_SET);
	eval 'sub SEEK_CUR () {1;}' unless defined(&SEEK_CUR);
	eval 'sub SEEK_END () {2;}' unless defined(&SEEK_END);
	if(defined(&__USE_GNU)) {
	    eval 'sub SEEK_DATA () {3;}' unless defined(&SEEK_DATA);
	    eval 'sub SEEK_HOLE () {4;}' unless defined(&SEEK_HOLE);
	}
    }
    if(defined (&__USE_MISC)  && !defined (&L_SET)) {
	eval 'sub L_SET () { &SEEK_SET;}' unless defined(&L_SET);
	eval 'sub L_INCR () { &SEEK_CUR;}' unless defined(&L_INCR);
	eval 'sub L_XTND () { &SEEK_END;}' unless defined(&L_XTND);
    }
    unless(defined(&__USE_FILE_OFFSET64)) {
    } else {
	if(defined(&__REDIRECT_NTH)) {
	} else {
	    eval 'sub lseek () { &lseek64;}' unless defined(&lseek);
	}
    }
    if(defined(&__USE_LARGEFILE64)) {
    }
    if(defined(&__USE_MISC)) {
    }
    if(defined (&__USE_UNIX98) || defined (&__USE_XOPEN2K8)) {
	unless(defined(&__USE_FILE_OFFSET64)) {
	} else {
	    if(defined(&__REDIRECT)) {
	    } else {
		eval 'sub pread () { &pread64;}' unless defined(&pread);
		eval 'sub pwrite () { &pwrite64;}' unless defined(&pwrite);
	    }
	}
	if(defined(&__USE_LARGEFILE64)) {
	}
    }
    if(defined(&__USE_GNU)) {
    }
    if((defined (&__USE_XOPEN_EXTENDED)  && !defined (&__USE_XOPEN2K8)) || defined (&__USE_MISC)) {
    }
    if(defined (&__USE_XOPEN_EXTENDED) || defined (&__USE_XOPEN2K8)) {
    }
    if(defined(&__USE_ATFILE)) {
    }
    if(defined (&__USE_XOPEN_EXTENDED) || defined (&__USE_XOPEN2K8)) {
    }
    if(defined(&__USE_GNU)) {
    }
    if((defined (&__USE_XOPEN_EXTENDED)  && !defined (&__USE_XOPEN2K8)) || defined (&__USE_MISC)) {
    }
    if(defined(&__USE_GNU)) {
    }
    if(defined(&__USE_GNU)) {
    }
    if(defined(&__USE_XOPEN2K8)) {
    }
    if(defined(&__USE_GNU)) {
    }
    if(defined (&__USE_MISC) || defined (&__USE_XOPEN)) {
    }
    require 'bits/confname.ph';
    if(defined(&__USE_POSIX2)) {
    }
    if(defined (&__USE_XOPEN_EXTENDED) || defined (&__USE_XOPEN2K8)) {
    }
    if(defined (&__USE_MISC) || defined (&__USE_XOPEN_EXTENDED)) {
    }
    if(defined (&__USE_XOPEN_EXTENDED) || defined (&__USE_XOPEN2K8)) {
    }
    if(defined(&__USE_GNU)) {
    }
    if(defined (&__USE_MISC) || defined (&__USE_XOPEN_EXTENDED)) {
    }
    if(defined(&__USE_XOPEN2K)) {
    }
    if(defined (&__USE_MISC) || defined (&__USE_XOPEN_EXTENDED)) {
    }
    if(defined(&__USE_XOPEN2K)) {
    }
    if(defined(&__USE_GNU)) {
    }
    if((defined (&__USE_XOPEN_EXTENDED)  && !defined (&__USE_XOPEN2K8)) || defined (&__USE_MISC)) {
    }
    if(defined(&__USE_GNU)) {
    }
    if(defined(&__USE_MISC)) {
    }
    if(defined(&__USE_ATFILE)) {
    }
    if(defined (&__USE_XOPEN_EXTENDED) || defined (&__USE_XOPEN2K)) {
    }
    if(defined(&__USE_ATFILE)) {
    }
    if(defined(&__USE_ATFILE)) {
    }
    if(defined(&__USE_POSIX199506)) {
    }
    if(defined(&__USE_MISC)) {
    }
    if(defined(&__USE_POSIX2)) {
	require 'bits/getopt_posix.ph';
    }
    if(defined (&__USE_XOPEN_EXTENDED) || defined (&__USE_XOPEN2K)) {
    }
    if(defined (&__USE_MISC)) {
    }
    if(defined (&__USE_MISC) || (defined (&__USE_XOPEN)  && !defined (&__USE_XOPEN2K))) {
    }
    if(defined(&__USE_GNU)) {
    }
    if(defined (&__USE_MISC) || defined (&__USE_XOPEN_EXTENDED)) {
	if(defined (&__USE_MISC) || !defined (&__USE_XOPEN2K)) {
	}
    }
    if(defined (&__USE_XOPEN_EXTENDED) || defined (&__USE_XOPEN2K8)) {
	unless(defined(&__USE_FILE_OFFSET64)) {
	} else {
	    if(defined(&__REDIRECT_NTH)) {
	    } else {
		eval 'sub truncate () { &truncate64;}' unless defined(&truncate);
	    }
	}
	if(defined(&__USE_LARGEFILE64)) {
	}
    }
    if(defined (&__USE_POSIX199309) || defined (&__USE_XOPEN_EXTENDED) || defined (&__USE_XOPEN2K)) {
	unless(defined(&__USE_FILE_OFFSET64)) {
	} else {
	    if(defined(&__REDIRECT_NTH)) {
	    } else {
		eval 'sub ftruncate () { &ftruncate64;}' unless defined(&ftruncate);
	    }
	}
	if(defined(&__USE_LARGEFILE64)) {
	}
    }
    if((defined (&__USE_XOPEN_EXTENDED)  && !defined (&__USE_XOPEN2K)) || defined (&__USE_MISC)) {
    }
    if(defined(&__USE_MISC)) {
    }
    if((defined (&__USE_MISC) || defined (&__USE_XOPEN_EXTENDED))  && !defined (&F_LOCK)) {
	eval 'sub F_ULOCK () {0;}' unless defined(&F_ULOCK);
	eval 'sub F_LOCK () {1;}' unless defined(&F_LOCK);
	eval 'sub F_TLOCK () {2;}' unless defined(&F_TLOCK);
	eval 'sub F_TEST () {3;}' unless defined(&F_TEST);
	unless(defined(&__USE_FILE_OFFSET64)) {
	} else {
	    if(defined(&__REDIRECT)) {
	    } else {
		eval 'sub lockf () { &lockf64;}' unless defined(&lockf);
	    }
	}
	if(defined(&__USE_LARGEFILE64)) {
	}
    }
    if(defined(&__USE_GNU)) {
	eval 'sub TEMP_FAILURE_RETRY {
	    my($expression) = @_;
    	    eval q(( &__extension__ ({ \'long int __result\';  &do  &__result = ($expression);  &while ( &__result == -1  &&  &errno ==  &EINTR);  &__result; })));
	}' unless defined(&TEMP_FAILURE_RETRY);
    }
    if(defined (&__USE_POSIX199309) || defined (&__USE_UNIX98)) {
    }
    if(defined(&__USE_MISC)) {
    }
    if(defined(&__USE_XOPEN)) {
    }
    if(defined (&__USE_XOPEN)  && !defined (&__USE_XOPEN2K)) {
    }
    if(defined (&__USE_UNIX98)  && !defined (&__USE_XOPEN2K)) {
    }
    if(defined(&__USE_MISC)) {
    }
    if(defined(&__USE_GNU)) {
    }
    if((defined(&__USE_FORTIFY_LEVEL) ? &__USE_FORTIFY_LEVEL : undef) > 0 && defined (&__fortify_function)) {
	require 'bits/unistd.ph';
    }
    require 'bits/unistd_ext.ph';
}
1;
