require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&____sigval_t_defined)) {
    eval 'sub ____sigval_t_defined () {1;}' unless defined(&____sigval_t_defined);
    if(defined(&__USE_POSIX199309)) {
    } else {
    }
}
1;
