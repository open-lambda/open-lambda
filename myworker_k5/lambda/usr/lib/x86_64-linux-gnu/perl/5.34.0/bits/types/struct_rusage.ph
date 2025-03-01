require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&__rusage_defined)) {
    eval 'sub __rusage_defined () {1;}' unless defined(&__rusage_defined);
    require 'bits/types.ph';
    require 'bits/types/struct_timeval.ph';
}
1;
