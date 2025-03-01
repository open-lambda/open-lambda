require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_BITS_TYPES_H)) {
    die("Never include <bits/time64.h> directly; use <sys/types.h> instead.");
}
unless(defined(&_BITS_TIME64_H)) {
    eval 'sub _BITS_TIME64_H () {1;}' unless defined(&_BITS_TIME64_H);
    if((defined(&__TIMESIZE) ? &__TIMESIZE : undef) == 64) {
	eval 'sub __TIME64_T_TYPE () { &__TIME_T_TYPE;}' unless defined(&__TIME64_T_TYPE);
    } else {
	eval 'sub __TIME64_T_TYPE () { &__SQUAD_TYPE;}' unless defined(&__TIME64_T_TYPE);
    }
}
1;
