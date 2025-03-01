require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&__osockaddr_defined)) {
    eval 'sub __osockaddr_defined () {1;}' unless defined(&__osockaddr_defined);
}
1;
