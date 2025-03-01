require '_h2ph_pre.ph';

no warnings qw(redefine misc);

eval 'sub __LDOUBLE_REDIRECTS_TO_FLOAT128_ABI () {0;}' unless defined(&__LDOUBLE_REDIRECTS_TO_FLOAT128_ABI);
1;
