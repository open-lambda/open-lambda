require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&____sigset_t_defined)) {
    eval 'sub ____sigset_t_defined () {1;}' unless defined(&____sigset_t_defined);
    eval 'sub _SIGSET_NWORDS () {(1024/ (8* $sizeof{\'unsigned long int\'}));}' unless defined(&_SIGSET_NWORDS);
}
1;
