require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_ENDIAN_H)) {
    eval 'sub _ENDIAN_H () {1;}' unless defined(&_ENDIAN_H);
    require 'features.ph';
    require 'bits/endian.ph';
    if(defined(&__USE_MISC)) {
	eval 'sub LITTLE_ENDIAN () { &__LITTLE_ENDIAN;}' unless defined(&LITTLE_ENDIAN);
	eval 'sub BIG_ENDIAN () { &__BIG_ENDIAN;}' unless defined(&BIG_ENDIAN);
	eval 'sub PDP_ENDIAN () { &__PDP_ENDIAN;}' unless defined(&PDP_ENDIAN);
	eval 'sub BYTE_ORDER () { &__BYTE_ORDER;}' unless defined(&BYTE_ORDER);
    }
    if(defined (&__USE_MISC)  && !defined (&__ASSEMBLER__)) {
	require 'bits/byteswap.ph';
	require 'bits/uintn-identity.ph';
	if((defined(&__BYTE_ORDER) ? &__BYTE_ORDER : undef) == (defined(&__LITTLE_ENDIAN) ? &__LITTLE_ENDIAN : undef)) {
	    eval 'sub htobe16 {
	        my($x) = @_;
    		eval q( &__bswap_16 ($x));
	    }' unless defined(&htobe16);
	    eval 'sub htole16 {
	        my($x) = @_;
    		eval q( &__uint16_identity ($x));
	    }' unless defined(&htole16);
	    eval 'sub be16toh {
	        my($x) = @_;
    		eval q( &__bswap_16 ($x));
	    }' unless defined(&be16toh);
	    eval 'sub le16toh {
	        my($x) = @_;
    		eval q( &__uint16_identity ($x));
	    }' unless defined(&le16toh);
	    eval 'sub htobe32 {
	        my($x) = @_;
    		eval q( &__bswap_32 ($x));
	    }' unless defined(&htobe32);
	    eval 'sub htole32 {
	        my($x) = @_;
    		eval q( &__uint32_identity ($x));
	    }' unless defined(&htole32);
	    eval 'sub be32toh {
	        my($x) = @_;
    		eval q( &__bswap_32 ($x));
	    }' unless defined(&be32toh);
	    eval 'sub le32toh {
	        my($x) = @_;
    		eval q( &__uint32_identity ($x));
	    }' unless defined(&le32toh);
	    eval 'sub htobe64 {
	        my($x) = @_;
    		eval q( &__bswap_64 ($x));
	    }' unless defined(&htobe64);
	    eval 'sub htole64 {
	        my($x) = @_;
    		eval q( &__uint64_identity ($x));
	    }' unless defined(&htole64);
	    eval 'sub be64toh {
	        my($x) = @_;
    		eval q( &__bswap_64 ($x));
	    }' unless defined(&be64toh);
	    eval 'sub le64toh {
	        my($x) = @_;
    		eval q( &__uint64_identity ($x));
	    }' unless defined(&le64toh);
	} else {
	    eval 'sub htobe16 {
	        my($x) = @_;
    		eval q( &__uint16_identity ($x));
	    }' unless defined(&htobe16);
	    eval 'sub htole16 {
	        my($x) = @_;
    		eval q( &__bswap_16 ($x));
	    }' unless defined(&htole16);
	    eval 'sub be16toh {
	        my($x) = @_;
    		eval q( &__uint16_identity ($x));
	    }' unless defined(&be16toh);
	    eval 'sub le16toh {
	        my($x) = @_;
    		eval q( &__bswap_16 ($x));
	    }' unless defined(&le16toh);
	    eval 'sub htobe32 {
	        my($x) = @_;
    		eval q( &__uint32_identity ($x));
	    }' unless defined(&htobe32);
	    eval 'sub htole32 {
	        my($x) = @_;
    		eval q( &__bswap_32 ($x));
	    }' unless defined(&htole32);
	    eval 'sub be32toh {
	        my($x) = @_;
    		eval q( &__uint32_identity ($x));
	    }' unless defined(&be32toh);
	    eval 'sub le32toh {
	        my($x) = @_;
    		eval q( &__bswap_32 ($x));
	    }' unless defined(&le32toh);
	    eval 'sub htobe64 {
	        my($x) = @_;
    		eval q( &__uint64_identity ($x));
	    }' unless defined(&htobe64);
	    eval 'sub htole64 {
	        my($x) = @_;
    		eval q( &__bswap_64 ($x));
	    }' unless defined(&htole64);
	    eval 'sub be64toh {
	        my($x) = @_;
    		eval q( &__uint64_identity ($x));
	    }' unless defined(&be64toh);
	    eval 'sub le64toh {
	        my($x) = @_;
    		eval q( &__bswap_64 ($x));
	    }' unless defined(&le64toh);
	}
    }
}
1;
