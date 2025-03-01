require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_LINUX_STDDEF_H)) {
    eval 'sub _LINUX_STDDEF_H () {1;}' unless defined(&_LINUX_STDDEF_H);
    unless(defined(&__always_inline)) {
	eval 'sub __always_inline () { &__inline__;}' unless defined(&__always_inline);
    }
    eval 'sub __struct_group () {( &TAG,  &NAME,  &ATTRS,  &MEMBERS...) \'union union\' { 1; 1; };}' unless defined(&__struct_group);
    eval 'sub __DECLARE_FLEX_ARRAY {
        my($TYPE, $NAME) = @_;
	    eval q(1; $TYPE $NAME->[]; });
    }' unless defined(&__DECLARE_FLEX_ARRAY);
}
1;
