require '_h2ph_pre.ph';

no warnings qw(redefine misc);

require 'bits/wordsize.ph';
if(defined (&__x86_64__)  && defined (&__ILP32__)) {
    eval 'sub __TIMESIZE () {64;}' unless defined(&__TIMESIZE);
} else {
    eval 'sub __TIMESIZE () { &__WORDSIZE;}' unless defined(&__TIMESIZE);
}
1;
