require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&__timeval_defined)) {
    eval 'sub __timeval_defined () {1;}' unless defined(&__timeval_defined);
    require 'bits/types.ph';
    if(defined(&__USE_TIME_BITS64)) {
    } else {
    }
}
1;
