require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_SYS_SOCKET_H)) {
    die("Never include <bits/socket-constants.h> directly; use <sys/socket.h> instead.");
}
require 'bits/timesize.ph';
eval 'sub SOL_SOCKET () {1;}' unless defined(&SOL_SOCKET);
eval 'sub SO_ACCEPTCONN () {30;}' unless defined(&SO_ACCEPTCONN);
eval 'sub SO_BROADCAST () {6;}' unless defined(&SO_BROADCAST);
eval 'sub SO_DONTROUTE () {5;}' unless defined(&SO_DONTROUTE);
eval 'sub SO_ERROR () {4;}' unless defined(&SO_ERROR);
eval 'sub SO_KEEPALIVE () {9;}' unless defined(&SO_KEEPALIVE);
eval 'sub SO_LINGER () {13;}' unless defined(&SO_LINGER);
eval 'sub SO_OOBINLINE () {10;}' unless defined(&SO_OOBINLINE);
eval 'sub SO_RCVBUF () {8;}' unless defined(&SO_RCVBUF);
eval 'sub SO_RCVLOWAT () {18;}' unless defined(&SO_RCVLOWAT);
eval 'sub SO_REUSEADDR () {2;}' unless defined(&SO_REUSEADDR);
eval 'sub SO_SNDBUF () {7;}' unless defined(&SO_SNDBUF);
eval 'sub SO_SNDLOWAT () {19;}' unless defined(&SO_SNDLOWAT);
eval 'sub SO_TYPE () {3;}' unless defined(&SO_TYPE);
if(((defined(&__TIMESIZE) ? &__TIMESIZE : undef) == 64 && (defined(&__WORDSIZE) ? &__WORDSIZE : undef) == 32 && (!defined (&__SYSCALL_WORDSIZE) || (defined(&__SYSCALL_WORDSIZE) ? &__SYSCALL_WORDSIZE : undef) == 32))) {
    eval 'sub SO_RCVTIMEO () {66;}' unless defined(&SO_RCVTIMEO);
    eval 'sub SO_SNDTIMEO () {67;}' unless defined(&SO_SNDTIMEO);
    eval 'sub SO_TIMESTAMP () {63;}' unless defined(&SO_TIMESTAMP);
    eval 'sub SO_TIMESTAMPNS () {64;}' unless defined(&SO_TIMESTAMPNS);
    eval 'sub SO_TIMESTAMPING () {65;}' unless defined(&SO_TIMESTAMPING);
} else {
    if((defined(&__TIMESIZE) ? &__TIMESIZE : undef) == 64) {
	eval 'sub SO_RCVTIMEO () {20;}' unless defined(&SO_RCVTIMEO);
	eval 'sub SO_SNDTIMEO () {21;}' unless defined(&SO_SNDTIMEO);
	eval 'sub SO_TIMESTAMP () {29;}' unless defined(&SO_TIMESTAMP);
	eval 'sub SO_TIMESTAMPNS () {35;}' unless defined(&SO_TIMESTAMPNS);
	eval 'sub SO_TIMESTAMPING () {37;}' unless defined(&SO_TIMESTAMPING);
    } else {
	eval 'sub SO_RCVTIMEO_OLD () {20;}' unless defined(&SO_RCVTIMEO_OLD);
	eval 'sub SO_SNDTIMEO_OLD () {21;}' unless defined(&SO_SNDTIMEO_OLD);
	eval 'sub SO_RCVTIMEO_NEW () {66;}' unless defined(&SO_RCVTIMEO_NEW);
	eval 'sub SO_SNDTIMEO_NEW () {67;}' unless defined(&SO_SNDTIMEO_NEW);
	eval 'sub SO_TIMESTAMP_OLD () {29;}' unless defined(&SO_TIMESTAMP_OLD);
	eval 'sub SO_TIMESTAMPNS_OLD () {35;}' unless defined(&SO_TIMESTAMPNS_OLD);
	eval 'sub SO_TIMESTAMPING_OLD () {37;}' unless defined(&SO_TIMESTAMPING_OLD);
	eval 'sub SO_TIMESTAMP_NEW () {63;}' unless defined(&SO_TIMESTAMP_NEW);
	eval 'sub SO_TIMESTAMPNS_NEW () {64;}' unless defined(&SO_TIMESTAMPNS_NEW);
	eval 'sub SO_TIMESTAMPING_NEW () {65;}' unless defined(&SO_TIMESTAMPING_NEW);
	if(defined(&__USE_TIME_BITS64)) {
	    eval 'sub SO_RCVTIMEO () { &SO_RCVTIMEO_NEW;}' unless defined(&SO_RCVTIMEO);
	    eval 'sub SO_SNDTIMEO () { &SO_SNDTIMEO_NEW;}' unless defined(&SO_SNDTIMEO);
	    eval 'sub SO_TIMESTAMP () { &SO_TIMESTAMP_NEW;}' unless defined(&SO_TIMESTAMP);
	    eval 'sub SO_TIMESTAMPNS () { &SO_TIMESTAMPNS_NEW;}' unless defined(&SO_TIMESTAMPNS);
	    eval 'sub SO_TIMESTAMPING () { &SO_TIMESTAMPING_NEW;}' unless defined(&SO_TIMESTAMPING);
	} else {
	    eval 'sub SO_RCVTIMEO () { &SO_RCVTIMEO_OLD;}' unless defined(&SO_RCVTIMEO);
	    eval 'sub SO_SNDTIMEO () { &SO_SNDTIMEO_OLD;}' unless defined(&SO_SNDTIMEO);
	    eval 'sub SO_TIMESTAMP () { &SO_TIMESTAMP_OLD;}' unless defined(&SO_TIMESTAMP);
	    eval 'sub SO_TIMESTAMPNS () { &SO_TIMESTAMPNS_OLD;}' unless defined(&SO_TIMESTAMPNS);
	    eval 'sub SO_TIMESTAMPING () { &SO_TIMESTAMPING_OLD;}' unless defined(&SO_TIMESTAMPING);
	}
    }
}
1;
