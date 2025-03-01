require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_SYS_SELECT_H)) {
    die("Never use <bits/select.h> directly; include <sys/select.h> instead.");
}
unless(defined(&__FD_ZERO)) {
    sub __FD_ZERO {
	my($s) = @_;
	eval q( &do { 'unsigned int __i';  &fd_set * &__arr = ($s);  &for ( &__i = 0;  &__i < $sizeof{ &fd_set} / $sizeof{ &__fd_mask}; ++ &__i)  &__FDS_BITS ( &__arr)[ &__i] = 0; }  &while (0));
    }
}
unless(defined(&__FD_SET)) {
    sub __FD_SET {
	my($d, $s) = @_;
	eval q((( &void) ( &__FDS_BITS ($s)[ &__FD_ELT($d)] |=  &__FD_MASK($d))));
    }
}
unless(defined(&__FD_CLR)) {
    sub __FD_CLR {
	my($d, $s) = @_;
	eval q((( &void) ( &__FDS_BITS ($s)[ &__FD_ELT($d)] &= ~ &__FD_MASK($d))));
    }
}
unless(defined(&__FD_ISSET)) {
    sub __FD_ISSET {
	my($d, $s) = @_;
	eval q((( &__FDS_BITS ($s)[ &__FD_ELT ($d)] &  &__FD_MASK ($d)) != 0));
    }
}
1;
