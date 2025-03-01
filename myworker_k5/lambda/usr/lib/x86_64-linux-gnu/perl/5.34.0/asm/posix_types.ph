require '_h2ph_pre.ph';

no warnings qw(redefine misc);

if(defined(&__i386__)) {
    require 'asm/posix_types_32.ph';
}
 elsif(defined(&__ILP32__)) {
    require 'asm/posix_types_x32.ph';
} else {
    require 'asm/posix_types_64.ph';
}
1;
