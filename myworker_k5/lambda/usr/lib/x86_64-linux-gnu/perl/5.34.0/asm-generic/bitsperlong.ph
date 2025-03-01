require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&__ASM_GENERIC_BITS_PER_LONG)) {
    eval 'sub __ASM_GENERIC_BITS_PER_LONG () {1;}' unless defined(&__ASM_GENERIC_BITS_PER_LONG);
    unless(defined(&__BITS_PER_LONG)) {
	eval 'sub __BITS_PER_LONG () {32;}' unless defined(&__BITS_PER_LONG);
    }
}
1;
