require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_ASM_GENERIC_IOCTL_H)) {
    eval 'sub _ASM_GENERIC_IOCTL_H () {1;}' unless defined(&_ASM_GENERIC_IOCTL_H);
    eval 'sub _IOC_NRBITS () {8;}' unless defined(&_IOC_NRBITS);
    eval 'sub _IOC_TYPEBITS () {8;}' unless defined(&_IOC_TYPEBITS);
    unless(defined(&_IOC_SIZEBITS)) {
	eval 'sub _IOC_SIZEBITS () {14;}' unless defined(&_IOC_SIZEBITS);
    }
    unless(defined(&_IOC_DIRBITS)) {
	eval 'sub _IOC_DIRBITS () {2;}' unless defined(&_IOC_DIRBITS);
    }
    eval 'sub _IOC_NRMASK () {((1<<  &_IOC_NRBITS)-1);}' unless defined(&_IOC_NRMASK);
    eval 'sub _IOC_TYPEMASK () {((1<<  &_IOC_TYPEBITS)-1);}' unless defined(&_IOC_TYPEMASK);
    eval 'sub _IOC_SIZEMASK () {((1<<  &_IOC_SIZEBITS)-1);}' unless defined(&_IOC_SIZEMASK);
    eval 'sub _IOC_DIRMASK () {((1<<  &_IOC_DIRBITS)-1);}' unless defined(&_IOC_DIRMASK);
    eval 'sub _IOC_NRSHIFT () {0;}' unless defined(&_IOC_NRSHIFT);
    eval 'sub _IOC_TYPESHIFT () {( &_IOC_NRSHIFT+ &_IOC_NRBITS);}' unless defined(&_IOC_TYPESHIFT);
    eval 'sub _IOC_SIZESHIFT () {( &_IOC_TYPESHIFT+ &_IOC_TYPEBITS);}' unless defined(&_IOC_SIZESHIFT);
    eval 'sub _IOC_DIRSHIFT () {( &_IOC_SIZESHIFT+ &_IOC_SIZEBITS);}' unless defined(&_IOC_DIRSHIFT);
    unless(defined(&_IOC_NONE)) {
	eval 'sub _IOC_NONE () {0;}' unless defined(&_IOC_NONE);
    }
    unless(defined(&_IOC_WRITE)) {
	eval 'sub _IOC_WRITE () {1;}' unless defined(&_IOC_WRITE);
    }
    unless(defined(&_IOC_READ)) {
	eval 'sub _IOC_READ () {2;}' unless defined(&_IOC_READ);
    }
    eval 'sub _IOC {
        my($dir,$type,$nr,$size) = @_;
	    eval q(((($dir) <<  &_IOC_DIRSHIFT) | (($type) <<  &_IOC_TYPESHIFT) | (($nr) <<  &_IOC_NRSHIFT) | (($size) <<  &_IOC_SIZESHIFT)));
    }' unless defined(&_IOC);
    eval 'sub _IOC_TYPECHECK {
        my($t) = @_;
	    eval q(($sizeof{$t}));
    }' unless defined(&_IOC_TYPECHECK);
    eval 'sub _IO {
        my($type,$nr) = @_;
	    eval q( &_IOC( &_IOC_NONE,($type),($nr),0));
    }' unless defined(&_IO);
    eval 'sub _IOR {
        my($type,$nr,$size) = @_;
	    eval q( &_IOC( &_IOC_READ,($type),($nr),( &_IOC_TYPECHECK($size))));
    }' unless defined(&_IOR);
    eval 'sub _IOW {
        my($type,$nr,$size) = @_;
	    eval q( &_IOC( &_IOC_WRITE,($type),($nr),( &_IOC_TYPECHECK($size))));
    }' unless defined(&_IOW);
    eval 'sub _IOWR {
        my($type,$nr,$size) = @_;
	    eval q( &_IOC( &_IOC_READ| &_IOC_WRITE,($type),($nr),( &_IOC_TYPECHECK($size))));
    }' unless defined(&_IOWR);
    eval 'sub _IOR_BAD {
        my($type,$nr,$size) = @_;
	    eval q( &_IOC( &_IOC_READ,($type),($nr),$sizeof{$size}));
    }' unless defined(&_IOR_BAD);
    eval 'sub _IOW_BAD {
        my($type,$nr,$size) = @_;
	    eval q( &_IOC( &_IOC_WRITE,($type),($nr),$sizeof{$size}));
    }' unless defined(&_IOW_BAD);
    eval 'sub _IOWR_BAD {
        my($type,$nr,$size) = @_;
	    eval q( &_IOC( &_IOC_READ| &_IOC_WRITE,($type),($nr),$sizeof{$size}));
    }' unless defined(&_IOWR_BAD);
    eval 'sub _IOC_DIR {
        my($nr) = @_;
	    eval q(((($nr) >>  &_IOC_DIRSHIFT) &  &_IOC_DIRMASK));
    }' unless defined(&_IOC_DIR);
    eval 'sub _IOC_TYPE {
        my($nr) = @_;
	    eval q(((($nr) >>  &_IOC_TYPESHIFT) &  &_IOC_TYPEMASK));
    }' unless defined(&_IOC_TYPE);
    eval 'sub _IOC_NR {
        my($nr) = @_;
	    eval q(((($nr) >>  &_IOC_NRSHIFT) &  &_IOC_NRMASK));
    }' unless defined(&_IOC_NR);
    eval 'sub _IOC_SIZE {
        my($nr) = @_;
	    eval q(((($nr) >>  &_IOC_SIZESHIFT) &  &_IOC_SIZEMASK));
    }' unless defined(&_IOC_SIZE);
    eval 'sub IOC_IN () {( &_IOC_WRITE <<  &_IOC_DIRSHIFT);}' unless defined(&IOC_IN);
    eval 'sub IOC_OUT () {( &_IOC_READ <<  &_IOC_DIRSHIFT);}' unless defined(&IOC_OUT);
    eval 'sub IOC_INOUT () {(( &_IOC_WRITE| &_IOC_READ) <<  &_IOC_DIRSHIFT);}' unless defined(&IOC_INOUT);
    eval 'sub IOCSIZE_MASK () {( &_IOC_SIZEMASK <<  &_IOC_SIZESHIFT);}' unless defined(&IOCSIZE_MASK);
    eval 'sub IOCSIZE_SHIFT () {( &_IOC_SIZESHIFT);}' unless defined(&IOCSIZE_SHIFT);
}
1;
