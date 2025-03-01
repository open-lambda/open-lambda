require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_SYS_SYSLOG_H)) {
    die("Never include <bits/syslog.h> directly; use <sys/syslog.h> instead.");
}
if(defined(&__va_arg_pack)) {
}
 elsif(!defined (&__cplusplus)) {
    eval 'sub syslog () {( &pri, ...)  &__syslog_chk ( &pri,  &__USE_FORTIFY_LEVEL - 1,  &__VA_ARGS__);}' unless defined(&syslog);
}
if(defined(&__USE_MISC)) {
}
1;
