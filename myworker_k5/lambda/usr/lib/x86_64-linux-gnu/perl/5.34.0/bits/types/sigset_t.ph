require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&__sigset_t_defined)) {
    eval 'sub __sigset_t_defined () {1;}' unless defined(&__sigset_t_defined);
    require 'bits/types/__sigset_t.ph';
}
1;
