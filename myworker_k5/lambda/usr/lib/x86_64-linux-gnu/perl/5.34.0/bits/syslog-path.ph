require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_SYS_SYSLOG_H)) {
    die("Never include this file directly.  Use <sys/syslog.h> instead");
}
unless(defined(&_BITS_SYSLOG_PATH_H)) {
    eval 'sub _BITS_SYSLOG_PATH_H () {1;}' unless defined(&_BITS_SYSLOG_PATH_H);
    eval 'sub _PATH_LOG () {"/dev/log";}' unless defined(&_PATH_LOG);
}
1;
