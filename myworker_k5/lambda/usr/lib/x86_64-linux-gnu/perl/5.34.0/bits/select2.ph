require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_SYS_SELECT_H)) {
    die("Never include <bits/select2.h> directly; use <sys/select.h> instead.");
}
undef(&__FD_ELT) if defined(&__FD_ELT);
unless(defined(&__FD_ELT)) {
    sub __FD_ELT {
	my($d) = @_;
	eval q( &__extension__ ({ 'long int __d' = ($d); ( &__builtin_constant_p ( &__d) ? (0<=  &__d  &&  &__d <  &__FD_SETSIZE ? ( &__d /  &__NFDBITS) :  &__fdelt_warn ( &__d)) :  &__fdelt_chk ( &__d)); }));
    }
}
1;
