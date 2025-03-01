require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&__clock_t_defined)) {
    eval 'sub __clock_t_defined () {1;}' unless defined(&__clock_t_defined);
    require 'bits/types.ph';
}
1;
