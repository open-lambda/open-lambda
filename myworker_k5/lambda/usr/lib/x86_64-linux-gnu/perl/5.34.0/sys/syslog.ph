require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_SYS_SYSLOG_H)) {
    eval 'sub _SYS_SYSLOG_H () {1;}' unless defined(&_SYS_SYSLOG_H);
    require 'features.ph';
    eval 'sub __need___va_list () {1;}' unless defined(&__need___va_list);
    require 'stdarg.ph';
    require 'bits/syslog-path.ph';
    eval 'sub LOG_EMERG () {0;}' unless defined(&LOG_EMERG);
    eval 'sub LOG_ALERT () {1;}' unless defined(&LOG_ALERT);
    eval 'sub LOG_CRIT () {2;}' unless defined(&LOG_CRIT);
    eval 'sub LOG_ERR () {3;}' unless defined(&LOG_ERR);
    eval 'sub LOG_WARNING () {4;}' unless defined(&LOG_WARNING);
    eval 'sub LOG_NOTICE () {5;}' unless defined(&LOG_NOTICE);
    eval 'sub LOG_INFO () {6;}' unless defined(&LOG_INFO);
    eval 'sub LOG_DEBUG () {7;}' unless defined(&LOG_DEBUG);
    eval 'sub LOG_PRIMASK () {0x7;}' unless defined(&LOG_PRIMASK);
    eval 'sub LOG_PRI {
        my($p) = @_;
	    eval q((($p) &  &LOG_PRIMASK));
    }' unless defined(&LOG_PRI);
    eval 'sub LOG_MAKEPRI {
        my($fac, $pri) = @_;
	    eval q((($fac) | ($pri)));
    }' unless defined(&LOG_MAKEPRI);
    if(defined(&SYSLOG_NAMES)) {
	eval 'sub INTERNAL_NOPRI () {0x10;}' unless defined(&INTERNAL_NOPRI);
	eval 'sub INTERNAL_MARK () { &LOG_MAKEPRI( &LOG_NFACILITIES << 3, 0);}' unless defined(&INTERNAL_MARK);
    }
    eval 'sub LOG_KERN () {(0<<3);}' unless defined(&LOG_KERN);
    eval 'sub LOG_USER () {(1<<3);}' unless defined(&LOG_USER);
    eval 'sub LOG_MAIL () {(2<<3);}' unless defined(&LOG_MAIL);
    eval 'sub LOG_DAEMON () {(3<<3);}' unless defined(&LOG_DAEMON);
    eval 'sub LOG_AUTH () {(4<<3);}' unless defined(&LOG_AUTH);
    eval 'sub LOG_SYSLOG () {(5<<3);}' unless defined(&LOG_SYSLOG);
    eval 'sub LOG_LPR () {(6<<3);}' unless defined(&LOG_LPR);
    eval 'sub LOG_NEWS () {(7<<3);}' unless defined(&LOG_NEWS);
    eval 'sub LOG_UUCP () {(8<<3);}' unless defined(&LOG_UUCP);
    eval 'sub LOG_CRON () {(9<<3);}' unless defined(&LOG_CRON);
    eval 'sub LOG_AUTHPRIV () {(10<<3);}' unless defined(&LOG_AUTHPRIV);
    eval 'sub LOG_FTP () {(11<<3);}' unless defined(&LOG_FTP);
    eval 'sub LOG_LOCAL0 () {(16<<3);}' unless defined(&LOG_LOCAL0);
    eval 'sub LOG_LOCAL1 () {(17<<3);}' unless defined(&LOG_LOCAL1);
    eval 'sub LOG_LOCAL2 () {(18<<3);}' unless defined(&LOG_LOCAL2);
    eval 'sub LOG_LOCAL3 () {(19<<3);}' unless defined(&LOG_LOCAL3);
    eval 'sub LOG_LOCAL4 () {(20<<3);}' unless defined(&LOG_LOCAL4);
    eval 'sub LOG_LOCAL5 () {(21<<3);}' unless defined(&LOG_LOCAL5);
    eval 'sub LOG_LOCAL6 () {(22<<3);}' unless defined(&LOG_LOCAL6);
    eval 'sub LOG_LOCAL7 () {(23<<3);}' unless defined(&LOG_LOCAL7);
    eval 'sub LOG_NFACILITIES () {24;}' unless defined(&LOG_NFACILITIES);
    eval 'sub LOG_FACMASK () {0x3f8;}' unless defined(&LOG_FACMASK);
    eval 'sub LOG_FAC {
        my($p) = @_;
	    eval q(((($p) &  &LOG_FACMASK) >> 3));
    }' unless defined(&LOG_FAC);
    if(defined(&SYSLOG_NAMES)) {
    }
    eval 'sub LOG_MASK {
        my($pri) = @_;
	    eval q((1<< ($pri)));
    }' unless defined(&LOG_MASK);
    eval 'sub LOG_UPTO {
        my($pri) = @_;
	    eval q(((1<< (($pri)+1)) - 1));
    }' unless defined(&LOG_UPTO);
    eval 'sub LOG_PID () {0x1;}' unless defined(&LOG_PID);
    eval 'sub LOG_CONS () {0x2;}' unless defined(&LOG_CONS);
    eval 'sub LOG_ODELAY () {0x4;}' unless defined(&LOG_ODELAY);
    eval 'sub LOG_NDELAY () {0x8;}' unless defined(&LOG_NDELAY);
    eval 'sub LOG_NOWAIT () {0x10;}' unless defined(&LOG_NOWAIT);
    eval 'sub LOG_PERROR () {0x20;}' unless defined(&LOG_PERROR);
    if(defined(&__USE_MISC)) {
    }
    if((defined(&__USE_FORTIFY_LEVEL) ? &__USE_FORTIFY_LEVEL : undef) > 0 && defined (&__fortify_function)) {
	require 'bits/syslog.ph';
    }
    require 'bits/floatn.ph';
    if(defined (&__LDBL_COMPAT) || (defined(&__LDOUBLE_REDIRECTS_TO_FLOAT128_ABI) ? &__LDOUBLE_REDIRECTS_TO_FLOAT128_ABI : undef) == 1) {
	require 'bits/syslog-ldbl.ph';
    }
}
1;
