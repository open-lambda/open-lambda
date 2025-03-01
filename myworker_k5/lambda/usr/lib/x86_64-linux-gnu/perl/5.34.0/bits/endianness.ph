require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_BITS_ENDIANNESS_H)) {
    eval 'sub _BITS_ENDIANNESS_H () {1;}' unless defined(&_BITS_ENDIANNESS_H);
    unless(defined(&_BITS_ENDIAN_H)) {
	die("Never use <bits/endianness.h> directly; include <endian.h> instead.");
    }
    eval 'sub __BYTE_ORDER () { &__LITTLE_ENDIAN;}' unless defined(&__BYTE_ORDER);
}
1;
