require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_SYS_IOCTL_H)) {
    die("Never use <bits/ioctl-types.h> directly; include <sys/ioctl.h> instead.");
}
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
eval 'sub N_TTY () {0;}' unless defined(&N_TTY);
eval 'sub N_SLIP () {1;}' unless defined(&N_SLIP);
eval 'sub N_MOUSE () {2;}' unless defined(&N_MOUSE);
eval 'sub N_PPP () {3;}' unless defined(&N_PPP);
eval 'sub N_STRIP () {4;}' unless defined(&N_STRIP);
eval 'sub N_AX25 () {5;}' unless defined(&N_AX25);
eval 'sub N_X25 () {6;}' unless defined(&N_X25);
eval 'sub N_6PACK () {7;}' unless defined(&N_6PACK);
eval 'sub N_MASC () {8;}' unless defined(&N_MASC);
eval 'sub N_R3964 () {9;}' unless defined(&N_R3964);
eval 'sub N_PROFIBUS_FDL () {10;}' unless defined(&N_PROFIBUS_FDL);
eval 'sub N_IRDA () {11;}' unless defined(&N_IRDA);
eval 'sub N_SMSBLOCK () {12;}' unless defined(&N_SMSBLOCK);
eval 'sub N_HDLC () {13;}' unless defined(&N_HDLC);
eval 'sub N_SYNC_PPP () {14;}' unless defined(&N_SYNC_PPP);
eval 'sub N_HCI () {15;}' unless defined(&N_HCI);
1;
