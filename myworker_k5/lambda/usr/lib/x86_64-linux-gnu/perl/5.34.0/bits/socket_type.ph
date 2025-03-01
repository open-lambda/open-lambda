require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_SYS_SOCKET_H)) {
    die("Never include <bits/socket_type.h> directly; use <sys/socket.h> instead.");
}
eval("sub SOCK_STREAM () { 1; }") unless defined(&SOCK_STREAM);
eval("sub SOCK_DGRAM () { 2; }") unless defined(&SOCK_DGRAM);
eval("sub SOCK_RAW () { 3; }") unless defined(&SOCK_RAW);
eval("sub SOCK_RDM () { 4; }") unless defined(&SOCK_RDM);
eval("sub SOCK_SEQPACKET () { 5; }") unless defined(&SOCK_SEQPACKET);
eval("sub SOCK_DCCP () { 6; }") unless defined(&SOCK_DCCP);
eval("sub SOCK_PACKET () { 10; }") unless defined(&SOCK_PACKET);
eval("sub SOCK_CLOEXEC () { 02000000; }") unless defined(&SOCK_CLOEXEC);
eval("sub SOCK_NONBLOCK () { 00004000; }") unless defined(&SOCK_NONBLOCK);
1;
