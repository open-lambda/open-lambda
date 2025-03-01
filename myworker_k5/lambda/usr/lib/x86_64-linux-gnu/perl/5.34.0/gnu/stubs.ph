require '_h2ph_pre.ph';

no warnings qw(redefine misc);

if(!defined (&__x86_64__)) {
    require 'gnu/stubs-32.ph';
}
if(defined (&__x86_64__)  && defined (&__LP64__)) {
    require 'gnu/stubs-64.ph';
}
if(defined (&__x86_64__)  && defined (&__ILP32__)) {
    require 'gnu/stubs-x32.ph';
}
1;
