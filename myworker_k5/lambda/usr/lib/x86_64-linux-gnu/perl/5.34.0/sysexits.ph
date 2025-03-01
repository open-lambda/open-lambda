require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_SYSEXITS_H)) {
    eval 'sub _SYSEXITS_H () {1;}' unless defined(&_SYSEXITS_H);
    eval 'sub EX_OK () {0;}' unless defined(&EX_OK);
    eval 'sub EX__BASE () {64;}' unless defined(&EX__BASE);
    eval 'sub EX_USAGE () {64;}' unless defined(&EX_USAGE);
    eval 'sub EX_DATAERR () {65;}' unless defined(&EX_DATAERR);
    eval 'sub EX_NOINPUT () {66;}' unless defined(&EX_NOINPUT);
    eval 'sub EX_NOUSER () {67;}' unless defined(&EX_NOUSER);
    eval 'sub EX_NOHOST () {68;}' unless defined(&EX_NOHOST);
    eval 'sub EX_UNAVAILABLE () {69;}' unless defined(&EX_UNAVAILABLE);
    eval 'sub EX_SOFTWARE () {70;}' unless defined(&EX_SOFTWARE);
    eval 'sub EX_OSERR () {71;}' unless defined(&EX_OSERR);
    eval 'sub EX_OSFILE () {72;}' unless defined(&EX_OSFILE);
    eval 'sub EX_CANTCREAT () {73;}' unless defined(&EX_CANTCREAT);
    eval 'sub EX_IOERR () {74;}' unless defined(&EX_IOERR);
    eval 'sub EX_TEMPFAIL () {75;}' unless defined(&EX_TEMPFAIL);
    eval 'sub EX_PROTOCOL () {76;}' unless defined(&EX_PROTOCOL);
    eval 'sub EX_NOPERM () {77;}' unless defined(&EX_NOPERM);
    eval 'sub EX_CONFIG () {78;}' unless defined(&EX_CONFIG);
    eval 'sub EX__MAX () {78;}' unless defined(&EX__MAX);
}
1;
