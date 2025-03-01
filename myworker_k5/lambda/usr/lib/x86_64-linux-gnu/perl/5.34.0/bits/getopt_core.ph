require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_GETOPT_CORE_H)) {
    eval 'sub _GETOPT_CORE_H () {1;}' unless defined(&_GETOPT_CORE_H);
}
1;
