require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&__sigval_t_defined)) {
    eval 'sub __sigval_t_defined () {1;}' unless defined(&__sigval_t_defined);
    require 'bits/types/__sigval_t.ph';
    unless(defined(&__USE_POSIX199309)) {
	die("sigval_t defined for standard not including union sigval");
    }
}
1;
