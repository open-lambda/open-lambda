require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&__iovec_defined)) {
    eval 'sub __iovec_defined () {1;}' unless defined(&__iovec_defined);
    eval 'sub __need_size_t () {1;}' unless defined(&__need_size_t);
    require 'stddef.ph';
}
1;
