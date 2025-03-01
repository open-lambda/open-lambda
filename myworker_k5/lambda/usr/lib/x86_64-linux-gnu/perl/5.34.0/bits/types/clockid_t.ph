require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&__clockid_t_defined)) {
    eval 'sub __clockid_t_defined () {1;}' unless defined(&__clockid_t_defined);
    require 'bits/types.ph';
}
1;
