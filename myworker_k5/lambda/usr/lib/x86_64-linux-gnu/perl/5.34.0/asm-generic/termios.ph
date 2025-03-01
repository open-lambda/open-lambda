require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_ASM_GENERIC_TERMIOS_H)) {
    eval 'sub _ASM_GENERIC_TERMIOS_H () {1;}' unless defined(&_ASM_GENERIC_TERMIOS_H);
    require 'asm/termbits.ph';
    require 'asm/ioctls.ph';
    eval 'sub NCC () {8;}' unless defined(&NCC);
    eval 'sub TIOCM_LE () {0x1;}' unless defined(&TIOCM_LE);
    eval 'sub TIOCM_DTR () {0x2;}' unless defined(&TIOCM_DTR);
    eval 'sub TIOCM_RTS () {0x4;}' unless defined(&TIOCM_RTS);
    eval 'sub TIOCM_ST () {0x8;}' unless defined(&TIOCM_ST);
    eval 'sub TIOCM_SR () {0x10;}' unless defined(&TIOCM_SR);
    eval 'sub TIOCM_CTS () {0x20;}' unless defined(&TIOCM_CTS);
    eval 'sub TIOCM_CAR () {0x40;}' unless defined(&TIOCM_CAR);
    eval 'sub TIOCM_RNG () {0x80;}' unless defined(&TIOCM_RNG);
    eval 'sub TIOCM_DSR () {0x100;}' unless defined(&TIOCM_DSR);
    eval 'sub TIOCM_CD () { &TIOCM_CAR;}' unless defined(&TIOCM_CD);
    eval 'sub TIOCM_RI () { &TIOCM_RNG;}' unless defined(&TIOCM_RI);
    eval 'sub TIOCM_OUT1 () {0x2000;}' unless defined(&TIOCM_OUT1);
    eval 'sub TIOCM_OUT2 () {0x4000;}' unless defined(&TIOCM_OUT2);
    eval 'sub TIOCM_LOOP () {0x8000;}' unless defined(&TIOCM_LOOP);
}
1;
