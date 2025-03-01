require '_h2ph_pre.ph';

no warnings qw(redefine misc);

if(defined(&_LIBC)) {
    die("Applications\ may\ not\ define\ the\ macro\ _LIBC");
}
eval 'sub __stub___compat_bdflush () {1;}' unless defined(&__stub___compat_bdflush);
eval 'sub __stub_chflags () {1;}' unless defined(&__stub_chflags);
eval 'sub __stub_fchflags () {1;}' unless defined(&__stub_fchflags);
eval 'sub __stub_gtty () {1;}' unless defined(&__stub_gtty);
eval 'sub __stub_revoke () {1;}' unless defined(&__stub_revoke);
eval 'sub __stub_setlogin () {1;}' unless defined(&__stub_setlogin);
eval 'sub __stub_sigreturn () {1;}' unless defined(&__stub_sigreturn);
eval 'sub __stub_stty () {1;}' unless defined(&__stub_stty);
1;
