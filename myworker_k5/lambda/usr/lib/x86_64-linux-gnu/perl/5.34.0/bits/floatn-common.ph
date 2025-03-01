require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_BITS_FLOATN_COMMON_H)) {
    eval 'sub _BITS_FLOATN_COMMON_H () {1;}' unless defined(&_BITS_FLOATN_COMMON_H);
    require 'features.ph';
    require 'bits/long-double.ph';
    eval 'sub __HAVE_FLOAT16 () {0;}' unless defined(&__HAVE_FLOAT16);
    eval 'sub __HAVE_FLOAT32 () {1;}' unless defined(&__HAVE_FLOAT32);
    eval 'sub __HAVE_FLOAT64 () {1;}' unless defined(&__HAVE_FLOAT64);
    eval 'sub __HAVE_FLOAT32X () {1;}' unless defined(&__HAVE_FLOAT32X);
    eval 'sub __HAVE_FLOAT128X () {0;}' unless defined(&__HAVE_FLOAT128X);
    eval 'sub __HAVE_DISTINCT_FLOAT16 () { &__HAVE_FLOAT16;}' unless defined(&__HAVE_DISTINCT_FLOAT16);
    eval 'sub __HAVE_DISTINCT_FLOAT32 () {0;}' unless defined(&__HAVE_DISTINCT_FLOAT32);
    eval 'sub __HAVE_DISTINCT_FLOAT64 () {0;}' unless defined(&__HAVE_DISTINCT_FLOAT64);
    eval 'sub __HAVE_DISTINCT_FLOAT32X () {0;}' unless defined(&__HAVE_DISTINCT_FLOAT32X);
    eval 'sub __HAVE_DISTINCT_FLOAT64X () {0;}' unless defined(&__HAVE_DISTINCT_FLOAT64X);
    eval 'sub __HAVE_DISTINCT_FLOAT128X () { &__HAVE_FLOAT128X;}' unless defined(&__HAVE_DISTINCT_FLOAT128X);
    eval 'sub __HAVE_FLOAT128_UNLIKE_LDBL () {( &__HAVE_DISTINCT_FLOAT128  &&  &__LDBL_MANT_DIG__ != 113);}' unless defined(&__HAVE_FLOAT128_UNLIKE_LDBL);
    if( &__GNUC_PREREQ (7, 0)  && !defined (&__cplusplus)) {
	eval 'sub __HAVE_FLOATN_NOT_TYPEDEF () {1;}' unless defined(&__HAVE_FLOATN_NOT_TYPEDEF);
    } else {
	eval 'sub __HAVE_FLOATN_NOT_TYPEDEF () {0;}' unless defined(&__HAVE_FLOATN_NOT_TYPEDEF);
    }
    unless(defined(&__ASSEMBLER__)) {
	if((defined(&__HAVE_FLOAT16) ? &__HAVE_FLOAT16 : undef)) {
	    if(! &__GNUC_PREREQ (7, 0) || defined (&__cplusplus)) {
		eval 'sub __f16 {
		    my($x) = @_;
    		    eval q((( &_Float16) $x &f));
		}' unless defined(&__f16);
	    } else {
		eval 'sub __f16 {
		    my($x) = @_;
    		    eval q($x &f16);
		}' unless defined(&__f16);
	    }
	}
	if((defined(&__HAVE_FLOAT32) ? &__HAVE_FLOAT32 : undef)) {
	    if(! &__GNUC_PREREQ (7, 0) || defined (&__cplusplus)) {
		eval 'sub __f32 {
		    my($x) = @_;
    		    eval q($x &f);
		}' unless defined(&__f32);
	    } else {
		eval 'sub __f32 {
		    my($x) = @_;
    		    eval q($x &f32);
		}' unless defined(&__f32);
	    }
	}
	if((defined(&__HAVE_FLOAT64) ? &__HAVE_FLOAT64 : undef)) {
	    if(! &__GNUC_PREREQ (7, 0) || defined (&__cplusplus)) {
		if(defined(&__NO_LONG_DOUBLE_MATH)) {
		    eval 'sub __f64 {
		        my($x) = @_;
    			eval q($x &l);
		    }' unless defined(&__f64);
		} else {
		    eval 'sub __f64 {
		        my($x) = @_;
    			eval q($x);
		    }' unless defined(&__f64);
		}
	    } else {
		eval 'sub __f64 {
		    my($x) = @_;
    		    eval q($x &f64);
		}' unless defined(&__f64);
	    }
	}
	if((defined(&__HAVE_FLOAT32X) ? &__HAVE_FLOAT32X : undef)) {
	    if(! &__GNUC_PREREQ (7, 0) || defined (&__cplusplus)) {
		eval 'sub __f32x {
		    my($x) = @_;
    		    eval q($x);
		}' unless defined(&__f32x);
	    } else {
		eval 'sub __f32x {
		    my($x) = @_;
    		    eval q($x &f32x);
		}' unless defined(&__f32x);
	    }
	}
	if((defined(&__HAVE_FLOAT64X) ? &__HAVE_FLOAT64X : undef)) {
	    if(! &__GNUC_PREREQ (7, 0) || defined (&__cplusplus)) {
		if((defined(&__HAVE_FLOAT64X_LONG_DOUBLE) ? &__HAVE_FLOAT64X_LONG_DOUBLE : undef)) {
		    eval 'sub __f64x {
		        my($x) = @_;
    			eval q($x &l);
		    }' unless defined(&__f64x);
		} else {
		    eval 'sub __f64x {
		        my($x) = @_;
    			eval q( &__f128 ($x));
		    }' unless defined(&__f64x);
		}
	    } else {
		eval 'sub __f64x {
		    my($x) = @_;
    		    eval q($x &f64x);
		}' unless defined(&__f64x);
	    }
	}
	if((defined(&__HAVE_FLOAT128X) ? &__HAVE_FLOAT128X : undef)) {
	    if(! &__GNUC_PREREQ (7, 0) || defined (&__cplusplus)) {
		die("_Float128X supported but no constant suffix");
	    } else {
		eval 'sub __f128x {
		    my($x) = @_;
    		    eval q($x &f128x);
		}' unless defined(&__f128x);
	    }
	}
	if((defined(&__HAVE_FLOAT16) ? &__HAVE_FLOAT16 : undef)) {
	    if(! &__GNUC_PREREQ (7, 0) || defined (&__cplusplus)) {
		eval 'sub __CFLOAT16 () { &__cfloat16;}' unless defined(&__CFLOAT16);
	    } else {
		eval 'sub __CFLOAT16 () { &_Complex  &_Float16;}' unless defined(&__CFLOAT16);
	    }
	}
	if((defined(&__HAVE_FLOAT32) ? &__HAVE_FLOAT32 : undef)) {
	    if(! &__GNUC_PREREQ (7, 0) || defined (&__cplusplus)) {
		eval 'sub __CFLOAT32 () { &_Complex \'float\';}' unless defined(&__CFLOAT32);
	    } else {
		eval 'sub __CFLOAT32 () { &_Complex  &_Float32;}' unless defined(&__CFLOAT32);
	    }
	}
	if((defined(&__HAVE_FLOAT64) ? &__HAVE_FLOAT64 : undef)) {
	    if(! &__GNUC_PREREQ (7, 0) || defined (&__cplusplus)) {
		if(defined(&__NO_LONG_DOUBLE_MATH)) {
		    eval 'sub __CFLOAT64 () { &_Complex \'long double\';}' unless defined(&__CFLOAT64);
		} else {
		    eval 'sub __CFLOAT64 () { &_Complex \'double\';}' unless defined(&__CFLOAT64);
		}
	    } else {
		eval 'sub __CFLOAT64 () { &_Complex  &_Float64;}' unless defined(&__CFLOAT64);
	    }
	}
	if((defined(&__HAVE_FLOAT32X) ? &__HAVE_FLOAT32X : undef)) {
	    if(! &__GNUC_PREREQ (7, 0) || defined (&__cplusplus)) {
		eval 'sub __CFLOAT32X () { &_Complex \'double\';}' unless defined(&__CFLOAT32X);
	    } else {
		eval 'sub __CFLOAT32X () { &_Complex  &_Float32x;}' unless defined(&__CFLOAT32X);
	    }
	}
	if((defined(&__HAVE_FLOAT64X) ? &__HAVE_FLOAT64X : undef)) {
	    if(! &__GNUC_PREREQ (7, 0) || defined (&__cplusplus)) {
		if((defined(&__HAVE_FLOAT64X_LONG_DOUBLE) ? &__HAVE_FLOAT64X_LONG_DOUBLE : undef)) {
		    eval 'sub __CFLOAT64X () { &_Complex \'long double\';}' unless defined(&__CFLOAT64X);
		} else {
		    eval 'sub __CFLOAT64X () { &__CFLOAT128;}' unless defined(&__CFLOAT64X);
		}
	    } else {
		eval 'sub __CFLOAT64X () { &_Complex  &_Float64x;}' unless defined(&__CFLOAT64X);
	    }
	}
	if((defined(&__HAVE_FLOAT128X) ? &__HAVE_FLOAT128X : undef)) {
	    if(! &__GNUC_PREREQ (7, 0) || defined (&__cplusplus)) {
		die("_Float128X supported but no complex type");
	    } else {
		eval 'sub __CFLOAT128X () { &_Complex  &_Float128x;}' unless defined(&__CFLOAT128X);
	    }
	}
	if((defined(&__HAVE_FLOAT16) ? &__HAVE_FLOAT16 : undef)) {
	    if(! &__GNUC_PREREQ (7, 0) || defined (&__cplusplus)) {
	    }
	    if(! &__GNUC_PREREQ (7, 0)) {
		eval 'sub __builtin_huge_valf16 () {
		    eval q((( &_Float16)  &__builtin_huge_val ()));
		}' unless defined(&__builtin_huge_valf16);
		eval 'sub __builtin_inff16 () {
		    eval q((( &_Float16)  &__builtin_inf ()));
		}' unless defined(&__builtin_inff16);
		eval 'sub __builtin_nanf16 {
		    my($x) = @_;
    		    eval q((( &_Float16)  &__builtin_nan ($x)));
		}' unless defined(&__builtin_nanf16);
		eval 'sub __builtin_nansf16 {
		    my($x) = @_;
    		    eval q((( &_Float16)  &__builtin_nans ($x)));
		}' unless defined(&__builtin_nansf16);
	    }
	}
	if((defined(&__HAVE_FLOAT32) ? &__HAVE_FLOAT32 : undef)) {
	    if(! &__GNUC_PREREQ (7, 0) || defined (&__cplusplus)) {
	    }
	    if(! &__GNUC_PREREQ (7, 0)) {
		eval 'sub __builtin_huge_valf32 () {
		    eval q(( &__builtin_huge_valf ()));
		}' unless defined(&__builtin_huge_valf32);
		eval 'sub __builtin_inff32 () {
		    eval q(( &__builtin_inff ()));
		}' unless defined(&__builtin_inff32);
		eval 'sub __builtin_nanf32 {
		    my($x) = @_;
    		    eval q(( &__builtin_nanf ($x)));
		}' unless defined(&__builtin_nanf32);
		eval 'sub __builtin_nansf32 {
		    my($x) = @_;
    		    eval q(( &__builtin_nansf ($x)));
		}' unless defined(&__builtin_nansf32);
	    }
	}
	if((defined(&__HAVE_FLOAT64) ? &__HAVE_FLOAT64 : undef)) {
	    if(defined(&__NO_LONG_DOUBLE_MATH)) {
		if(! &__GNUC_PREREQ (7, 0) || defined (&__cplusplus)) {
		}
		if(! &__GNUC_PREREQ (7, 0)) {
		    eval 'sub __builtin_huge_valf64 () {
		        eval q(( &__builtin_huge_vall ()));
		    }' unless defined(&__builtin_huge_valf64);
		    eval 'sub __builtin_inff64 () {
		        eval q(( &__builtin_infl ()));
		    }' unless defined(&__builtin_inff64);
		    eval 'sub __builtin_nanf64 {
		        my($x) = @_;
    			eval q(( &__builtin_nanl ($x)));
		    }' unless defined(&__builtin_nanf64);
		    eval 'sub __builtin_nansf64 {
		        my($x) = @_;
    			eval q(( &__builtin_nansl ($x)));
		    }' unless defined(&__builtin_nansf64);
		}
	    } else {
		if(! &__GNUC_PREREQ (7, 0) || defined (&__cplusplus)) {
		}
		if(! &__GNUC_PREREQ (7, 0)) {
		    eval 'sub __builtin_huge_valf64 () {
		        eval q(( &__builtin_huge_val ()));
		    }' unless defined(&__builtin_huge_valf64);
		    eval 'sub __builtin_inff64 () {
		        eval q(( &__builtin_inf ()));
		    }' unless defined(&__builtin_inff64);
		    eval 'sub __builtin_nanf64 {
		        my($x) = @_;
    			eval q(( &__builtin_nan ($x)));
		    }' unless defined(&__builtin_nanf64);
		    eval 'sub __builtin_nansf64 {
		        my($x) = @_;
    			eval q(( &__builtin_nans ($x)));
		    }' unless defined(&__builtin_nansf64);
		}
	    }
	}
	if((defined(&__HAVE_FLOAT32X) ? &__HAVE_FLOAT32X : undef)) {
	    if(! &__GNUC_PREREQ (7, 0) || defined (&__cplusplus)) {
	    }
	    if(! &__GNUC_PREREQ (7, 0)) {
		eval 'sub __builtin_huge_valf32x () {
		    eval q(( &__builtin_huge_val ()));
		}' unless defined(&__builtin_huge_valf32x);
		eval 'sub __builtin_inff32x () {
		    eval q(( &__builtin_inf ()));
		}' unless defined(&__builtin_inff32x);
		eval 'sub __builtin_nanf32x {
		    my($x) = @_;
    		    eval q(( &__builtin_nan ($x)));
		}' unless defined(&__builtin_nanf32x);
		eval 'sub __builtin_nansf32x {
		    my($x) = @_;
    		    eval q(( &__builtin_nans ($x)));
		}' unless defined(&__builtin_nansf32x);
	    }
	}
	if((defined(&__HAVE_FLOAT64X) ? &__HAVE_FLOAT64X : undef)) {
	    if((defined(&__HAVE_FLOAT64X_LONG_DOUBLE) ? &__HAVE_FLOAT64X_LONG_DOUBLE : undef)) {
		if(! &__GNUC_PREREQ (7, 0) || defined (&__cplusplus)) {
		}
		if(! &__GNUC_PREREQ (7, 0)) {
		    eval 'sub __builtin_huge_valf64x () {
		        eval q(( &__builtin_huge_vall ()));
		    }' unless defined(&__builtin_huge_valf64x);
		    eval 'sub __builtin_inff64x () {
		        eval q(( &__builtin_infl ()));
		    }' unless defined(&__builtin_inff64x);
		    eval 'sub __builtin_nanf64x {
		        my($x) = @_;
    			eval q(( &__builtin_nanl ($x)));
		    }' unless defined(&__builtin_nanf64x);
		    eval 'sub __builtin_nansf64x {
		        my($x) = @_;
    			eval q(( &__builtin_nansl ($x)));
		    }' unless defined(&__builtin_nansf64x);
		}
	    } else {
		if(! &__GNUC_PREREQ (7, 0) || defined (&__cplusplus)) {
		}
		if(! &__GNUC_PREREQ (7, 0)) {
		    eval 'sub __builtin_huge_valf64x () {
		        eval q(( &__builtin_huge_valf128 ()));
		    }' unless defined(&__builtin_huge_valf64x);
		    eval 'sub __builtin_inff64x () {
		        eval q(( &__builtin_inff128 ()));
		    }' unless defined(&__builtin_inff64x);
		    eval 'sub __builtin_nanf64x {
		        my($x) = @_;
    			eval q(( &__builtin_nanf128 ($x)));
		    }' unless defined(&__builtin_nanf64x);
		    eval 'sub __builtin_nansf64x {
		        my($x) = @_;
    			eval q(( &__builtin_nansf128 ($x)));
		    }' unless defined(&__builtin_nansf64x);
		}
	    }
	}
	if((defined(&__HAVE_FLOAT128X) ? &__HAVE_FLOAT128X : undef)) {
	    if(! &__GNUC_PREREQ (7, 0) || defined (&__cplusplus)) {
		die("_Float128x supported but no type");
	    }
	    if(! &__GNUC_PREREQ (7, 0)) {
		eval 'sub __builtin_huge_valf128x () {
		    eval q((( &_Float128x)  &__builtin_huge_val ()));
		}' unless defined(&__builtin_huge_valf128x);
		eval 'sub __builtin_inff128x () {
		    eval q((( &_Float128x)  &__builtin_inf ()));
		}' unless defined(&__builtin_inff128x);
		eval 'sub __builtin_nanf128x {
		    my($x) = @_;
    		    eval q((( &_Float128x)  &__builtin_nan ($x)));
		}' unless defined(&__builtin_nanf128x);
		eval 'sub __builtin_nansf128x {
		    my($x) = @_;
    		    eval q((( &_Float128x)  &__builtin_nans ($x)));
		}' unless defined(&__builtin_nansf128x);
	    }
	}
    }
}
1;
