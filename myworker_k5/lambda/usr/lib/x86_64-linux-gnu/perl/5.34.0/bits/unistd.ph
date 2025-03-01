require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_UNISTD_H)) {
    die("Never include <bits/unistd.h> directly; use <unistd.h> instead.");
}
if(defined (&__USE_UNIX98) || defined (&__USE_XOPEN2K8)) {
    unless(defined(&__USE_FILE_OFFSET64)) {
    } else {
    }
    if(defined(&__USE_LARGEFILE64)) {
    }
}
if(defined (&__USE_XOPEN_EXTENDED) || defined (&__USE_XOPEN2K)) {
}
if(defined(&__USE_ATFILE)) {
}
if(defined (&__USE_MISC) || defined (&__USE_XOPEN_EXTENDED)) {
}
if(defined(&__USE_POSIX199506)) {
}
if(defined (&__USE_MISC) || defined (&__USE_UNIX98)) {
}
if(defined (&__USE_MISC) || (defined (&__USE_XOPEN)  && !defined (&__USE_UNIX98))) {
}
1;
