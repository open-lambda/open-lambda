require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_STDARG_H)) {
    unless(defined(&_ANSI_STDARG_H_)) {
	unless(defined(&__need___va_list)) {
	    eval 'sub _STDARG_H () {1;}' unless defined(&_STDARG_H);
	    eval 'sub _ANSI_STDARG_H_ () {1;}' unless defined(&_ANSI_STDARG_H_);
	}
	undef(&__need___va_list) if defined(&__need___va_list);
	unless(defined(&__GNUC_VA_LIST)) {
	    eval 'sub __GNUC_VA_LIST () {1;}' unless defined(&__GNUC_VA_LIST);
	}
	if(defined(&_STDARG_H)) {
	    eval 'sub va_start {
	        my($v,$l) = @_;
    		eval q( &__builtin_va_start($v,$l));
	    }' unless defined(&va_start);
	    eval 'sub va_end {
	        my($v) = @_;
    		eval q( &__builtin_va_end($v));
	    }' unless defined(&va_end);
	    eval 'sub va_arg {
	        my($v,$l) = @_;
    		eval q( &__builtin_va_arg($v,$l));
	    }' unless defined(&va_arg);
	    if(!defined(&__STRICT_ANSI__) || (defined(&__STDC_VERSION__) ? &__STDC_VERSION__ : undef) + 0>= 199900 || (defined(&__cplusplus) ? &__cplusplus : undef) + 0>= 201103) {
		eval 'sub va_copy {
		    my($d,$s) = @_;
    		    eval q( &__builtin_va_copy($d,$s));
		}' unless defined(&va_copy);
	    }
	    eval 'sub __va_copy {
	        my($d,$s) = @_;
    		eval q( &__builtin_va_copy($d,$s));
	    }' unless defined(&__va_copy);
	    if(defined(&_BSD_VA_LIST)) {
		undef(&_BSD_VA_LIST) if defined(&_BSD_VA_LIST);
	    }
	    if(defined(&__svr4__) || (defined(&_SCO_DS)  && !defined(&__VA_LIST))) {
		unless(defined(&_VA_LIST_)) {
		    eval 'sub _VA_LIST_ () {1;}' unless defined(&_VA_LIST_);
		    if(defined(&__i860__)) {
			unless(defined(&_VA_LIST)) {
			    eval 'sub _VA_LIST () { &va_list;}' unless defined(&_VA_LIST);
			}
		    }
		    if(defined(&_SCO_DS)) {
			eval 'sub __VA_LIST () {1;}' unless defined(&__VA_LIST);
		    }
		}
	    } else {
		if(!defined (&_VA_LIST_) || defined (&__BSD_NET2__) || defined (&____386BSD____) || defined (&__bsdi__) || defined (&__sequent__) || defined (&__FreeBSD__) || defined(&WINNT)) {
		    unless(defined(&_VA_LIST_DEFINED)) {
			unless(defined(&_VA_LIST)) {
			    unless(defined(&_VA_LIST_T_H)) {
				unless(defined(&__va_list__)) {
				}
			    }
			}
		    }
		    if(!(defined (&__BSD_NET2__) || defined (&____386BSD____) || defined (&__bsdi__) || defined (&__sequent__) || defined (&__FreeBSD__))) {
			eval 'sub _VA_LIST_ () {1;}' unless defined(&_VA_LIST_);
		    }
		    unless(defined(&_VA_LIST)) {
			eval 'sub _VA_LIST () {1;}' unless defined(&_VA_LIST);
		    }
		    unless(defined(&_VA_LIST_DEFINED)) {
			eval 'sub _VA_LIST_DEFINED () {1;}' unless defined(&_VA_LIST_DEFINED);
		    }
		    unless(defined(&_VA_LIST_T_H)) {
			eval 'sub _VA_LIST_T_H () {1;}' unless defined(&_VA_LIST_T_H);
		    }
		    unless(defined(&__va_list__)) {
			eval 'sub __va_list__ () {1;}' unless defined(&__va_list__);
		    }
		}
	    }
	}
    }
}
1;
