require '_h2ph_pre.ph';

no warnings qw(redefine misc);

if(!defined (&_SYS_WAIT_H)  && !defined (&_STDLIB_H)) {
    die("Never include <bits/waitflags.h> directly; use <sys/wait.h> instead.");
}
eval 'sub WNOHANG () {1;}' unless defined(&WNOHANG);
eval 'sub WUNTRACED () {2;}' unless defined(&WUNTRACED);
if(defined (&__USE_XOPEN_EXTENDED) || defined (&__USE_XOPEN2K8)) {
    eval 'sub WSTOPPED () {2;}' unless defined(&WSTOPPED);
    eval 'sub WEXITED () {4;}' unless defined(&WEXITED);
    eval 'sub WCONTINUED () {8;}' unless defined(&WCONTINUED);
    eval 'sub WNOWAIT () {0x1000000;}' unless defined(&WNOWAIT);
}
eval 'sub __WNOTHREAD () {0x20000000;}' unless defined(&__WNOTHREAD);
eval 'sub __WALL () {0x40000000;}' unless defined(&__WALL);
eval 'sub __WCLONE () {0x80000000;}' unless defined(&__WCLONE);
1;
