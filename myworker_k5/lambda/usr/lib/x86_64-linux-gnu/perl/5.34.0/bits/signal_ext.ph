require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_SIGNAL_H)) {
    die("Never include <bits/signal_ext.h> directly; use <signal.h> instead.");
}
if(defined(&__USE_GNU)) {
}
1;
