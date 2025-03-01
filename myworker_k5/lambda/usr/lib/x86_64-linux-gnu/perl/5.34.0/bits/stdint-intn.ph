require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_BITS_STDINT_INTN_H)) {
    eval 'sub _BITS_STDINT_INTN_H () {1;}' unless defined(&_BITS_STDINT_INTN_H);
    require 'bits/types.ph';
}
1;
