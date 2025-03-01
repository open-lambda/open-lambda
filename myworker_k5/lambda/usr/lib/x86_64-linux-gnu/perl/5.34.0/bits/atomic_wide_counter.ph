require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_BITS_ATOMIC_WIDE_COUNTER_H)) {
    eval 'sub _BITS_ATOMIC_WIDE_COUNTER_H () {1;}' unless defined(&_BITS_ATOMIC_WIDE_COUNTER_H);
}
1;
