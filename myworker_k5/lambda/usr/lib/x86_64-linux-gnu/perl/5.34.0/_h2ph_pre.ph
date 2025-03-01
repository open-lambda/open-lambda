# This file was created by h2ph version 4
no warnings qw(portable);
unless (defined &_FILE_OFFSET_BITS) { sub _FILE_OFFSET_BITS() { 64 } }

unless (defined &_FORTIFY_SOURCE) { sub _FORTIFY_SOURCE() { 2 } }

unless (defined &_GNU_SOURCE) { sub _GNU_SOURCE() { 1 } }

unless (defined &_LARGEFILE64_SOURCE) { sub _LARGEFILE64_SOURCE() { 1 } }

unless (defined &_LARGEFILE_SOURCE) { sub _LARGEFILE_SOURCE() { 1 } }

unless (defined &_LP64) { sub _LP64() { 1 } }

unless (defined &_POSIX_C_SOURCE) { sub _POSIX_C_SOURCE() { 200809 } }

unless (defined &_POSIX_SOURCE) { sub _POSIX_SOURCE() { 1 } }

unless (defined &_REENTRANT) { sub _REENTRANT() { 1 } }

unless (defined &_STDC_PREDEF_H) { sub _STDC_PREDEF_H() { 1 } }

unless (defined &_XOPEN_SOURCE) { sub _XOPEN_SOURCE() { 700 } }

unless (defined &_XOPEN_SOURCE_EXTENDED) { sub _XOPEN_SOURCE_EXTENDED() { 1 } }

unless (defined &__ATOMIC_ACQUIRE) { sub __ATOMIC_ACQUIRE() { 2 } }

unless (defined &__ATOMIC_ACQ_REL) { sub __ATOMIC_ACQ_REL() { 4 } }

unless (defined &__ATOMIC_CONSUME) { sub __ATOMIC_CONSUME() { 1 } }

unless (defined &__ATOMIC_HLE_ACQUIRE) { sub __ATOMIC_HLE_ACQUIRE() { 65536 } }

unless (defined &__ATOMIC_HLE_RELEASE) { sub __ATOMIC_HLE_RELEASE() { 131072 } }

unless (defined &__ATOMIC_RELAXED) { sub __ATOMIC_RELAXED() { 0 } }

unless (defined &__ATOMIC_RELEASE) { sub __ATOMIC_RELEASE() { 3 } }

unless (defined &__ATOMIC_SEQ_CST) { sub __ATOMIC_SEQ_CST() { 5 } }

unless (defined &__BIGGEST_ALIGNMENT__) { sub __BIGGEST_ALIGNMENT__() { 16 } }

unless (defined &__BYTE_ORDER__) { sub __BYTE_ORDER__() { 1234 } }

unless (defined &__CET__) { sub __CET__() { 3 } }

unless (defined &__CHAR16_TYPE__) { sub __CHAR16_TYPE__() { "short\\\ unsigned\\\ int" } }

unless (defined &__CHAR32_TYPE__) { sub __CHAR32_TYPE__() { "unsigned\\\ int" } }

unless (defined &__CHAR_BIT__) { sub __CHAR_BIT__() { 8 } }

unless (defined &__DBL_DECIMAL_DIG__) { sub __DBL_DECIMAL_DIG__() { 17 } }

unless (defined &__DBL_DENORM_MIN__) { sub __DBL_DENORM_MIN__() { "\(double\)4\.94065645841246544176568792868221372e\-324L" } }

unless (defined &__DBL_DIG__) { sub __DBL_DIG__() { 15 } }

unless (defined &__DBL_EPSILON__) { sub __DBL_EPSILON__() { "\(double\)2\.22044604925031308084726333618164062e\-16L" } }

unless (defined &__DBL_HAS_DENORM__) { sub __DBL_HAS_DENORM__() { 1 } }

unless (defined &__DBL_HAS_INFINITY__) { sub __DBL_HAS_INFINITY__() { 1 } }

unless (defined &__DBL_HAS_QUIET_NAN__) { sub __DBL_HAS_QUIET_NAN__() { 1 } }

unless (defined &__DBL_IS_IEC_60559__) { sub __DBL_IS_IEC_60559__() { 2 } }

unless (defined &__DBL_MANT_DIG__) { sub __DBL_MANT_DIG__() { 53 } }

unless (defined &__DBL_MAX_10_EXP__) { sub __DBL_MAX_10_EXP__() { 308 } }

unless (defined &__DBL_MAX_EXP__) { sub __DBL_MAX_EXP__() { 1024 } }

unless (defined &__DBL_MAX__) { sub __DBL_MAX__() { "\(double\)1\.79769313486231570814527423731704357e\+308L" } }

unless (defined &__DBL_MIN_10_EXP__) { sub __DBL_MIN_10_EXP__() { -307 } }

unless (defined &__DBL_MIN_EXP__) { sub __DBL_MIN_EXP__() { -1021 } }

unless (defined &__DBL_MIN__) { sub __DBL_MIN__() { "\(double\)2\.22507385850720138309023271733240406e\-308L" } }

unless (defined &__DBL_NORM_MAX__) { sub __DBL_NORM_MAX__() { "\(double\)1\.79769313486231570814527423731704357e\+308L" } }

unless (defined &__DEC128_EPSILON__) { sub __DEC128_EPSILON__() { "1E\-33DL" } }

unless (defined &__DEC128_MANT_DIG__) { sub __DEC128_MANT_DIG__() { 34 } }

unless (defined &__DEC128_MAX_EXP__) { sub __DEC128_MAX_EXP__() { 6145 } }

unless (defined &__DEC128_MAX__) { sub __DEC128_MAX__() { "9\.999999999999999999999999999999999E6144DL" } }

unless (defined &__DEC128_MIN_EXP__) { sub __DEC128_MIN_EXP__() { -6142 } }

unless (defined &__DEC128_MIN__) { sub __DEC128_MIN__() { "1E\-6143DL" } }

unless (defined &__DEC128_SUBNORMAL_MIN__) { sub __DEC128_SUBNORMAL_MIN__() { "0\.000000000000000000000000000000001E\-6143DL" } }

unless (defined &__DEC32_EPSILON__) { sub __DEC32_EPSILON__() { "1E\-6DF" } }

unless (defined &__DEC32_MANT_DIG__) { sub __DEC32_MANT_DIG__() { 7 } }

unless (defined &__DEC32_MAX_EXP__) { sub __DEC32_MAX_EXP__() { 97 } }

unless (defined &__DEC32_MAX__) { sub __DEC32_MAX__() { "9\.999999E96DF" } }

unless (defined &__DEC32_MIN_EXP__) { sub __DEC32_MIN_EXP__() { -94 } }

unless (defined &__DEC32_MIN__) { sub __DEC32_MIN__() { "1E\-95DF" } }

unless (defined &__DEC32_SUBNORMAL_MIN__) { sub __DEC32_SUBNORMAL_MIN__() { "0\.000001E\-95DF" } }

unless (defined &__DEC64_EPSILON__) { sub __DEC64_EPSILON__() { "1E\-15DD" } }

unless (defined &__DEC64_MANT_DIG__) { sub __DEC64_MANT_DIG__() { 16 } }

unless (defined &__DEC64_MAX_EXP__) { sub __DEC64_MAX_EXP__() { 385 } }

unless (defined &__DEC64_MAX__) { sub __DEC64_MAX__() { "9\.999999999999999E384DD" } }

unless (defined &__DEC64_MIN_EXP__) { sub __DEC64_MIN_EXP__() { -382 } }

unless (defined &__DEC64_MIN__) { sub __DEC64_MIN__() { "1E\-383DD" } }

unless (defined &__DEC64_SUBNORMAL_MIN__) { sub __DEC64_SUBNORMAL_MIN__() { "0\.000000000000001E\-383DD" } }

unless (defined &__DECIMAL_BID_FORMAT__) { sub __DECIMAL_BID_FORMAT__() { 1 } }

unless (defined &__DECIMAL_DIG__) { sub __DECIMAL_DIG__() { 21 } }

unless (defined &__DEC_EVAL_METHOD__) { sub __DEC_EVAL_METHOD__() { 2 } }

unless (defined &__ELF__) { sub __ELF__() { 1 } }

unless (defined &__FINITE_MATH_ONLY__) { sub __FINITE_MATH_ONLY__() { 0 } }

unless (defined &__FLOAT_WORD_ORDER__) { sub __FLOAT_WORD_ORDER__() { 1234 } }

unless (defined &__FLT128_DECIMAL_DIG__) { sub __FLT128_DECIMAL_DIG__() { 36 } }

unless (defined &__FLT128_DENORM_MIN__) { sub __FLT128_DENORM_MIN__() { "6\.47517511943802511092443895822764655e\-4966F128" } }

unless (defined &__FLT128_DIG__) { sub __FLT128_DIG__() { 33 } }

unless (defined &__FLT128_EPSILON__) { sub __FLT128_EPSILON__() { "1\.92592994438723585305597794258492732e\-34F128" } }

unless (defined &__FLT128_HAS_DENORM__) { sub __FLT128_HAS_DENORM__() { 1 } }

unless (defined &__FLT128_HAS_INFINITY__) { sub __FLT128_HAS_INFINITY__() { 1 } }

unless (defined &__FLT128_HAS_QUIET_NAN__) { sub __FLT128_HAS_QUIET_NAN__() { 1 } }

unless (defined &__FLT128_IS_IEC_60559__) { sub __FLT128_IS_IEC_60559__() { 2 } }

unless (defined &__FLT128_MANT_DIG__) { sub __FLT128_MANT_DIG__() { 113 } }

unless (defined &__FLT128_MAX_10_EXP__) { sub __FLT128_MAX_10_EXP__() { 4932 } }

unless (defined &__FLT128_MAX_EXP__) { sub __FLT128_MAX_EXP__() { 16384 } }

unless (defined &__FLT128_MAX__) { sub __FLT128_MAX__() { "1\.18973149535723176508575932662800702e\+4932F128" } }

unless (defined &__FLT128_MIN_10_EXP__) { sub __FLT128_MIN_10_EXP__() { -4931 } }

unless (defined &__FLT128_MIN_EXP__) { sub __FLT128_MIN_EXP__() { -16381 } }

unless (defined &__FLT128_MIN__) { sub __FLT128_MIN__() { "3\.36210314311209350626267781732175260e\-4932F128" } }

unless (defined &__FLT128_NORM_MAX__) { sub __FLT128_NORM_MAX__() { "1\.18973149535723176508575932662800702e\+4932F128" } }

unless (defined &__FLT32X_DECIMAL_DIG__) { sub __FLT32X_DECIMAL_DIG__() { 17 } }

unless (defined &__FLT32X_DENORM_MIN__) { sub __FLT32X_DENORM_MIN__() { "4\.94065645841246544176568792868221372e\-324F32x" } }

unless (defined &__FLT32X_DIG__) { sub __FLT32X_DIG__() { 15 } }

unless (defined &__FLT32X_EPSILON__) { sub __FLT32X_EPSILON__() { "2\.22044604925031308084726333618164062e\-16F32x" } }

unless (defined &__FLT32X_HAS_DENORM__) { sub __FLT32X_HAS_DENORM__() { 1 } }

unless (defined &__FLT32X_HAS_INFINITY__) { sub __FLT32X_HAS_INFINITY__() { 1 } }

unless (defined &__FLT32X_HAS_QUIET_NAN__) { sub __FLT32X_HAS_QUIET_NAN__() { 1 } }

unless (defined &__FLT32X_IS_IEC_60559__) { sub __FLT32X_IS_IEC_60559__() { 2 } }

unless (defined &__FLT32X_MANT_DIG__) { sub __FLT32X_MANT_DIG__() { 53 } }

unless (defined &__FLT32X_MAX_10_EXP__) { sub __FLT32X_MAX_10_EXP__() { 308 } }

unless (defined &__FLT32X_MAX_EXP__) { sub __FLT32X_MAX_EXP__() { 1024 } }

unless (defined &__FLT32X_MAX__) { sub __FLT32X_MAX__() { "1\.79769313486231570814527423731704357e\+308F32x" } }

unless (defined &__FLT32X_MIN_10_EXP__) { sub __FLT32X_MIN_10_EXP__() { -307 } }

unless (defined &__FLT32X_MIN_EXP__) { sub __FLT32X_MIN_EXP__() { -1021 } }

unless (defined &__FLT32X_MIN__) { sub __FLT32X_MIN__() { "2\.22507385850720138309023271733240406e\-308F32x" } }

unless (defined &__FLT32X_NORM_MAX__) { sub __FLT32X_NORM_MAX__() { "1\.79769313486231570814527423731704357e\+308F32x" } }

unless (defined &__FLT32_DECIMAL_DIG__) { sub __FLT32_DECIMAL_DIG__() { 9 } }

unless (defined &__FLT32_DENORM_MIN__) { sub __FLT32_DENORM_MIN__() { "1\.40129846432481707092372958328991613e\-45F32" } }

unless (defined &__FLT32_DIG__) { sub __FLT32_DIG__() { 6 } }

unless (defined &__FLT32_EPSILON__) { sub __FLT32_EPSILON__() { "1\.19209289550781250000000000000000000e\-7F32" } }

unless (defined &__FLT32_HAS_DENORM__) { sub __FLT32_HAS_DENORM__() { 1 } }

unless (defined &__FLT32_HAS_INFINITY__) { sub __FLT32_HAS_INFINITY__() { 1 } }

unless (defined &__FLT32_HAS_QUIET_NAN__) { sub __FLT32_HAS_QUIET_NAN__() { 1 } }

unless (defined &__FLT32_IS_IEC_60559__) { sub __FLT32_IS_IEC_60559__() { 2 } }

unless (defined &__FLT32_MANT_DIG__) { sub __FLT32_MANT_DIG__() { 24 } }

unless (defined &__FLT32_MAX_10_EXP__) { sub __FLT32_MAX_10_EXP__() { 38 } }

unless (defined &__FLT32_MAX_EXP__) { sub __FLT32_MAX_EXP__() { 128 } }

unless (defined &__FLT32_MAX__) { sub __FLT32_MAX__() { "3\.40282346638528859811704183484516925e\+38F32" } }

unless (defined &__FLT32_MIN_10_EXP__) { sub __FLT32_MIN_10_EXP__() { -37 } }

unless (defined &__FLT32_MIN_EXP__) { sub __FLT32_MIN_EXP__() { -125 } }

unless (defined &__FLT32_MIN__) { sub __FLT32_MIN__() { "1\.17549435082228750796873653722224568e\-38F32" } }

unless (defined &__FLT32_NORM_MAX__) { sub __FLT32_NORM_MAX__() { "3\.40282346638528859811704183484516925e\+38F32" } }

unless (defined &__FLT64X_DECIMAL_DIG__) { sub __FLT64X_DECIMAL_DIG__() { 21 } }

unless (defined &__FLT64X_DENORM_MIN__) { sub __FLT64X_DENORM_MIN__() { "3\.64519953188247460252840593361941982e\-4951F64x" } }

unless (defined &__FLT64X_DIG__) { sub __FLT64X_DIG__() { 18 } }

unless (defined &__FLT64X_EPSILON__) { sub __FLT64X_EPSILON__() { "1\.08420217248550443400745280086994171e\-19F64x" } }

unless (defined &__FLT64X_HAS_DENORM__) { sub __FLT64X_HAS_DENORM__() { 1 } }

unless (defined &__FLT64X_HAS_INFINITY__) { sub __FLT64X_HAS_INFINITY__() { 1 } }

unless (defined &__FLT64X_HAS_QUIET_NAN__) { sub __FLT64X_HAS_QUIET_NAN__() { 1 } }

unless (defined &__FLT64X_IS_IEC_60559__) { sub __FLT64X_IS_IEC_60559__() { 2 } }

unless (defined &__FLT64X_MANT_DIG__) { sub __FLT64X_MANT_DIG__() { 64 } }

unless (defined &__FLT64X_MAX_10_EXP__) { sub __FLT64X_MAX_10_EXP__() { 4932 } }

unless (defined &__FLT64X_MAX_EXP__) { sub __FLT64X_MAX_EXP__() { 16384 } }

unless (defined &__FLT64X_MAX__) { sub __FLT64X_MAX__() { "1\.18973149535723176502126385303097021e\+4932F64x" } }

unless (defined &__FLT64X_MIN_10_EXP__) { sub __FLT64X_MIN_10_EXP__() { -4931 } }

unless (defined &__FLT64X_MIN_EXP__) { sub __FLT64X_MIN_EXP__() { -16381 } }

unless (defined &__FLT64X_MIN__) { sub __FLT64X_MIN__() { "3\.36210314311209350626267781732175260e\-4932F64x" } }

unless (defined &__FLT64X_NORM_MAX__) { sub __FLT64X_NORM_MAX__() { "1\.18973149535723176502126385303097021e\+4932F64x" } }

unless (defined &__FLT64_DECIMAL_DIG__) { sub __FLT64_DECIMAL_DIG__() { 17 } }

unless (defined &__FLT64_DENORM_MIN__) { sub __FLT64_DENORM_MIN__() { "4\.94065645841246544176568792868221372e\-324F64" } }

unless (defined &__FLT64_DIG__) { sub __FLT64_DIG__() { 15 } }

unless (defined &__FLT64_EPSILON__) { sub __FLT64_EPSILON__() { "2\.22044604925031308084726333618164062e\-16F64" } }

unless (defined &__FLT64_HAS_DENORM__) { sub __FLT64_HAS_DENORM__() { 1 } }

unless (defined &__FLT64_HAS_INFINITY__) { sub __FLT64_HAS_INFINITY__() { 1 } }

unless (defined &__FLT64_HAS_QUIET_NAN__) { sub __FLT64_HAS_QUIET_NAN__() { 1 } }

unless (defined &__FLT64_IS_IEC_60559__) { sub __FLT64_IS_IEC_60559__() { 2 } }

unless (defined &__FLT64_MANT_DIG__) { sub __FLT64_MANT_DIG__() { 53 } }

unless (defined &__FLT64_MAX_10_EXP__) { sub __FLT64_MAX_10_EXP__() { 308 } }

unless (defined &__FLT64_MAX_EXP__) { sub __FLT64_MAX_EXP__() { 1024 } }

unless (defined &__FLT64_MAX__) { sub __FLT64_MAX__() { "1\.79769313486231570814527423731704357e\+308F64" } }

unless (defined &__FLT64_MIN_10_EXP__) { sub __FLT64_MIN_10_EXP__() { -307 } }

unless (defined &__FLT64_MIN_EXP__) { sub __FLT64_MIN_EXP__() { -1021 } }

unless (defined &__FLT64_MIN__) { sub __FLT64_MIN__() { "2\.22507385850720138309023271733240406e\-308F64" } }

unless (defined &__FLT64_NORM_MAX__) { sub __FLT64_NORM_MAX__() { "1\.79769313486231570814527423731704357e\+308F64" } }

unless (defined &__FLT_DECIMAL_DIG__) { sub __FLT_DECIMAL_DIG__() { 9 } }

unless (defined &__FLT_DENORM_MIN__) { sub __FLT_DENORM_MIN__() { 1.40129846432481707092372958328991613e-45 } }

unless (defined &__FLT_DIG__) { sub __FLT_DIG__() { 6 } }

unless (defined &__FLT_EPSILON__) { sub __FLT_EPSILON__() { 1.19209289550781250000000000000000000e-7 } }

unless (defined &__FLT_EVAL_METHOD_TS_18661_3__) { sub __FLT_EVAL_METHOD_TS_18661_3__() { 0 } }

unless (defined &__FLT_EVAL_METHOD__) { sub __FLT_EVAL_METHOD__() { 0 } }

unless (defined &__FLT_HAS_DENORM__) { sub __FLT_HAS_DENORM__() { 1 } }

unless (defined &__FLT_HAS_INFINITY__) { sub __FLT_HAS_INFINITY__() { 1 } }

unless (defined &__FLT_HAS_QUIET_NAN__) { sub __FLT_HAS_QUIET_NAN__() { 1 } }

unless (defined &__FLT_IS_IEC_60559__) { sub __FLT_IS_IEC_60559__() { 2 } }

unless (defined &__FLT_MANT_DIG__) { sub __FLT_MANT_DIG__() { 24 } }

unless (defined &__FLT_MAX_10_EXP__) { sub __FLT_MAX_10_EXP__() { 38 } }

unless (defined &__FLT_MAX_EXP__) { sub __FLT_MAX_EXP__() { 128 } }

unless (defined &__FLT_MAX__) { sub __FLT_MAX__() { 3.40282346638528859811704183484516925e+38 } }

unless (defined &__FLT_MIN_10_EXP__) { sub __FLT_MIN_10_EXP__() { -37 } }

unless (defined &__FLT_MIN_EXP__) { sub __FLT_MIN_EXP__() { -125 } }

unless (defined &__FLT_MIN__) { sub __FLT_MIN__() { 1.17549435082228750796873653722224568e-38 } }

unless (defined &__FLT_NORM_MAX__) { sub __FLT_NORM_MAX__() { 3.40282346638528859811704183484516925e+38 } }

unless (defined &__FLT_RADIX__) { sub __FLT_RADIX__() { 2 } }

unless (defined &__FXSR__) { sub __FXSR__() { 1 } }

unless (defined &__GCC_ASM_FLAG_OUTPUTS__) { sub __GCC_ASM_FLAG_OUTPUTS__() { 1 } }

unless (defined &__GCC_ATOMIC_BOOL_LOCK_FREE) { sub __GCC_ATOMIC_BOOL_LOCK_FREE() { 2 } }

unless (defined &__GCC_ATOMIC_CHAR16_T_LOCK_FREE) { sub __GCC_ATOMIC_CHAR16_T_LOCK_FREE() { 2 } }

unless (defined &__GCC_ATOMIC_CHAR32_T_LOCK_FREE) { sub __GCC_ATOMIC_CHAR32_T_LOCK_FREE() { 2 } }

unless (defined &__GCC_ATOMIC_CHAR_LOCK_FREE) { sub __GCC_ATOMIC_CHAR_LOCK_FREE() { 2 } }

unless (defined &__GCC_ATOMIC_INT_LOCK_FREE) { sub __GCC_ATOMIC_INT_LOCK_FREE() { 2 } }

unless (defined &__GCC_ATOMIC_LLONG_LOCK_FREE) { sub __GCC_ATOMIC_LLONG_LOCK_FREE() { 2 } }

unless (defined &__GCC_ATOMIC_LONG_LOCK_FREE) { sub __GCC_ATOMIC_LONG_LOCK_FREE() { 2 } }

unless (defined &__GCC_ATOMIC_POINTER_LOCK_FREE) { sub __GCC_ATOMIC_POINTER_LOCK_FREE() { 2 } }

unless (defined &__GCC_ATOMIC_SHORT_LOCK_FREE) { sub __GCC_ATOMIC_SHORT_LOCK_FREE() { 2 } }

unless (defined &__GCC_ATOMIC_TEST_AND_SET_TRUEVAL) { sub __GCC_ATOMIC_TEST_AND_SET_TRUEVAL() { 1 } }

unless (defined &__GCC_ATOMIC_WCHAR_T_LOCK_FREE) { sub __GCC_ATOMIC_WCHAR_T_LOCK_FREE() { 2 } }

unless (defined &__GCC_HAVE_DWARF2_CFI_ASM) { sub __GCC_HAVE_DWARF2_CFI_ASM() { 1 } }

unless (defined &__GCC_HAVE_SYNC_COMPARE_AND_SWAP_1) { sub __GCC_HAVE_SYNC_COMPARE_AND_SWAP_1() { 1 } }

unless (defined &__GCC_HAVE_SYNC_COMPARE_AND_SWAP_2) { sub __GCC_HAVE_SYNC_COMPARE_AND_SWAP_2() { 1 } }

unless (defined &__GCC_HAVE_SYNC_COMPARE_AND_SWAP_4) { sub __GCC_HAVE_SYNC_COMPARE_AND_SWAP_4() { 1 } }

unless (defined &__GCC_HAVE_SYNC_COMPARE_AND_SWAP_8) { sub __GCC_HAVE_SYNC_COMPARE_AND_SWAP_8() { 1 } }

unless (defined &__GCC_IEC_559) { sub __GCC_IEC_559() { 2 } }

unless (defined &__GCC_IEC_559_COMPLEX) { sub __GCC_IEC_559_COMPLEX() { 2 } }

unless (defined &__GLIBC_MINOR__) { sub __GLIBC_MINOR__() { 35 } }

unless (defined &__GLIBC__) { sub __GLIBC__() { 2 } }

unless (defined &__GNUC_EXECUTION_CHARSET_NAME) { sub __GNUC_EXECUTION_CHARSET_NAME() { "\"UTF\-8\"" } }

unless (defined &__GNUC_MINOR__) { sub __GNUC_MINOR__() { 3 } }

unless (defined &__GNUC_PATCHLEVEL__) { sub __GNUC_PATCHLEVEL__() { 0 } }

unless (defined &__GNUC_STDC_INLINE__) { sub __GNUC_STDC_INLINE__() { 1 } }

unless (defined &__GNUC_WIDE_EXECUTION_CHARSET_NAME) { sub __GNUC_WIDE_EXECUTION_CHARSET_NAME() { "\"UTF\-32LE\"" } }

unless (defined &__GNUC__) { sub __GNUC__() { 11 } }

unless (defined &__GNU_LIBRARY__) { sub __GNU_LIBRARY__() { 6 } }

unless (defined &__GXX_ABI_VERSION) { sub __GXX_ABI_VERSION() { 1016 } }

unless (defined &__HAVE_SPECULATION_SAFE_VALUE) { sub __HAVE_SPECULATION_SAFE_VALUE() { 1 } }

unless (defined &__INT16_C) { sub __INT16_C() { &__INT16_C } }

unless (defined &__INT16_MAX__) { sub __INT16_MAX__() { 0x7fff } }

unless (defined &__INT16_TYPE__) { sub __INT16_TYPE__() { "short\\\ int" } }

unless (defined &__INT32_C) { sub __INT32_C() { &__INT32_C } }

unless (defined &__INT32_MAX__) { sub __INT32_MAX__() { 0x7fffffff } }

unless (defined &__INT32_TYPE__) { sub __INT32_TYPE__() { "int" } }

unless (defined &__INT64_C) { sub __INT64_C() { &__INT64_C } }

unless (defined &__INT64_MAX__) { sub __INT64_MAX__() { hex('0x7fffffffffffffff') } }

unless (defined &__INT64_TYPE__) { sub __INT64_TYPE__() { "long\\\ int" } }

unless (defined &__INT8_C) { sub __INT8_C() { &__INT8_C } }

unless (defined &__INT8_MAX__) { sub __INT8_MAX__() { 0x7f } }

unless (defined &__INT8_TYPE__) { sub __INT8_TYPE__() { "signed\\\ char" } }

unless (defined &__INTMAX_C) { sub __INTMAX_C() { &__INTMAX_C } }

unless (defined &__INTMAX_MAX__) { sub __INTMAX_MAX__() { hex('0x7fffffffffffffff') } }

unless (defined &__INTMAX_TYPE__) { sub __INTMAX_TYPE__() { "long\\\ int" } }

unless (defined &__INTMAX_WIDTH__) { sub __INTMAX_WIDTH__() { 64 } }

unless (defined &__INTPTR_MAX__) { sub __INTPTR_MAX__() { hex('0x7fffffffffffffff') } }

unless (defined &__INTPTR_TYPE__) { sub __INTPTR_TYPE__() { "long\\\ int" } }

unless (defined &__INTPTR_WIDTH__) { sub __INTPTR_WIDTH__() { 64 } }

unless (defined &__INT_FAST16_MAX__) { sub __INT_FAST16_MAX__() { hex('0x7fffffffffffffff') } }

unless (defined &__INT_FAST16_TYPE__) { sub __INT_FAST16_TYPE__() { "long\\\ int" } }

unless (defined &__INT_FAST16_WIDTH__) { sub __INT_FAST16_WIDTH__() { 64 } }

unless (defined &__INT_FAST32_MAX__) { sub __INT_FAST32_MAX__() { hex('0x7fffffffffffffff') } }

unless (defined &__INT_FAST32_TYPE__) { sub __INT_FAST32_TYPE__() { "long\\\ int" } }

unless (defined &__INT_FAST32_WIDTH__) { sub __INT_FAST32_WIDTH__() { 64 } }

unless (defined &__INT_FAST64_MAX__) { sub __INT_FAST64_MAX__() { hex('0x7fffffffffffffff') } }

unless (defined &__INT_FAST64_TYPE__) { sub __INT_FAST64_TYPE__() { "long\\\ int" } }

unless (defined &__INT_FAST64_WIDTH__) { sub __INT_FAST64_WIDTH__() { 64 } }

unless (defined &__INT_FAST8_MAX__) { sub __INT_FAST8_MAX__() { 0x7f } }

unless (defined &__INT_FAST8_TYPE__) { sub __INT_FAST8_TYPE__() { "signed\\\ char" } }

unless (defined &__INT_FAST8_WIDTH__) { sub __INT_FAST8_WIDTH__() { 8 } }

unless (defined &__INT_LEAST16_MAX__) { sub __INT_LEAST16_MAX__() { 0x7fff } }

unless (defined &__INT_LEAST16_TYPE__) { sub __INT_LEAST16_TYPE__() { "short\\\ int" } }

unless (defined &__INT_LEAST16_WIDTH__) { sub __INT_LEAST16_WIDTH__() { 16 } }

unless (defined &__INT_LEAST32_MAX__) { sub __INT_LEAST32_MAX__() { 0x7fffffff } }

unless (defined &__INT_LEAST32_TYPE__) { sub __INT_LEAST32_TYPE__() { "int" } }

unless (defined &__INT_LEAST32_WIDTH__) { sub __INT_LEAST32_WIDTH__() { 32 } }

unless (defined &__INT_LEAST64_MAX__) { sub __INT_LEAST64_MAX__() { hex('0x7fffffffffffffff') } }

unless (defined &__INT_LEAST64_TYPE__) { sub __INT_LEAST64_TYPE__() { "long\\\ int" } }

unless (defined &__INT_LEAST64_WIDTH__) { sub __INT_LEAST64_WIDTH__() { 64 } }

unless (defined &__INT_LEAST8_MAX__) { sub __INT_LEAST8_MAX__() { 0x7f } }

unless (defined &__INT_LEAST8_TYPE__) { sub __INT_LEAST8_TYPE__() { "signed\\\ char" } }

unless (defined &__INT_LEAST8_WIDTH__) { sub __INT_LEAST8_WIDTH__() { 8 } }

unless (defined &__INT_MAX__) { sub __INT_MAX__() { 0x7fffffff } }

unless (defined &__INT_WIDTH__) { sub __INT_WIDTH__() { 32 } }

unless (defined &__LDBL_DECIMAL_DIG__) { sub __LDBL_DECIMAL_DIG__() { 21 } }

unless (defined &__LDBL_DENORM_MIN__) { sub __LDBL_DENORM_MIN__() { 3.64519953188247460252840593361941982e-4951 } }

unless (defined &__LDBL_DIG__) { sub __LDBL_DIG__() { 18 } }

unless (defined &__LDBL_EPSILON__) { sub __LDBL_EPSILON__() { 1.08420217248550443400745280086994171e-19 } }

unless (defined &__LDBL_HAS_DENORM__) { sub __LDBL_HAS_DENORM__() { 1 } }

unless (defined &__LDBL_HAS_INFINITY__) { sub __LDBL_HAS_INFINITY__() { 1 } }

unless (defined &__LDBL_HAS_QUIET_NAN__) { sub __LDBL_HAS_QUIET_NAN__() { 1 } }

unless (defined &__LDBL_IS_IEC_60559__) { sub __LDBL_IS_IEC_60559__() { 2 } }

unless (defined &__LDBL_MANT_DIG__) { sub __LDBL_MANT_DIG__() { 64 } }

unless (defined &__LDBL_MAX_10_EXP__) { sub __LDBL_MAX_10_EXP__() { 4932 } }

unless (defined &__LDBL_MAX_EXP__) { sub __LDBL_MAX_EXP__() { 16384 } }

unless (defined &__LDBL_MAX__) { sub __LDBL_MAX__() { 1.18973149535723176502126385303097021e+4932 } }

unless (defined &__LDBL_MIN_10_EXP__) { sub __LDBL_MIN_10_EXP__() { -4931 } }

unless (defined &__LDBL_MIN_EXP__) { sub __LDBL_MIN_EXP__() { -16381 } }

unless (defined &__LDBL_MIN__) { sub __LDBL_MIN__() { 3.36210314311209350626267781732175260e-4932 } }

unless (defined &__LDBL_NORM_MAX__) { sub __LDBL_NORM_MAX__() { 1.18973149535723176502126385303097021e+4932 } }

unless (defined &__LONG_LONG_MAX__) { sub __LONG_LONG_MAX__() { hex('0x7fffffffffffffff') } }

unless (defined &__LONG_LONG_WIDTH__) { sub __LONG_LONG_WIDTH__() { 64 } }

unless (defined &__LONG_MAX__) { sub __LONG_MAX__() { hex('0x7fffffffffffffff') } }

unless (defined &__LONG_WIDTH__) { sub __LONG_WIDTH__() { 64 } }

unless (defined &__LP64__) { sub __LP64__() { 1 } }

unless (defined &__MMX_WITH_SSE__) { sub __MMX_WITH_SSE__() { 1 } }

unless (defined &__MMX__) { sub __MMX__() { 1 } }

unless (defined &__ORDER_BIG_ENDIAN__) { sub __ORDER_BIG_ENDIAN__() { 4321 } }

unless (defined &__ORDER_LITTLE_ENDIAN__) { sub __ORDER_LITTLE_ENDIAN__() { 1234 } }

unless (defined &__ORDER_PDP_ENDIAN__) { sub __ORDER_PDP_ENDIAN__() { 3412 } }

unless (defined &__PIC__) { sub __PIC__() { 2 } }

unless (defined &__PIE__) { sub __PIE__() { 2 } }

unless (defined &__PRAGMA_REDEFINE_EXTNAME) { sub __PRAGMA_REDEFINE_EXTNAME() { 1 } }

unless (defined &__PTRDIFF_MAX__) { sub __PTRDIFF_MAX__() { hex('0x7fffffffffffffff') } }

unless (defined &__PTRDIFF_TYPE__) { sub __PTRDIFF_TYPE__() { "long\\\ int" } }

unless (defined &__PTRDIFF_WIDTH__) { sub __PTRDIFF_WIDTH__() { 64 } }

unless (defined &__SCHAR_MAX__) { sub __SCHAR_MAX__() { 0x7f } }

unless (defined &__SCHAR_WIDTH__) { sub __SCHAR_WIDTH__() { 8 } }

unless (defined &__SEG_FS) { sub __SEG_FS() { 1 } }

unless (defined &__SEG_GS) { sub __SEG_GS() { 1 } }

unless (defined &__SHRT_MAX__) { sub __SHRT_MAX__() { 0x7fff } }

unless (defined &__SHRT_WIDTH__) { sub __SHRT_WIDTH__() { 16 } }

unless (defined &__SIG_ATOMIC_MAX__) { sub __SIG_ATOMIC_MAX__() { 0x7fffffff } }

unless (defined &__SIG_ATOMIC_MIN__) { sub __SIG_ATOMIC_MIN__() { "\-0x7fffffff\\\ \-\\\ 1" } }

unless (defined &__SIG_ATOMIC_TYPE__) { sub __SIG_ATOMIC_TYPE__() { "int" } }

unless (defined &__SIG_ATOMIC_WIDTH__) { sub __SIG_ATOMIC_WIDTH__() { 32 } }

unless (defined &__SIZEOF_DOUBLE__) { sub __SIZEOF_DOUBLE__() { 8 } }

unless (defined &__SIZEOF_FLOAT128__) { sub __SIZEOF_FLOAT128__() { 16 } }

unless (defined &__SIZEOF_FLOAT80__) { sub __SIZEOF_FLOAT80__() { 16 } }

unless (defined &__SIZEOF_FLOAT__) { sub __SIZEOF_FLOAT__() { 4 } }

unless (defined &__SIZEOF_INT128__) { sub __SIZEOF_INT128__() { 16 } }

unless (defined &__SIZEOF_INT__) { sub __SIZEOF_INT__() { 4 } }

unless (defined &__SIZEOF_LONG_DOUBLE__) { sub __SIZEOF_LONG_DOUBLE__() { 16 } }

unless (defined &__SIZEOF_LONG_LONG__) { sub __SIZEOF_LONG_LONG__() { 8 } }

unless (defined &__SIZEOF_LONG__) { sub __SIZEOF_LONG__() { 8 } }

unless (defined &__SIZEOF_POINTER__) { sub __SIZEOF_POINTER__() { 8 } }

unless (defined &__SIZEOF_PTRDIFF_T__) { sub __SIZEOF_PTRDIFF_T__() { 8 } }

unless (defined &__SIZEOF_SHORT__) { sub __SIZEOF_SHORT__() { 2 } }

unless (defined &__SIZEOF_SIZE_T__) { sub __SIZEOF_SIZE_T__() { 8 } }

unless (defined &__SIZEOF_WCHAR_T__) { sub __SIZEOF_WCHAR_T__() { 4 } }

unless (defined &__SIZEOF_WINT_T__) { sub __SIZEOF_WINT_T__() { 4 } }

unless (defined &__SIZE_MAX__) { sub __SIZE_MAX__() { hex('0xffffffffffffffff') } }

unless (defined &__SIZE_TYPE__) { sub __SIZE_TYPE__() { "long\\\ unsigned\\\ int" } }

unless (defined &__SIZE_WIDTH__) { sub __SIZE_WIDTH__() { 64 } }

unless (defined &__SSE2_MATH__) { sub __SSE2_MATH__() { 1 } }

unless (defined &__SSE2__) { sub __SSE2__() { 1 } }

unless (defined &__SSE_MATH__) { sub __SSE_MATH__() { 1 } }

unless (defined &__SSE__) { sub __SSE__() { 1 } }

unless (defined &__SSP_STRONG__) { sub __SSP_STRONG__() { 3 } }

unless (defined &__STDC_HOSTED__) { sub __STDC_HOSTED__() { 1 } }

unless (defined &__STDC_IEC_559_COMPLEX__) { sub __STDC_IEC_559_COMPLEX__() { 1 } }

unless (defined &__STDC_IEC_559__) { sub __STDC_IEC_559__() { 1 } }

unless (defined &__STDC_IEC_60559_BFP__) { sub __STDC_IEC_60559_BFP__() { 201404 } }

unless (defined &__STDC_IEC_60559_COMPLEX__) { sub __STDC_IEC_60559_COMPLEX__() { 201404 } }

unless (defined &__STDC_ISO_10646__) { sub __STDC_ISO_10646__() { 201706 } }

unless (defined &__STDC_UTF_16__) { sub __STDC_UTF_16__() { 1 } }

unless (defined &__STDC_UTF_32__) { sub __STDC_UTF_32__() { 1 } }

unless (defined &__STDC_VERSION__) { sub __STDC_VERSION__() { 201710 } }

unless (defined &__STDC__) { sub __STDC__() { 1 } }

unless (defined &__UINT16_C) { sub __UINT16_C() { &__UINT16_C } }

unless (defined &__UINT16_MAX__) { sub __UINT16_MAX__() { 0xffff } }

unless (defined &__UINT16_TYPE__) { sub __UINT16_TYPE__() { "short\\\ unsigned\\\ int" } }

unless (defined &__UINT32_C) { sub __UINT32_C() { &__UINT32_C } }

unless (defined &__UINT32_MAX__) { sub __UINT32_MAX__() { 0xffffffff } }

unless (defined &__UINT32_TYPE__) { sub __UINT32_TYPE__() { "unsigned\\\ int" } }

unless (defined &__UINT64_C) { sub __UINT64_C() { &__UINT64_C } }

unless (defined &__UINT64_MAX__) { sub __UINT64_MAX__() { hex('0xffffffffffffffff') } }

unless (defined &__UINT64_TYPE__) { sub __UINT64_TYPE__() { "long\\\ unsigned\\\ int" } }

unless (defined &__UINT8_C) { sub __UINT8_C() { &__UINT8_C } }

unless (defined &__UINT8_MAX__) { sub __UINT8_MAX__() { 0xff } }

unless (defined &__UINT8_TYPE__) { sub __UINT8_TYPE__() { "unsigned\\\ char" } }

unless (defined &__UINTMAX_C) { sub __UINTMAX_C() { &__UINTMAX_C } }

unless (defined &__UINTMAX_MAX__) { sub __UINTMAX_MAX__() { hex('0xffffffffffffffff') } }

unless (defined &__UINTMAX_TYPE__) { sub __UINTMAX_TYPE__() { "long\\\ unsigned\\\ int" } }

unless (defined &__UINTPTR_MAX__) { sub __UINTPTR_MAX__() { hex('0xffffffffffffffff') } }

unless (defined &__UINTPTR_TYPE__) { sub __UINTPTR_TYPE__() { "long\\\ unsigned\\\ int" } }

unless (defined &__UINT_FAST16_MAX__) { sub __UINT_FAST16_MAX__() { hex('0xffffffffffffffff') } }

unless (defined &__UINT_FAST16_TYPE__) { sub __UINT_FAST16_TYPE__() { "long\\\ unsigned\\\ int" } }

unless (defined &__UINT_FAST32_MAX__) { sub __UINT_FAST32_MAX__() { hex('0xffffffffffffffff') } }

unless (defined &__UINT_FAST32_TYPE__) { sub __UINT_FAST32_TYPE__() { "long\\\ unsigned\\\ int" } }

unless (defined &__UINT_FAST64_MAX__) { sub __UINT_FAST64_MAX__() { hex('0xffffffffffffffff') } }

unless (defined &__UINT_FAST64_TYPE__) { sub __UINT_FAST64_TYPE__() { "long\\\ unsigned\\\ int" } }

unless (defined &__UINT_FAST8_MAX__) { sub __UINT_FAST8_MAX__() { 0xff } }

unless (defined &__UINT_FAST8_TYPE__) { sub __UINT_FAST8_TYPE__() { "unsigned\\\ char" } }

unless (defined &__UINT_LEAST16_MAX__) { sub __UINT_LEAST16_MAX__() { 0xffff } }

unless (defined &__UINT_LEAST16_TYPE__) { sub __UINT_LEAST16_TYPE__() { "short\\\ unsigned\\\ int" } }

unless (defined &__UINT_LEAST32_MAX__) { sub __UINT_LEAST32_MAX__() { 0xffffffff } }

unless (defined &__UINT_LEAST32_TYPE__) { sub __UINT_LEAST32_TYPE__() { "unsigned\\\ int" } }

unless (defined &__UINT_LEAST64_MAX__) { sub __UINT_LEAST64_MAX__() { hex('0xffffffffffffffff') } }

unless (defined &__UINT_LEAST64_TYPE__) { sub __UINT_LEAST64_TYPE__() { "long\\\ unsigned\\\ int" } }

unless (defined &__UINT_LEAST8_MAX__) { sub __UINT_LEAST8_MAX__() { 0xff } }

unless (defined &__UINT_LEAST8_TYPE__) { sub __UINT_LEAST8_TYPE__() { "unsigned\\\ char" } }

unless (defined &__USE_FILE_OFFSET64) { sub __USE_FILE_OFFSET64() { 1 } }

unless (defined &__USE_GNU) { sub __USE_GNU() { 1 } }

unless (defined &__USE_LARGEFILE) { sub __USE_LARGEFILE() { 1 } }

unless (defined &__USE_LARGEFILE64) { sub __USE_LARGEFILE64() { 1 } }

unless (defined &__USE_MISC) { sub __USE_MISC() { 1 } }

unless (defined &__USE_POSIX) { sub __USE_POSIX() { 1 } }

unless (defined &__USE_POSIX199309) { sub __USE_POSIX199309() { 1 } }

unless (defined &__USE_POSIX199506) { sub __USE_POSIX199506() { 1 } }

unless (defined &__USE_POSIX2) { sub __USE_POSIX2() { 1 } }

unless (defined &__USE_UNIX98) { sub __USE_UNIX98() { 1 } }

unless (defined &__USE_XOPEN) { sub __USE_XOPEN() { 1 } }

unless (defined &__USE_XOPEN_EXTENDED) { sub __USE_XOPEN_EXTENDED() { 1 } }

unless (defined &__VERSION__) { sub __VERSION__() { "\"11\.3\.0\"" } }

unless (defined &__WCHAR_MAX__) { sub __WCHAR_MAX__() { 0x7fffffff } }

unless (defined &__WCHAR_MIN__) { sub __WCHAR_MIN__() { "\-0x7fffffff\\\ \-\\\ 1" } }

unless (defined &__WCHAR_TYPE__) { sub __WCHAR_TYPE__() { "int" } }

unless (defined &__WCHAR_WIDTH__) { sub __WCHAR_WIDTH__() { 32 } }

unless (defined &__WINT_MAX__) { sub __WINT_MAX__() { 0xffffffff } }

unless (defined &__WINT_MIN__) { sub __WINT_MIN__() { 0 } }

unless (defined &__WINT_TYPE__) { sub __WINT_TYPE__() { "unsigned\\\ int" } }

unless (defined &__WINT_WIDTH__) { sub __WINT_WIDTH__() { 32 } }

unless (defined &__amd64) { sub __amd64() { 1 } }

unless (defined &__amd64__) { sub __amd64__() { 1 } }

unless (defined &__code_model_small__) { sub __code_model_small__() { 1 } }

unless (defined &__gnu_linux__) { sub __gnu_linux__() { 1 } }

unless (defined &__k8) { sub __k8() { 1 } }

unless (defined &__k8__) { sub __k8__() { 1 } }

unless (defined &__linux) { sub __linux() { 1 } }

unless (defined &__linux__) { sub __linux__() { 1 } }

unless (defined &__pic__) { sub __pic__() { 2 } }

unless (defined &__pie__) { sub __pie__() { 2 } }

unless (defined &__unix) { sub __unix() { 1 } }

unless (defined &__unix__) { sub __unix__() { 1 } }

unless (defined &__x86_64) { sub __x86_64() { 1 } }

unless (defined &__x86_64__) { sub __x86_64__() { 1 } }

unless (defined &linux) { sub linux() { 1 } }

unless (defined &unix) { sub unix() { 1 } }


1;
