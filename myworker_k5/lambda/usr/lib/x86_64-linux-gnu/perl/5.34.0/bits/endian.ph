require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_BITS_ENDIAN_H)) {
    eval 'sub _BITS_ENDIAN_H () {1;}' unless defined(&_BITS_ENDIAN_H);
    eval 'sub __LITTLE_ENDIAN () {1234;}' unless defined(&__LITTLE_ENDIAN);
    eval 'sub __BIG_ENDIAN () {4321;}' unless defined(&__BIG_ENDIAN);
    eval 'sub __PDP_ENDIAN () {3412;}' unless defined(&__PDP_ENDIAN);
    require 'bits/endianness.ph';
    unless(defined(&__FLOAT_WORD_ORDER)) {
	eval 'sub __FLOAT_WORD_ORDER () { &__BYTE_ORDER;}' unless defined(&__FLOAT_WORD_ORDER);
    }
    if((defined(&__BYTE_ORDER) ? &__BYTE_ORDER : undef) == (defined(&__LITTLE_ENDIAN) ? &__LITTLE_ENDIAN : undef)) {
	eval 'sub __LONG_LONG_PAIR {
	    my($HI, $LO) = @_;
    	    eval q($LO, $HI);
	}' unless defined(&__LONG_LONG_PAIR);
    }
 elsif((defined(&__BYTE_ORDER) ? &__BYTE_ORDER : undef) == (defined(&__BIG_ENDIAN) ? &__BIG_ENDIAN : undef)) {
	eval 'sub __LONG_LONG_PAIR {
	    my($HI, $LO) = @_;
    	    eval q($HI, $LO);
	}' unless defined(&__LONG_LONG_PAIR);
    }
}
1;
