require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_SYS_CDEFS_H)) {
    eval 'sub _SYS_CDEFS_H () {1;}' unless defined(&_SYS_CDEFS_H);
    unless(defined(&_FEATURES_H)) {
	require 'features.ph';
    }
    if(defined (&__GNUC__)  && !defined (&__STDC__)) {
	die("You need a ISO C conforming compiler to use the glibc headers");
    }
    undef(&__P) if defined(&__P);
    undef(&__PMT) if defined(&__PMT);
    if((defined (&__has_attribute)  && (!defined (&__clang_minor__) || 3< (defined(&__clang_major__) ? &__clang_major__ : undef) + (5<= (defined(&__clang_minor__) ? &__clang_minor__ : undef))))) {
	eval 'sub __glibc_has_attribute {
	    my($attr) = @_;
    	    eval q( &__has_attribute ($attr));
	}' unless defined(&__glibc_has_attribute);
    } else {
	eval 'sub __glibc_has_attribute {
	    my($attr) = @_;
    	    eval q(0);
	}' unless defined(&__glibc_has_attribute);
    }
    if(defined(&__has_builtin)) {
	eval 'sub __glibc_has_builtin {
	    my($name) = @_;
    	    eval q( &__has_builtin ($name));
	}' unless defined(&__glibc_has_builtin);
    } else {
	eval 'sub __glibc_has_builtin {
	    my($name) = @_;
    	    eval q(0);
	}' unless defined(&__glibc_has_builtin);
    }
    if(defined(&__has_extension)) {
	eval 'sub __glibc_has_extension {
	    my($ext) = @_;
    	    eval q( &__has_extension ($ext));
	}' unless defined(&__glibc_has_extension);
    } else {
	eval 'sub __glibc_has_extension {
	    my($ext) = @_;
    	    eval q(0);
	}' unless defined(&__glibc_has_extension);
    }
    if(defined (&__GNUC__) || defined (&__clang__)) {
	if( &__GNUC_PREREQ (4, 6)  && !defined (&_LIBC)) {
	    eval 'sub __LEAF () {,  &__leaf__;}' unless defined(&__LEAF);
	    eval 'sub __LEAF_ATTR () { &__attribute__ (( &__leaf__));}' unless defined(&__LEAF_ATTR);
	} else {
	    eval 'sub __LEAF () {1;}' unless defined(&__LEAF);
	    eval 'sub __LEAF_ATTR () {1;}' unless defined(&__LEAF_ATTR);
	}
	if(!defined (&__cplusplus)  && ( &__GNUC_PREREQ (3, 4) ||  &__glibc_has_attribute ((defined(&__nothrow__) ? &__nothrow__ : undef)))) {
	    eval 'sub __THROW () { &__attribute__ (( &__nothrow__  &__LEAF));}' unless defined(&__THROW);
	    eval 'sub __THROWNL () { &__attribute__ (( &__nothrow__));}' unless defined(&__THROWNL);
	    eval 'sub __NTH {
	        my($fct) = @_;
    		eval q( &__attribute__ (( &__nothrow__  &__LEAF)) $fct);
	    }' unless defined(&__NTH);
	    eval 'sub __NTHNL {
	        my($fct) = @_;
    		eval q( &__attribute__ (( &__nothrow__)) $fct);
	    }' unless defined(&__NTHNL);
	} else {
	    if(defined (&__cplusplus)  && ( &__GNUC_PREREQ (2,8) || (defined(&__clang_major) ? &__clang_major : undef) >= 4)) {
		if((defined(&__cplusplus) ? &__cplusplus : undef) >= 201103) {
		    eval 'sub __THROW () { &noexcept ( &true);}' unless defined(&__THROW);
		} else {
		    eval 'sub __THROW () { &throw ();}' unless defined(&__THROW);
		}
		eval 'sub __THROWNL () { &__THROW;}' unless defined(&__THROWNL);
		eval 'sub __NTH {
		    my($fct) = @_;
    		    eval q( &__LEAF_ATTR $fct  &__THROW);
		}' unless defined(&__NTH);
		eval 'sub __NTHNL {
		    my($fct) = @_;
    		    eval q($fct  &__THROW);
		}' unless defined(&__NTHNL);
	    } else {
		eval 'sub __THROW () {1;}' unless defined(&__THROW);
		eval 'sub __THROWNL () {1;}' unless defined(&__THROWNL);
		eval 'sub __NTH {
		    my($fct) = @_;
    		    eval q($fct);
		}' unless defined(&__NTH);
		eval 'sub __NTHNL {
		    my($fct) = @_;
    		    eval q($fct);
		}' unless defined(&__NTHNL);
	    }
	}
    } else {
	if((defined (&__cplusplus) || (defined (&__STDC_VERSION__)  && (defined(&__STDC_VERSION__) ? &__STDC_VERSION__ : undef) >= 199901))) {
	    eval 'sub __inline () { &inline;}' unless defined(&__inline);
	} else {
	    eval 'sub __inline () {1;}' unless defined(&__inline);
	}
	eval 'sub __THROW () {1;}' unless defined(&__THROW);
	eval 'sub __THROWNL () {1;}' unless defined(&__THROWNL);
	eval 'sub __NTH {
	    my($fct) = @_;
    	    eval q($fct);
	}' unless defined(&__NTH);
    }
    eval 'sub __P {
        my($args) = @_;
	    eval q($args);
    }' unless defined(&__P);
    eval 'sub __PMT {
        my($args) = @_;
	    eval q($args);
    }' unless defined(&__PMT);
    eval 'sub __CONCAT {
        my($x,$y) = @_;
	    eval q($x  $y);
    }' unless defined(&__CONCAT);
    eval 'sub __STRING {
        my($x) = @_;
	    eval q($x);
    }' unless defined(&__STRING);
    eval 'sub __ptr_t () { &void *;}' unless defined(&__ptr_t);
    if(defined(&__cplusplus)) {
	eval 'sub __BEGIN_DECLS () { &extern "C" {;}' unless defined(&__BEGIN_DECLS);
	eval 'sub __END_DECLS () {};}' unless defined(&__END_DECLS);
    } else {
	eval 'sub __BEGIN_DECLS () {1;}' unless defined(&__BEGIN_DECLS);
	eval 'sub __END_DECLS () {1;}' unless defined(&__END_DECLS);
    }
    eval 'sub __bos {
        my($ptr) = @_;
	    eval q( &__builtin_object_size ($ptr,  &__USE_FORTIFY_LEVEL > 1));
    }' unless defined(&__bos);
    eval 'sub __bos0 {
        my($ptr) = @_;
	    eval q( &__builtin_object_size ($ptr, 0));
    }' unless defined(&__bos0);
    if((defined(&__USE_FORTIFY_LEVEL) ? &__USE_FORTIFY_LEVEL : undef) == 3 && ( &__glibc_clang_prereq (9, 0) ||  &__GNUC_PREREQ (12, 0))) {
	eval 'sub __glibc_objsize0 {
	    my($__o) = @_;
    	    eval q( &__builtin_dynamic_object_size ($__o, 0));
	}' unless defined(&__glibc_objsize0);
	eval 'sub __glibc_objsize {
	    my($__o) = @_;
    	    eval q( &__builtin_dynamic_object_size ($__o, 1));
	}' unless defined(&__glibc_objsize);
    } else {
	eval 'sub __glibc_objsize0 {
	    my($__o) = @_;
    	    eval q( &__bos0 ($__o));
	}' unless defined(&__glibc_objsize0);
	eval 'sub __glibc_objsize {
	    my($__o) = @_;
    	    eval q( &__bos ($__o));
	}' unless defined(&__glibc_objsize);
    }
    eval 'sub __glibc_safe_len_cond {
        my($__l, $__s, $__osz) = @_;
	    eval q((($__l) <= ($__osz) / ($__s)));
    }' unless defined(&__glibc_safe_len_cond);
    eval 'sub __glibc_unsigned_or_positive {
        my($__l) = @_;
	    eval q((( &__typeof ($__l)) 0< ( &__typeof ($__l)) -1|| ( &__builtin_constant_p ($__l)  && ($__l) > 0)));
    }' unless defined(&__glibc_unsigned_or_positive);
    eval 'sub __glibc_safe_or_unknown_len {
        my($__l, $__s, $__osz) = @_;
	    eval q(( &__glibc_unsigned_or_positive ($__l)  &&  &__builtin_constant_p ( &__glibc_safe_len_cond (( &__SIZE_TYPE__) ($__l), $__s, $__osz))  &&  &__glibc_safe_len_cond (( &__SIZE_TYPE__) ($__l), $__s, $__osz)));
    }' unless defined(&__glibc_safe_or_unknown_len);
    eval 'sub __glibc_unsafe_len {
        my($__l, $__s, $__osz) = @_;
	    eval q(( &__glibc_unsigned_or_positive ($__l)  &&  &__builtin_constant_p ( &__glibc_safe_len_cond (( &__SIZE_TYPE__) ($__l), $__s, $__osz))  && ! &__glibc_safe_len_cond (( &__SIZE_TYPE__) ($__l), $__s, $__osz)));
    }' unless defined(&__glibc_unsafe_len);
    eval 'sub __glibc_fortify () {( &f,  &__l,  &__s,  &__osz, ...) ( &__glibc_safe_or_unknown_len ( &__l,  &__s,  &__osz) ?  &__   &f   &_alias ( &__VA_ARGS__) : ( &__glibc_unsafe_len ( &__l,  &__s,  &__osz) ?  &__   &f   &_chk_warn ( &__VA_ARGS__,  &__osz) :  &__   &f   &_chk ( &__VA_ARGS__,  &__osz)));}' unless defined(&__glibc_fortify);
    eval 'sub __glibc_fortify_n () {( &f,  &__l,  &__s,  &__osz, ...) ( &__glibc_safe_or_unknown_len ( &__l,  &__s,  &__osz) ?  &__   &f   &_alias ( &__VA_ARGS__) : ( &__glibc_unsafe_len ( &__l,  &__s,  &__osz) ?  &__   &f   &_chk_warn ( &__VA_ARGS__, ( &__osz) / ( &__s)) :  &__   &f   &_chk ( &__VA_ARGS__, ( &__osz) / ( &__s))));}' unless defined(&__glibc_fortify_n);
    if( &__GNUC_PREREQ (4,3)) {
	eval 'sub __warnattr {
	    my($msg) = @_;
    	    eval q( &__attribute__(( &__warning__ ($msg))));
	}' unless defined(&__warnattr);
	eval 'sub __errordecl {
	    my($name, $msg) = @_;
    	    eval q( &extern  &void $name ( &void)  &__attribute__(( &__error__ ($msg))));
	}' unless defined(&__errordecl);
    } else {
	eval 'sub __warnattr {
	    my($msg) = @_;
    	    eval q();
	}' unless defined(&__warnattr);
	eval 'sub __errordecl {
	    my($name, $msg) = @_;
    	    eval q( &extern  &void $name ( &void));
	}' unless defined(&__errordecl);
    }
    if(defined (&__STDC_VERSION__)  && (defined(&__STDC_VERSION__) ? &__STDC_VERSION__ : undef) >= 199901  && !defined (&__HP_cc)) {
	eval 'sub __flexarr () {[];}' unless defined(&__flexarr);
	eval 'sub __glibc_c99_flexarr_available () {1;}' unless defined(&__glibc_c99_flexarr_available);
    }
 elsif( &__GNUC_PREREQ (2,97) || defined (&__clang__)) {
	eval 'sub __flexarr () {[];}' unless defined(&__flexarr);
	eval 'sub __glibc_c99_flexarr_available () {1;}' unless defined(&__glibc_c99_flexarr_available);
    }
 elsif(defined (&__GNUC__)) {
	eval 'sub __flexarr () {[0];}' unless defined(&__flexarr);
	eval 'sub __glibc_c99_flexarr_available () {1;}' unless defined(&__glibc_c99_flexarr_available);
    } else {
	eval 'sub __flexarr () {[1];}' unless defined(&__flexarr);
	eval 'sub __glibc_c99_flexarr_available () {0;}' unless defined(&__glibc_c99_flexarr_available);
    }
    if((defined (&__GNUC__)  && (defined(&__GNUC__) ? &__GNUC__ : undef) >= 2) || ((defined(&__clang_major__) ? &__clang_major__ : undef) >= 4)) {
	eval 'sub __REDIRECT {
	    my($name, $proto, $alias) = @_;
    	    eval q(\\"(assembly code)\\");
	}' unless defined(&__REDIRECT);
	if(defined(&__cplusplus)) {
	    eval 'sub __REDIRECT_NTH {
	        my($name, $proto, $alias) = @_;
    		eval q(\\"(assembly code)\\");
	    }' unless defined(&__REDIRECT_NTH);
	    eval 'sub __REDIRECT_NTHNL {
	        my($name, $proto, $alias) = @_;
    		eval q(\\"(assembly code)\\");
	    }' unless defined(&__REDIRECT_NTHNL);
	} else {
	    eval 'sub __REDIRECT_NTH {
	        my($name, $proto, $alias) = @_;
    		eval q(\\"(assembly code)\\");
	    }' unless defined(&__REDIRECT_NTH);
	    eval 'sub __REDIRECT_NTHNL {
	        my($name, $proto, $alias) = @_;
    		eval q(\\"(assembly code)\\");
	    }' unless defined(&__REDIRECT_NTHNL);
	}
	eval 'sub __ASMNAME {
	    my($cname) = @_;
    	    eval q( &__ASMNAME2 ( &__USER_LABEL_PREFIX__, $cname));
	}' unless defined(&__ASMNAME);
	eval 'sub __ASMNAME2 {
	    my($prefix, $cname) = @_;
    	    eval q( &__STRING ($prefix) $cname);
	}' unless defined(&__ASMNAME2);
    }
    if(!(defined (&__GNUC__) || defined (&__clang__))) {
	eval 'sub __attribute__ {
	    my($xyz) = @_;
    	    eval q();
	}' unless defined(&__attribute__);
    }
    if( &__GNUC_PREREQ (2,96) ||  &__glibc_has_attribute ((defined(&__malloc__) ? &__malloc__ : undef))) {
	eval 'sub __attribute_malloc__ () { &__attribute__ (( &__malloc__));}' unless defined(&__attribute_malloc__);
    } else {
	eval 'sub __attribute_malloc__ () {1;}' unless defined(&__attribute_malloc__);
    }
    if( &__GNUC_PREREQ (4, 3)) {
	eval 'sub __attribute_alloc_size__ {
	    my($params) = @_;
    	    eval q( &__attribute__ (( &__alloc_size__ $params)));
	}' unless defined(&__attribute_alloc_size__);
    } else {
	eval 'sub __attribute_alloc_size__ {
	    my($params) = @_;
    	    eval q();
	}' unless defined(&__attribute_alloc_size__);
    }
    if( &__GNUC_PREREQ (4, 9) ||  &__glibc_has_attribute ((defined(&__alloc_align__) ? &__alloc_align__ : undef))) {
	eval 'sub __attribute_alloc_align__ {
	    my($param) = @_;
    	    eval q( &__attribute__ (( &__alloc_align__ $param)));
	}' unless defined(&__attribute_alloc_align__);
    } else {
	eval 'sub __attribute_alloc_align__ {
	    my($param) = @_;
    	    eval q();
	}' unless defined(&__attribute_alloc_align__);
    }
    if( &__GNUC_PREREQ (2,96) ||  &__glibc_has_attribute ((defined(&__pure__) ? &__pure__ : undef))) {
	eval 'sub __attribute_pure__ () { &__attribute__ (( &__pure__));}' unless defined(&__attribute_pure__);
    } else {
	eval 'sub __attribute_pure__ () {1;}' unless defined(&__attribute_pure__);
    }
    if( &__GNUC_PREREQ (2,5) ||  &__glibc_has_attribute ((defined(&__const__) ? &__const__ : undef))) {
	eval 'sub __attribute_const__ () { &__attribute__ (( &__const__));}' unless defined(&__attribute_const__);
    } else {
	eval 'sub __attribute_const__ () {1;}' unless defined(&__attribute_const__);
    }
    if( &__GNUC_PREREQ (2,7) ||  &__glibc_has_attribute ((defined(&__unused__) ? &__unused__ : undef))) {
	eval 'sub __attribute_maybe_unused__ () { &__attribute__ (( &__unused__));}' unless defined(&__attribute_maybe_unused__);
    } else {
	eval 'sub __attribute_maybe_unused__ () {1;}' unless defined(&__attribute_maybe_unused__);
    }
    if( &__GNUC_PREREQ (3,1) ||  &__glibc_has_attribute ((defined(&__used__) ? &__used__ : undef))) {
	eval 'sub __attribute_used__ () { &__attribute__ (( &__used__));}' unless defined(&__attribute_used__);
	eval 'sub __attribute_noinline__ () { &__attribute__ (( &__noinline__));}' unless defined(&__attribute_noinline__);
    } else {
	eval 'sub __attribute_used__ () { &__attribute__ (( &__unused__));}' unless defined(&__attribute_used__);
	eval 'sub __attribute_noinline__ () {1;}' unless defined(&__attribute_noinline__);
    }
    if( &__GNUC_PREREQ (3,2) ||  &__glibc_has_attribute ((defined(&__deprecated__) ? &__deprecated__ : undef))) {
	eval 'sub __attribute_deprecated__ () { &__attribute__ (( &__deprecated__));}' unless defined(&__attribute_deprecated__);
    } else {
	eval 'sub __attribute_deprecated__ () {1;}' unless defined(&__attribute_deprecated__);
    }
    if( &__GNUC_PREREQ (4,5) ||  &__glibc_has_extension ((defined(&__attribute_deprecated_with_message__) ? &__attribute_deprecated_with_message__ : undef))) {
	eval 'sub __attribute_deprecated_msg__ {
	    my($msg) = @_;
    	    eval q( &__attribute__ (( &__deprecated__ ($msg))));
	}' unless defined(&__attribute_deprecated_msg__);
    } else {
	eval 'sub __attribute_deprecated_msg__ {
	    my($msg) = @_;
    	    eval q( &__attribute_deprecated__);
	}' unless defined(&__attribute_deprecated_msg__);
    }
    if( &__GNUC_PREREQ (2,8) ||  &__glibc_has_attribute ((defined(&__format_arg__) ? &__format_arg__ : undef))) {
	eval 'sub __attribute_format_arg__ {
	    my($x) = @_;
    	    eval q( &__attribute__ (( &__format_arg__ ($x))));
	}' unless defined(&__attribute_format_arg__);
    } else {
	eval 'sub __attribute_format_arg__ {
	    my($x) = @_;
    	    eval q();
	}' unless defined(&__attribute_format_arg__);
    }
    if( &__GNUC_PREREQ (2,97) ||  &__glibc_has_attribute ((defined(&__format__) ? &__format__ : undef))) {
	eval 'sub __attribute_format_strfmon__ {
	    my($a,$b) = @_;
    	    eval q( &__attribute__ (( &__format__ ( &__strfmon__, $a, $b))));
	}' unless defined(&__attribute_format_strfmon__);
    } else {
	eval 'sub __attribute_format_strfmon__ {
	    my($a,$b) = @_;
    	    eval q();
	}' unless defined(&__attribute_format_strfmon__);
    }
    unless(defined(&__attribute_nonnull__)) {
	if( &__GNUC_PREREQ (3,3) ||  &__glibc_has_attribute ((defined(&__nonnull__) ? &__nonnull__ : undef))) {
	    eval 'sub __attribute_nonnull__ {
	        my($params) = @_;
    		eval q( &__attribute__ (( &__nonnull__ $params)));
	    }' unless defined(&__attribute_nonnull__);
	} else {
	    eval 'sub __attribute_nonnull__ {
	        my($params) = @_;
    		eval q();
	    }' unless defined(&__attribute_nonnull__);
	}
    }
    unless(defined(&__nonnull)) {
	eval 'sub __nonnull {
	    my($params) = @_;
    	    eval q( &__attribute_nonnull__ ($params));
	}' unless defined(&__nonnull);
    }
    unless(defined(&__returns_nonnull)) {
	if( &__GNUC_PREREQ (4, 9) ||  &__glibc_has_attribute ((defined(&__returns_nonnull__) ? &__returns_nonnull__ : undef))) {
	    eval 'sub __returns_nonnull () { &__attribute__ (( &__returns_nonnull__));}' unless defined(&__returns_nonnull);
	} else {
	    eval 'sub __returns_nonnull () {1;}' unless defined(&__returns_nonnull);
	}
    }
    if( &__GNUC_PREREQ (3,4) ||  &__glibc_has_attribute ((defined(&__warn_unused_result__) ? &__warn_unused_result__ : undef))) {
	eval 'sub __attribute_warn_unused_result__ () { &__attribute__ (( &__warn_unused_result__));}' unless defined(&__attribute_warn_unused_result__);
	if(defined (&__USE_FORTIFY_LEVEL)  && (defined(&__USE_FORTIFY_LEVEL) ? &__USE_FORTIFY_LEVEL : undef) > 0) {
	    eval 'sub __wur () { &__attribute_warn_unused_result__;}' unless defined(&__wur);
	}
    } else {
	eval 'sub __attribute_warn_unused_result__ () {1;}' unless defined(&__attribute_warn_unused_result__);
    }
    unless(defined(&__wur)) {
	eval 'sub __wur () {1;}' unless defined(&__wur);
    }
    if( &__GNUC_PREREQ (3,2) ||  &__glibc_has_attribute ((defined(&__always_inline__) ? &__always_inline__ : undef))) {
	undef(&__always_inline) if defined(&__always_inline);
	eval 'sub __always_inline () { &__inline  &__attribute__ (( &__always_inline__));}' unless defined(&__always_inline);
    } else {
	undef(&__always_inline) if defined(&__always_inline);
	eval 'sub __always_inline () { &__inline;}' unless defined(&__always_inline);
    }
    if( &__GNUC_PREREQ (4,3) ||  &__glibc_has_attribute ((defined(&__artificial__) ? &__artificial__ : undef))) {
	eval 'sub __attribute_artificial__ () { &__attribute__ (( &__artificial__));}' unless defined(&__attribute_artificial__);
    } else {
	eval 'sub __attribute_artificial__ () {1;}' unless defined(&__attribute_artificial__);
    }
    if((!defined (&__cplusplus) ||  &__GNUC_PREREQ (4,3) || (defined (&__clang__)  && (defined (&__GNUC_STDC_INLINE__) || defined (&__GNUC_GNU_INLINE__))))) {
	if(defined (&__GNUC_STDC_INLINE__) || defined (&__cplusplus)) {
	    eval 'sub __extern_inline () { &extern  &__inline  &__attribute__ (( &__gnu_inline__));}' unless defined(&__extern_inline);
	    eval 'sub __extern_always_inline () { &extern  &__always_inline  &__attribute__ (( &__gnu_inline__));}' unless defined(&__extern_always_inline);
	} else {
	    eval 'sub __extern_inline () { &extern  &__inline;}' unless defined(&__extern_inline);
	    eval 'sub __extern_always_inline () { &extern  &__always_inline;}' unless defined(&__extern_always_inline);
	}
    }
    if(defined(&__extern_always_inline)) {
	eval 'sub __fortify_function () { &__extern_always_inline  &__attribute_artificial__;}' unless defined(&__fortify_function);
    }
    if( &__GNUC_PREREQ (4,3)) {
	eval 'sub __va_arg_pack () {
	    eval q( &__builtin_va_arg_pack ());
	}' unless defined(&__va_arg_pack);
	eval 'sub __va_arg_pack_len () {
	    eval q( &__builtin_va_arg_pack_len ());
	}' unless defined(&__va_arg_pack_len);
    }
    if(!( &__GNUC_PREREQ (2,8) || defined (&__clang__))) {
	eval 'sub __extension__ () {1;}' unless defined(&__extension__);
    }
    if(!( &__GNUC_PREREQ (2,92) || (defined(&__clang_major__) ? &__clang_major__ : undef) >= 3)) {
	if(defined (&__STDC_VERSION__)  && (defined(&__STDC_VERSION__) ? &__STDC_VERSION__ : undef) >= 199901) {
	    eval 'sub __restrict () { &restrict;}' unless defined(&__restrict);
	} else {
	    eval 'sub __restrict () {1;}' unless defined(&__restrict);
	}
    }
    if(( &__GNUC_PREREQ (3,1) || (defined(&__clang_major__) ? &__clang_major__ : undef) >= 3)  && !defined (&__cplusplus)) {
	eval 'sub __restrict_arr () { &__restrict;}' unless defined(&__restrict_arr);
    } else {
	if(defined(&__GNUC__)) {
	    eval 'sub __restrict_arr () {1;}' unless defined(&__restrict_arr);
	} else {
	    if(defined (&__STDC_VERSION__)  && (defined(&__STDC_VERSION__) ? &__STDC_VERSION__ : undef) >= 199901) {
		eval 'sub __restrict_arr () { &restrict;}' unless defined(&__restrict_arr);
	    } else {
		eval 'sub __restrict_arr () {1;}' unless defined(&__restrict_arr);
	    }
	}
    }
    if(((defined(&__GNUC__) ? &__GNUC__ : undef) >= 3) ||  &__glibc_has_builtin ((defined(&__builtin_expect) ? &__builtin_expect : undef))) {
	eval 'sub __glibc_unlikely {
	    my($cond) = @_;
    	    eval q( &__builtin_expect (($cond), 0));
	}' unless defined(&__glibc_unlikely);
	eval 'sub __glibc_likely {
	    my($cond) = @_;
    	    eval q( &__builtin_expect (($cond), 1));
	}' unless defined(&__glibc_likely);
    } else {
	eval 'sub __glibc_unlikely {
	    my($cond) = @_;
    	    eval q(($cond));
	}' unless defined(&__glibc_unlikely);
	eval 'sub __glibc_likely {
	    my($cond) = @_;
    	    eval q(($cond));
	}' unless defined(&__glibc_likely);
    }
    if((!defined (&_Noreturn)  && (defined (&__STDC_VERSION__) ? (defined(&__STDC_VERSION__) ? &__STDC_VERSION__ : undef) : 0) < 201112 && !( &__GNUC_PREREQ (4,7) || (3< (defined(&__clang_major__) ? &__clang_major__ : undef) + (5<= (defined(&__clang_minor__) ? &__clang_minor__ : undef)))))) {
	if( &__GNUC_PREREQ (2,8)) {
	    eval 'sub _Noreturn () { &__attribute__ (( &__noreturn__));}' unless defined(&_Noreturn);
	} else {
	    eval 'sub _Noreturn () {1;}' unless defined(&_Noreturn);
	}
    }
    if( &__GNUC_PREREQ (8, 0)) {
	eval 'sub __attribute_nonstring__ () { &__attribute__ (( &__nonstring__));}' unless defined(&__attribute_nonstring__);
    } else {
	eval 'sub __attribute_nonstring__ () {1;}' unless defined(&__attribute_nonstring__);
    }
    undef(&__attribute_copy__) if defined(&__attribute_copy__);
    if( &__GNUC_PREREQ (9, 0)) {
	eval 'sub __attribute_copy__ {
	    my($arg) = @_;
    	    eval q( &__attribute__ (( &__copy__ ($arg))));
	}' unless defined(&__attribute_copy__);
    } else {
	eval 'sub __attribute_copy__ {
	    my($arg) = @_;
    	    eval q();
	}' unless defined(&__attribute_copy__);
    }
    if((!defined (&_Static_assert)  && !defined (&__cplusplus)  && (defined (&__STDC_VERSION__) ? (defined(&__STDC_VERSION__) ? &__STDC_VERSION__ : undef) : 0) < 201112 && (!( &__GNUC_PREREQ (4, 6) || (defined(&__clang_major__) ? &__clang_major__ : undef) >= 4) || defined (&__STRICT_ANSI__)))) {
	eval 'sub _Static_assert {
	    my($expr, $diagnostic) = @_;
    	    eval q( &extern  &int (* &__Static_assert_function ( &void)) [!!$sizeof{\'struct struct\' { \'int\'  &__error_if_negative: ($expr) ? 2: -1; }}]);
	}' unless defined(&_Static_assert);
    }
    unless(defined(&__GNULIB_CDEFS)) {
	require 'bits/wordsize.ph';
	require 'bits/long-double.ph';
    }
    if((defined(&__LDOUBLE_REDIRECTS_TO_FLOAT128_ABI) ? &__LDOUBLE_REDIRECTS_TO_FLOAT128_ABI : undef) == 1) {
	if(defined(&__REDIRECT)) {
	    eval 'sub __LDBL_REDIR {
	        my($name, $proto) = @_;
    		eval q(...  &unused__ldbl_redir);
	    }' unless defined(&__LDBL_REDIR);
	    eval 'sub __LDBL_REDIR_DECL {
	        my($name) = @_;
    		eval q( &extern  &__typeof ($name) $name  &__asm ( &__ASMNAME (\\"__\\" $name \\"ieee128\\")););
	    }' unless defined(&__LDBL_REDIR_DECL);
	    eval 'sub __LDBL_REDIR2_DECL {
	        my($name) = @_;
    		eval q( &extern  &__typeof ( &__$name)  &__$name  &__asm ( &__ASMNAME (\\"__\\" $name \\"ieee128\\")););
	    }' unless defined(&__LDBL_REDIR2_DECL);
	    eval 'sub __LDBL_REDIR1 {
	        my($name, $proto, $alias) = @_;
    		eval q(...  &unused__ldbl_redir1);
	    }' unless defined(&__LDBL_REDIR1);
	    eval 'sub __LDBL_REDIR1_DECL {
	        my($name, $alias) = @_;
    		eval q( &extern  &__typeof ($name) $name  &__asm ( &__ASMNAME ($alias)););
	    }' unless defined(&__LDBL_REDIR1_DECL);
	    eval 'sub __LDBL_REDIR1_NTH {
	        my($name, $proto, $alias) = @_;
    		eval q( &__REDIRECT_NTH ($name, $proto, $alias));
	    }' unless defined(&__LDBL_REDIR1_NTH);
	    eval 'sub __REDIRECT_NTH_LDBL {
	        my($name, $proto, $alias) = @_;
    		eval q( &__LDBL_REDIR1_NTH ($name, $proto,  &__$alias &ieee128));
	    }' unless defined(&__REDIRECT_NTH_LDBL);
	    eval 'sub __REDIRECT_LDBL {
	        my($name, $proto, $alias) = @_;
    		eval q(...  &unused__redirect_ldbl);
	    }' unless defined(&__REDIRECT_LDBL);
	    eval 'sub __LDBL_REDIR_NTH {
	        my($name, $proto) = @_;
    		eval q(...  &unused__ldbl_redir_nth);
	    }' unless defined(&__LDBL_REDIR_NTH);
	} else {
	}
    }
 elsif(defined (&__LONG_DOUBLE_MATH_OPTIONAL)  && defined (&__NO_LONG_DOUBLE_MATH)) {
	eval 'sub __LDBL_COMPAT () {1;}' unless defined(&__LDBL_COMPAT);
	if(defined(&__REDIRECT)) {
	    eval 'sub __LDBL_REDIR1 {
	        my($name, $proto, $alias) = @_;
    		eval q( &__REDIRECT ($name, $proto, $alias));
	    }' unless defined(&__LDBL_REDIR1);
	    eval 'sub __LDBL_REDIR {
	        my($name, $proto) = @_;
    		eval q( &__LDBL_REDIR1 ($name, $proto,  &__nldbl_$name));
	    }' unless defined(&__LDBL_REDIR);
	    eval 'sub __LDBL_REDIR1_NTH {
	        my($name, $proto, $alias) = @_;
    		eval q( &__REDIRECT_NTH ($name, $proto, $alias));
	    }' unless defined(&__LDBL_REDIR1_NTH);
	    eval 'sub __LDBL_REDIR_NTH {
	        my($name, $proto) = @_;
    		eval q( &__LDBL_REDIR1_NTH ($name, $proto,  &__nldbl_$name));
	    }' unless defined(&__LDBL_REDIR_NTH);
	    eval 'sub __LDBL_REDIR2_DECL {
	        my($name) = @_;
    		eval q( &extern  &__typeof ( &__$name)  &__$name  &__asm ( &__ASMNAME (\\"__nldbl___\\" $name)););
	    }' unless defined(&__LDBL_REDIR2_DECL);
	    eval 'sub __LDBL_REDIR1_DECL {
	        my($name, $alias) = @_;
    		eval q( &extern  &__typeof ($name) $name  &__asm ( &__ASMNAME ($alias)););
	    }' unless defined(&__LDBL_REDIR1_DECL);
	    eval 'sub __LDBL_REDIR_DECL {
	        my($name) = @_;
    		eval q( &extern  &__typeof ($name) $name  &__asm ( &__ASMNAME (\\"__nldbl_\\" $name)););
	    }' unless defined(&__LDBL_REDIR_DECL);
	    eval 'sub __REDIRECT_LDBL {
	        my($name, $proto, $alias) = @_;
    		eval q( &__LDBL_REDIR1 ($name, $proto,  &__nldbl_$alias));
	    }' unless defined(&__REDIRECT_LDBL);
	    eval 'sub __REDIRECT_NTH_LDBL {
	        my($name, $proto, $alias) = @_;
    		eval q( &__LDBL_REDIR1_NTH ($name, $proto,  &__nldbl_$alias));
	    }' unless defined(&__REDIRECT_NTH_LDBL);
	}
    }
    if((!defined (&__LDBL_COMPAT)  && (defined(&__LDOUBLE_REDIRECTS_TO_FLOAT128_ABI) ? &__LDOUBLE_REDIRECTS_TO_FLOAT128_ABI : undef) == 0) || !defined (&__REDIRECT)) {
	eval 'sub __LDBL_REDIR1 {
	    my($name, $proto, $alias) = @_;
    	    eval q($name $proto);
	}' unless defined(&__LDBL_REDIR1);
	eval 'sub __LDBL_REDIR {
	    my($name, $proto) = @_;
    	    eval q($name $proto);
	}' unless defined(&__LDBL_REDIR);
	eval 'sub __LDBL_REDIR1_NTH {
	    my($name, $proto, $alias) = @_;
    	    eval q($name $proto  &__THROW);
	}' unless defined(&__LDBL_REDIR1_NTH);
	eval 'sub __LDBL_REDIR_NTH {
	    my($name, $proto) = @_;
    	    eval q($name $proto  &__THROW);
	}' unless defined(&__LDBL_REDIR_NTH);
	eval 'sub __LDBL_REDIR2_DECL {
	    my($name) = @_;
    	    eval q();
	}' unless defined(&__LDBL_REDIR2_DECL);
	eval 'sub __LDBL_REDIR_DECL {
	    my($name) = @_;
    	    eval q();
	}' unless defined(&__LDBL_REDIR_DECL);
	if(defined(&__REDIRECT)) {
	    eval 'sub __REDIRECT_LDBL {
	        my($name, $proto, $alias) = @_;
    		eval q( &__REDIRECT ($name, $proto, $alias));
	    }' unless defined(&__REDIRECT_LDBL);
	    eval 'sub __REDIRECT_NTH_LDBL {
	        my($name, $proto, $alias) = @_;
    		eval q( &__REDIRECT_NTH ($name, $proto, $alias));
	    }' unless defined(&__REDIRECT_NTH_LDBL);
	}
    }
    if( &__GNUC_PREREQ (4,8) ||  &__glibc_clang_prereq (3,5)) {
	eval 'sub __glibc_macro_warning1 {
	    my($message) = @_;
    	    eval q( &_Pragma ($message));
	}' unless defined(&__glibc_macro_warning1);
	eval 'sub __glibc_macro_warning {
	    my($message) = @_;
    	    eval q( &__glibc_macro_warning1 ( &GCC  &warning $message));
	}' unless defined(&__glibc_macro_warning);
    } else {
	eval 'sub __glibc_macro_warning {
	    my($msg) = @_;
    	    eval q();
	}' unless defined(&__glibc_macro_warning);
    }
    if(!defined (&__cplusplus)  && ( &__GNUC_PREREQ (4, 9) ||  &__glibc_has_extension ((defined(&c_generic_selections) ? &c_generic_selections : undef)) || (!defined (&__GNUC__)  && defined (&__STDC_VERSION__)  && (defined(&__STDC_VERSION__) ? &__STDC_VERSION__ : undef) >= 201112))) {
	eval 'sub __HAVE_GENERIC_SELECTION () {1;}' unless defined(&__HAVE_GENERIC_SELECTION);
    } else {
	eval 'sub __HAVE_GENERIC_SELECTION () {0;}' unless defined(&__HAVE_GENERIC_SELECTION);
    }
    if( &__GNUC_PREREQ (10, 0)) {
	eval 'sub __attr_access {
	    my($x) = @_;
    	    eval q( &__attribute__ (( &__access__ $x)));
	}' unless defined(&__attr_access);
	if((defined(&__USE_FORTIFY_LEVEL) ? &__USE_FORTIFY_LEVEL : undef) == 3) {
	    eval 'sub __fortified_attr_access {
	        my($a, $o, $s) = @_;
    		eval q( &__attribute__ (( &__access__ ($a, $o))));
	    }' unless defined(&__fortified_attr_access);
	} else {
	    eval 'sub __fortified_attr_access {
	        my($a, $o, $s) = @_;
    		eval q( &__attr_access (($a, $o, $s)));
	    }' unless defined(&__fortified_attr_access);
	}
	if( &__GNUC_PREREQ (11, 0)) {
	    eval 'sub __attr_access_none {
	        my($argno) = @_;
    		eval q( &__attribute__ (( &__access__ ( &__none__, $argno))));
	    }' unless defined(&__attr_access_none);
	} else {
	    eval 'sub __attr_access_none {
	        my($argno) = @_;
    		eval q();
	    }' unless defined(&__attr_access_none);
	}
    } else {
	eval 'sub __fortified_attr_access {
	    my($a, $o, $s) = @_;
    	    eval q();
	}' unless defined(&__fortified_attr_access);
	eval 'sub __attr_access {
	    my($x) = @_;
    	    eval q();
	}' unless defined(&__attr_access);
	eval 'sub __attr_access_none {
	    my($argno) = @_;
    	    eval q();
	}' unless defined(&__attr_access_none);
    }
    if( &__GNUC_PREREQ (11, 0)) {
	eval 'sub __attr_dealloc {
	    my($dealloc, $argno) = @_;
    	    eval q( &__attribute__ (( &__malloc__ ($dealloc, $argno))));
	}' unless defined(&__attr_dealloc);
	eval 'sub __attr_dealloc_free () { &__attr_dealloc ( &__builtin_free, 1);}' unless defined(&__attr_dealloc_free);
    } else {
	eval 'sub __attr_dealloc {
	    my($dealloc, $argno) = @_;
    	    eval q();
	}' unless defined(&__attr_dealloc);
	eval 'sub __attr_dealloc_free () {1;}' unless defined(&__attr_dealloc_free);
    }
    if( &__GNUC_PREREQ (4, 1)) {
	eval 'sub __attribute_returns_twice__ () { &__attribute__ (( &__returns_twice__));}' unless defined(&__attribute_returns_twice__);
    } else {
	eval 'sub __attribute_returns_twice__ () {1;}' unless defined(&__attribute_returns_twice__);
    }
}
1;
