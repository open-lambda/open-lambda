require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&__sigstack_defined)) {
    eval 'sub __sigstack_defined () {1;}' unless defined(&__sigstack_defined);
}
1;
