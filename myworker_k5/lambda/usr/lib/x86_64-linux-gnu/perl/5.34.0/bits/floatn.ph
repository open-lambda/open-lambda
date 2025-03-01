require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_BITS_FLOATN_H)) {
    eval 'sub _BITS_FLOATN_H () {1;}' unless defined(&_BITS_FLOATN_H);
    require 'features.ph';
    if((defined (&__x86_64__) ?  &__GNUC_PREREQ (4, 3) : (defined (&__GNU__) ?  &__GNUC_PREREQ (4, 5) :  &__GNUC_PREREQ (4, 4)))) {
	eval 'sub __HAVE_FLOAT128 () {1;}' unless defined(&__HAVE_FLOAT128);
    } else {
	eval 'sub __HAVE_FLOAT128 () {0;}' unless defined(&__HAVE_FLOAT128);
    }
    if((defined(&__HAVE_FLOAT128) ? &__HAVE_FLOAT128 : undef)) {
	eval 'sub __HAVE_DISTINCT_FLOAT128 () {1;}' unless defined(&__HAVE_DISTINCT_FLOAT128);
    } else {
	eval 'sub __HAVE_DISTINCT_FLOAT128 () {0;}' unless defined(&__HAVE_DISTINCT_FLOAT128);
    }
    eval 'sub __HAVE_FLOAT64X () {1;}' unless defined(&__HAVE_FLOAT64X);
    eval 'sub __HAVE_FLOAT64X_LONG_DOUBLE () {1;}' unless defined(&__HAVE_FLOAT64X_LONG_DOUBLE);
    unless(defined(&__ASSEMBLER__)) {
	if((defined(&__HAVE_FLOAT128) ? &__HAVE_FLOAT128 : undef)) {
	    if(! &__GNUC_PREREQ (7, 0) || defined (&__cplusplus)) {
		eval 'sub __f128 {
		    my($x) = @_;
    		    eval q($x &q);
		}' unless defined(&__f128);
	    } else {
		eval 'sub __f128 {
		    my($x) = @_;
    		    eval q($x &f128);
		}' unless defined(&__f128);
	    }
	}
	if((defined(&__HAVE_FLOAT128) ? &__HAVE_FLOAT128 : undef)) {
	    if(! &__GNUC_PREREQ (7, 0) || defined (&__cplusplus)) {
		eval 'sub __CFLOAT128 () { &__cfloat128;}' unless defined(&__CFLOAT128);
	    } else {
		eval 'sub __CFLOAT128 () { &_Complex  &_Float128;}' unless defined(&__CFLOAT128);
	    }
	}
	if((defined(&__HAVE_FLOAT128) ? &__HAVE_FLOAT128 : undef)) {
	    if(! &__GNUC_PREREQ (7, 0) || defined (&__cplusplus)) {
	    }
	    if(! &__GNUC_PREREQ (7, 0)) {
		eval 'sub __builtin_huge_valf128 () {
		    eval q((( &_Float128)  &__builtin_huge_val ()));
		}' unless defined(&__builtin_huge_valf128);
	    }
	    if(! &__GNUC_PREREQ (7, 0)) {
		eval 'sub __builtin_copysignf128 () { &__builtin_copysignq;}' unless defined(&__builtin_copysignf128);
		eval 'sub __builtin_fabsf128 () { &__builtin_fabsq;}' unless defined(&__builtin_fabsf128);
		eval 'sub __builtin_inff128 () {
		    eval q((( &_Float128)  &__builtin_inf ()));
		}' unless defined(&__builtin_inff128);
		eval 'sub __builtin_nanf128 {
		    my($x) = @_;
    		    eval q((( &_Float128)  &__builtin_nan ($x)));
		}' unless defined(&__builtin_nanf128);
		eval 'sub __builtin_nansf128 {
		    my($x) = @_;
    		    eval q((( &_Float128)  &__builtin_nans ($x)));
		}' unless defined(&__builtin_nansf128);
	    }
	    if(! &__GNUC_PREREQ (6, 0)) {
		eval 'sub __builtin_signbitf128 () { &__signbitf128;}' unless defined(&__builtin_signbitf128);
	    }
	}
    }
    require 'bits/floatn-common.ph';
}
1;
