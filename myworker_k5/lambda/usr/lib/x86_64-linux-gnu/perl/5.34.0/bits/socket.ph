require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&__BITS_SOCKET_H)) {
    eval 'sub __BITS_SOCKET_H () {1;}' unless defined(&__BITS_SOCKET_H);
    unless(defined(&_SYS_SOCKET_H)) {
	die("Never include <bits/socket.h> directly; use <sys/socket.h> instead.");
    }
    eval 'sub __need_size_t () {1;}' unless defined(&__need_size_t);
    require 'stddef.ph';
    require 'sys/types.ph';
    unless(defined(&__socklen_t_defined)) {
	eval 'sub __socklen_t_defined () {1;}' unless defined(&__socklen_t_defined);
    }
    require 'bits/socket_type.ph';
    eval 'sub PF_UNSPEC () {0;}' unless defined(&PF_UNSPEC);
    eval 'sub PF_LOCAL () {1;}' unless defined(&PF_LOCAL);
    eval 'sub PF_UNIX () { &PF_LOCAL;}' unless defined(&PF_UNIX);
    eval 'sub PF_FILE () { &PF_LOCAL;}' unless defined(&PF_FILE);
    eval 'sub PF_INET () {2;}' unless defined(&PF_INET);
    eval 'sub PF_AX25 () {3;}' unless defined(&PF_AX25);
    eval 'sub PF_IPX () {4;}' unless defined(&PF_IPX);
    eval 'sub PF_APPLETALK () {5;}' unless defined(&PF_APPLETALK);
    eval 'sub PF_NETROM () {6;}' unless defined(&PF_NETROM);
    eval 'sub PF_BRIDGE () {7;}' unless defined(&PF_BRIDGE);
    eval 'sub PF_ATMPVC () {8;}' unless defined(&PF_ATMPVC);
    eval 'sub PF_X25 () {9;}' unless defined(&PF_X25);
    eval 'sub PF_INET6 () {10;}' unless defined(&PF_INET6);
    eval 'sub PF_ROSE () {11;}' unless defined(&PF_ROSE);
    eval 'sub PF_DECnet () {12;}' unless defined(&PF_DECnet);
    eval 'sub PF_NETBEUI () {13;}' unless defined(&PF_NETBEUI);
    eval 'sub PF_SECURITY () {14;}' unless defined(&PF_SECURITY);
    eval 'sub PF_KEY () {15;}' unless defined(&PF_KEY);
    eval 'sub PF_NETLINK () {16;}' unless defined(&PF_NETLINK);
    eval 'sub PF_ROUTE () { &PF_NETLINK;}' unless defined(&PF_ROUTE);
    eval 'sub PF_PACKET () {17;}' unless defined(&PF_PACKET);
    eval 'sub PF_ASH () {18;}' unless defined(&PF_ASH);
    eval 'sub PF_ECONET () {19;}' unless defined(&PF_ECONET);
    eval 'sub PF_ATMSVC () {20;}' unless defined(&PF_ATMSVC);
    eval 'sub PF_RDS () {21;}' unless defined(&PF_RDS);
    eval 'sub PF_SNA () {22;}' unless defined(&PF_SNA);
    eval 'sub PF_IRDA () {23;}' unless defined(&PF_IRDA);
    eval 'sub PF_PPPOX () {24;}' unless defined(&PF_PPPOX);
    eval 'sub PF_WANPIPE () {25;}' unless defined(&PF_WANPIPE);
    eval 'sub PF_LLC () {26;}' unless defined(&PF_LLC);
    eval 'sub PF_IB () {27;}' unless defined(&PF_IB);
    eval 'sub PF_MPLS () {28;}' unless defined(&PF_MPLS);
    eval 'sub PF_CAN () {29;}' unless defined(&PF_CAN);
    eval 'sub PF_TIPC () {30;}' unless defined(&PF_TIPC);
    eval 'sub PF_BLUETOOTH () {31;}' unless defined(&PF_BLUETOOTH);
    eval 'sub PF_IUCV () {32;}' unless defined(&PF_IUCV);
    eval 'sub PF_RXRPC () {33;}' unless defined(&PF_RXRPC);
    eval 'sub PF_ISDN () {34;}' unless defined(&PF_ISDN);
    eval 'sub PF_PHONET () {35;}' unless defined(&PF_PHONET);
    eval 'sub PF_IEEE802154 () {36;}' unless defined(&PF_IEEE802154);
    eval 'sub PF_CAIF () {37;}' unless defined(&PF_CAIF);
    eval 'sub PF_ALG () {38;}' unless defined(&PF_ALG);
    eval 'sub PF_NFC () {39;}' unless defined(&PF_NFC);
    eval 'sub PF_VSOCK () {40;}' unless defined(&PF_VSOCK);
    eval 'sub PF_KCM () {41;}' unless defined(&PF_KCM);
    eval 'sub PF_QIPCRTR () {42;}' unless defined(&PF_QIPCRTR);
    eval 'sub PF_SMC () {43;}' unless defined(&PF_SMC);
    eval 'sub PF_XDP () {44;}' unless defined(&PF_XDP);
    eval 'sub PF_MCTP () {45;}' unless defined(&PF_MCTP);
    eval 'sub PF_MAX () {46;}' unless defined(&PF_MAX);
    eval 'sub AF_UNSPEC () { &PF_UNSPEC;}' unless defined(&AF_UNSPEC);
    eval 'sub AF_LOCAL () { &PF_LOCAL;}' unless defined(&AF_LOCAL);
    eval 'sub AF_UNIX () { &PF_UNIX;}' unless defined(&AF_UNIX);
    eval 'sub AF_FILE () { &PF_FILE;}' unless defined(&AF_FILE);
    eval 'sub AF_INET () { &PF_INET;}' unless defined(&AF_INET);
    eval 'sub AF_AX25 () { &PF_AX25;}' unless defined(&AF_AX25);
    eval 'sub AF_IPX () { &PF_IPX;}' unless defined(&AF_IPX);
    eval 'sub AF_APPLETALK () { &PF_APPLETALK;}' unless defined(&AF_APPLETALK);
    eval 'sub AF_NETROM () { &PF_NETROM;}' unless defined(&AF_NETROM);
    eval 'sub AF_BRIDGE () { &PF_BRIDGE;}' unless defined(&AF_BRIDGE);
    eval 'sub AF_ATMPVC () { &PF_ATMPVC;}' unless defined(&AF_ATMPVC);
    eval 'sub AF_X25 () { &PF_X25;}' unless defined(&AF_X25);
    eval 'sub AF_INET6 () { &PF_INET6;}' unless defined(&AF_INET6);
    eval 'sub AF_ROSE () { &PF_ROSE;}' unless defined(&AF_ROSE);
    eval 'sub AF_DECnet () { &PF_DECnet;}' unless defined(&AF_DECnet);
    eval 'sub AF_NETBEUI () { &PF_NETBEUI;}' unless defined(&AF_NETBEUI);
    eval 'sub AF_SECURITY () { &PF_SECURITY;}' unless defined(&AF_SECURITY);
    eval 'sub AF_KEY () { &PF_KEY;}' unless defined(&AF_KEY);
    eval 'sub AF_NETLINK () { &PF_NETLINK;}' unless defined(&AF_NETLINK);
    eval 'sub AF_ROUTE () { &PF_ROUTE;}' unless defined(&AF_ROUTE);
    eval 'sub AF_PACKET () { &PF_PACKET;}' unless defined(&AF_PACKET);
    eval 'sub AF_ASH () { &PF_ASH;}' unless defined(&AF_ASH);
    eval 'sub AF_ECONET () { &PF_ECONET;}' unless defined(&AF_ECONET);
    eval 'sub AF_ATMSVC () { &PF_ATMSVC;}' unless defined(&AF_ATMSVC);
    eval 'sub AF_RDS () { &PF_RDS;}' unless defined(&AF_RDS);
    eval 'sub AF_SNA () { &PF_SNA;}' unless defined(&AF_SNA);
    eval 'sub AF_IRDA () { &PF_IRDA;}' unless defined(&AF_IRDA);
    eval 'sub AF_PPPOX () { &PF_PPPOX;}' unless defined(&AF_PPPOX);
    eval 'sub AF_WANPIPE () { &PF_WANPIPE;}' unless defined(&AF_WANPIPE);
    eval 'sub AF_LLC () { &PF_LLC;}' unless defined(&AF_LLC);
    eval 'sub AF_IB () { &PF_IB;}' unless defined(&AF_IB);
    eval 'sub AF_MPLS () { &PF_MPLS;}' unless defined(&AF_MPLS);
    eval 'sub AF_CAN () { &PF_CAN;}' unless defined(&AF_CAN);
    eval 'sub AF_TIPC () { &PF_TIPC;}' unless defined(&AF_TIPC);
    eval 'sub AF_BLUETOOTH () { &PF_BLUETOOTH;}' unless defined(&AF_BLUETOOTH);
    eval 'sub AF_IUCV () { &PF_IUCV;}' unless defined(&AF_IUCV);
    eval 'sub AF_RXRPC () { &PF_RXRPC;}' unless defined(&AF_RXRPC);
    eval 'sub AF_ISDN () { &PF_ISDN;}' unless defined(&AF_ISDN);
    eval 'sub AF_PHONET () { &PF_PHONET;}' unless defined(&AF_PHONET);
    eval 'sub AF_IEEE802154 () { &PF_IEEE802154;}' unless defined(&AF_IEEE802154);
    eval 'sub AF_CAIF () { &PF_CAIF;}' unless defined(&AF_CAIF);
    eval 'sub AF_ALG () { &PF_ALG;}' unless defined(&AF_ALG);
    eval 'sub AF_NFC () { &PF_NFC;}' unless defined(&AF_NFC);
    eval 'sub AF_VSOCK () { &PF_VSOCK;}' unless defined(&AF_VSOCK);
    eval 'sub AF_KCM () { &PF_KCM;}' unless defined(&AF_KCM);
    eval 'sub AF_QIPCRTR () { &PF_QIPCRTR;}' unless defined(&AF_QIPCRTR);
    eval 'sub AF_SMC () { &PF_SMC;}' unless defined(&AF_SMC);
    eval 'sub AF_XDP () { &PF_XDP;}' unless defined(&AF_XDP);
    eval 'sub AF_MCTP () { &PF_MCTP;}' unless defined(&AF_MCTP);
    eval 'sub AF_MAX () { &PF_MAX;}' unless defined(&AF_MAX);
    eval 'sub SOL_RAW () {255;}' unless defined(&SOL_RAW);
    eval 'sub SOL_DECNET () {261;}' unless defined(&SOL_DECNET);
    eval 'sub SOL_X25 () {262;}' unless defined(&SOL_X25);
    eval 'sub SOL_PACKET () {263;}' unless defined(&SOL_PACKET);
    eval 'sub SOL_ATM () {264;}' unless defined(&SOL_ATM);
    eval 'sub SOL_AAL () {265;}' unless defined(&SOL_AAL);
    eval 'sub SOL_IRDA () {266;}' unless defined(&SOL_IRDA);
    eval 'sub SOL_NETBEUI () {267;}' unless defined(&SOL_NETBEUI);
    eval 'sub SOL_LLC () {268;}' unless defined(&SOL_LLC);
    eval 'sub SOL_DCCP () {269;}' unless defined(&SOL_DCCP);
    eval 'sub SOL_NETLINK () {270;}' unless defined(&SOL_NETLINK);
    eval 'sub SOL_TIPC () {271;}' unless defined(&SOL_TIPC);
    eval 'sub SOL_RXRPC () {272;}' unless defined(&SOL_RXRPC);
    eval 'sub SOL_PPPOL2TP () {273;}' unless defined(&SOL_PPPOL2TP);
    eval 'sub SOL_BLUETOOTH () {274;}' unless defined(&SOL_BLUETOOTH);
    eval 'sub SOL_PNPIPE () {275;}' unless defined(&SOL_PNPIPE);
    eval 'sub SOL_RDS () {276;}' unless defined(&SOL_RDS);
    eval 'sub SOL_IUCV () {277;}' unless defined(&SOL_IUCV);
    eval 'sub SOL_CAIF () {278;}' unless defined(&SOL_CAIF);
    eval 'sub SOL_ALG () {279;}' unless defined(&SOL_ALG);
    eval 'sub SOL_NFC () {280;}' unless defined(&SOL_NFC);
    eval 'sub SOL_KCM () {281;}' unless defined(&SOL_KCM);
    eval 'sub SOL_TLS () {282;}' unless defined(&SOL_TLS);
    eval 'sub SOL_XDP () {283;}' unless defined(&SOL_XDP);
    eval 'sub SOMAXCONN () {4096;}' unless defined(&SOMAXCONN);
    require 'bits/sockaddr.ph';
    eval 'sub __ss_aligntype () {\'unsigned long int\';}' unless defined(&__ss_aligntype);
    eval 'sub _SS_PADSIZE () {( &_SS_SIZE -  &__SOCKADDR_COMMON_SIZE - $sizeof{ &__ss_aligntype});}' unless defined(&_SS_PADSIZE);
    eval("sub MSG_OOB () { 0x01; }") unless defined(&MSG_OOB);
    eval("sub MSG_PEEK () { 0x02; }") unless defined(&MSG_PEEK);
    eval("sub MSG_DONTROUTE () { 0x04; }") unless defined(&MSG_DONTROUTE);
    eval("sub MSG_CTRUNC () { 0x08; }") unless defined(&MSG_CTRUNC);
    eval("sub MSG_PROXY () { 0x10; }") unless defined(&MSG_PROXY);
    eval("sub MSG_TRUNC () { 0x20; }") unless defined(&MSG_TRUNC);
    eval("sub MSG_DONTWAIT () { 0x40; }") unless defined(&MSG_DONTWAIT);
    eval("sub MSG_EOR () { 0x80; }") unless defined(&MSG_EOR);
    eval("sub MSG_WAITALL () { 0x100; }") unless defined(&MSG_WAITALL);
    eval("sub MSG_FIN () { 0x200; }") unless defined(&MSG_FIN);
    eval("sub MSG_SYN () { 0x400; }") unless defined(&MSG_SYN);
    eval("sub MSG_CONFIRM () { 0x800; }") unless defined(&MSG_CONFIRM);
    eval("sub MSG_RST () { 0x1000; }") unless defined(&MSG_RST);
    eval("sub MSG_ERRQUEUE () { 0x2000; }") unless defined(&MSG_ERRQUEUE);
    eval("sub MSG_NOSIGNAL () { 0x4000; }") unless defined(&MSG_NOSIGNAL);
    eval("sub MSG_MORE () { 0x8000; }") unless defined(&MSG_MORE);
    eval("sub MSG_WAITFORONE () { 0x10000; }") unless defined(&MSG_WAITFORONE);
    eval("sub MSG_BATCH () { 0x40000; }") unless defined(&MSG_BATCH);
    eval("sub MSG_ZEROCOPY () { 0x4000000; }") unless defined(&MSG_ZEROCOPY);
    eval("sub MSG_FASTOPEN () { 0x20000000; }") unless defined(&MSG_FASTOPEN);
    eval("sub MSG_CMSG_CLOEXEC () { 0x40000000; }") unless defined(&MSG_CMSG_CLOEXEC);
    if((defined(&__glibc_c99_flexarr_available) ? &__glibc_c99_flexarr_available : undef)) {
    }
    if((defined(&__glibc_c99_flexarr_available) ? &__glibc_c99_flexarr_available : undef)) {
	eval 'sub CMSG_DATA {
	    my($cmsg) = @_;
    	    eval q((($cmsg)-> &__cmsg_data));
	}' unless defined(&CMSG_DATA);
    } else {
	eval 'sub CMSG_DATA {
	    my($cmsg) = @_;
    	    eval q(( ( ($cmsg) + 1)));
	}' unless defined(&CMSG_DATA);
    }
    eval 'sub CMSG_NXTHDR {
        my($mhdr, $cmsg) = @_;
	    eval q( &__cmsg_nxthdr ($mhdr, $cmsg));
    }' unless defined(&CMSG_NXTHDR);
    eval 'sub CMSG_FIRSTHDR {
        my($mhdr) = @_;
	    eval q(( ($mhdr)-> &msg_controllen >= $sizeof{\'struct cmsghdr\'} ?  ($mhdr)-> &msg_control :  0));
    }' unless defined(&CMSG_FIRSTHDR);
    eval 'sub CMSG_ALIGN {
        my($len) = @_;
	    eval q(((($len) + $sizeof{\'size_t\'} - 1) &  ~($sizeof{\'size_t\'} - 1)));
    }' unless defined(&CMSG_ALIGN);
    eval 'sub CMSG_SPACE {
        my($len) = @_;
	    eval q(( &CMSG_ALIGN ($len) +  &CMSG_ALIGN ($sizeof{\'struct cmsghdr\'})));
    }' unless defined(&CMSG_SPACE);
    eval 'sub CMSG_LEN {
        my($len) = @_;
	    eval q(( &CMSG_ALIGN ($sizeof{\'struct cmsghdr\'}) + ($len)));
    }' unless defined(&CMSG_LEN);
    if(defined(&__USE_EXTERN_INLINES)) {
	unless(defined(&_EXTERN_INLINE)) {
	    eval 'sub _EXTERN_INLINE () { &__extern_inline;}' unless defined(&_EXTERN_INLINE);
	}
    }
    eval("sub SCM_RIGHTS () { 0x01; }") unless defined(&SCM_RIGHTS);
    if(defined(&__USE_GNU)) {
    }
    if(defined(&__USE_MISC)) {
	require 'bits/types/time_t.ph';
	require 'asm/socket.ph';
    } else {
	eval 'sub SO_DEBUG () {1;}' unless defined(&SO_DEBUG);
	require 'bits/socket-constants.ph';
    }
}
1;
