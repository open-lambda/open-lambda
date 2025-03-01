require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&__ASM_GENERIC_SOCKIOS_H)) {
    eval 'sub __ASM_GENERIC_SOCKIOS_H () {1;}' unless defined(&__ASM_GENERIC_SOCKIOS_H);
    eval 'sub FIOSETOWN () {0x8901;}' unless defined(&FIOSETOWN);
    eval 'sub SIOCSPGRP () {0x8902;}' unless defined(&SIOCSPGRP);
    eval 'sub FIOGETOWN () {0x8903;}' unless defined(&FIOGETOWN);
    eval 'sub SIOCGPGRP () {0x8904;}' unless defined(&SIOCGPGRP);
    eval 'sub SIOCATMARK () {0x8905;}' unless defined(&SIOCATMARK);
    eval 'sub SIOCGSTAMP_OLD () {0x8906;}' unless defined(&SIOCGSTAMP_OLD);
    eval 'sub SIOCGSTAMPNS_OLD () {0x8907;}' unless defined(&SIOCGSTAMPNS_OLD);
}
1;
