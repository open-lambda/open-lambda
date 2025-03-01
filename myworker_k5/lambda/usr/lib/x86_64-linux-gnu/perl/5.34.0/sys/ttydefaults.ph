require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_SYS_TTYDEFAULTS_H_)) {
    eval 'sub _SYS_TTYDEFAULTS_H_ () {1;}' unless defined(&_SYS_TTYDEFAULTS_H_);
    eval 'sub TTYDEF_IFLAG () {( &BRKINT |  &ISTRIP |  &ICRNL |  &IMAXBEL |  &IXON |  &IXANY);}' unless defined(&TTYDEF_IFLAG);
    eval 'sub TTYDEF_OFLAG () {( &OPOST |  &ONLCR |  &XTABS);}' unless defined(&TTYDEF_OFLAG);
    eval 'sub TTYDEF_LFLAG () {( &ECHO |  &ICANON |  &ISIG |  &IEXTEN |  &ECHOE| &ECHOKE| &ECHOCTL);}' unless defined(&TTYDEF_LFLAG);
    eval 'sub TTYDEF_CFLAG () {( &CREAD |  &CS7 |  &PARENB |  &HUPCL);}' unless defined(&TTYDEF_CFLAG);
    eval 'sub TTYDEF_SPEED () {( &B9600);}' unless defined(&TTYDEF_SPEED);
    eval 'sub CTRL {
        my($x) = @_;
	    eval q(($x&037));
    }' unless defined(&CTRL);
    eval 'sub CEOF () { &CTRL(ord(\'d\'));}' unless defined(&CEOF);
    if(defined(&_POSIX_VDISABLE)) {
	eval 'sub CEOL () { &_POSIX_VDISABLE;}' unless defined(&CEOL);
    } else {
	eval 'sub CEOL () {ord(\'\\0\');}' unless defined(&CEOL);
    }
    eval 'sub CERASE () {0177;}' unless defined(&CERASE);
    eval 'sub CINTR () { &CTRL(ord(\'c\'));}' unless defined(&CINTR);
    if(defined(&_POSIX_VDISABLE)) {
	eval 'sub CSTATUS () { &_POSIX_VDISABLE;}' unless defined(&CSTATUS);
    } else {
	eval 'sub CSTATUS () {ord(\'\\0\');}' unless defined(&CSTATUS);
    }
    eval 'sub CKILL () { &CTRL(ord(\'u\'));}' unless defined(&CKILL);
    eval 'sub CMIN () {1;}' unless defined(&CMIN);
    eval 'sub CQUIT () {034;}' unless defined(&CQUIT);
    eval 'sub CSUSP () { &CTRL(ord(\'z\'));}' unless defined(&CSUSP);
    eval 'sub CTIME () {0;}' unless defined(&CTIME);
    eval 'sub CDSUSP () { &CTRL(ord(\'y\'));}' unless defined(&CDSUSP);
    eval 'sub CSTART () { &CTRL(ord(\'q\'));}' unless defined(&CSTART);
    eval 'sub CSTOP () { &CTRL(ord(\'s\'));}' unless defined(&CSTOP);
    eval 'sub CLNEXT () { &CTRL(ord(\'v\'));}' unless defined(&CLNEXT);
    eval 'sub CDISCARD () { &CTRL(ord(\'o\'));}' unless defined(&CDISCARD);
    eval 'sub CWERASE () { &CTRL(ord(\'w\'));}' unless defined(&CWERASE);
    eval 'sub CREPRINT () { &CTRL(ord(\'r\'));}' unless defined(&CREPRINT);
    eval 'sub CEOT () { &CEOF;}' unless defined(&CEOT);
    eval 'sub CBRK () { &CEOL;}' unless defined(&CBRK);
    eval 'sub CRPRNT () { &CREPRINT;}' unless defined(&CRPRNT);
    eval 'sub CFLUSH () { &CDISCARD;}' unless defined(&CFLUSH);
}
if(defined(&TTYDEFCHARS)) {
    undef(&TTYDEFCHARS) if defined(&TTYDEFCHARS);
}
1;
