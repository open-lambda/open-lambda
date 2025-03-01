require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_SYS_SYSLOG_H)) {
    die("Never include <bits/syslog-ldbl.h> directly; use <sys/syslog.h> instead.");
}
if(defined(&__USE_MISC)) {
}
if((defined(&__USE_FORTIFY_LEVEL) ? &__USE_FORTIFY_LEVEL : undef) > 0 && defined (&__fortify_function)) {
    if(defined(&__USE_MISC)) {
    }
}
1;
