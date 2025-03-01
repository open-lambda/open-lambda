require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_SYS_UCONTEXT_H)) {
    eval 'sub _SYS_UCONTEXT_H () {1;}' unless defined(&_SYS_UCONTEXT_H);
    require 'features.ph';
    require 'bits/types.ph';
    require 'bits/types/sigset_t.ph';
    require 'bits/types/stack_t.ph';
    if(defined(&__USE_MISC)) {
	eval 'sub __ctx {
	    my($fld) = @_;
    	    eval q($fld);
	}' unless defined(&__ctx);
    } else {
	eval 'sub __ctx {
	    my($fld) = @_;
    	    eval q( &__  $fld);
	}' unless defined(&__ctx);
    }
    if(defined(&__x86_64__)) {
	eval 'sub __NGREG () {23;}' unless defined(&__NGREG);
	if(defined(&__USE_MISC)) {
	    eval 'sub NGREG () { &__NGREG;}' unless defined(&NGREG);
	}
	if(defined(&__USE_GNU)) {
	    eval("sub REG_R8 () { 0; }") unless defined(&REG_R8);
	    eval("sub REG_R9 () { 1; }") unless defined(&REG_R9);
	    eval("sub REG_R10 () { 2; }") unless defined(&REG_R10);
	    eval("sub REG_R11 () { 3; }") unless defined(&REG_R11);
	    eval("sub REG_R12 () { 4; }") unless defined(&REG_R12);
	    eval("sub REG_R13 () { 5; }") unless defined(&REG_R13);
	    eval("sub REG_R14 () { 6; }") unless defined(&REG_R14);
	    eval("sub REG_R15 () { 7; }") unless defined(&REG_R15);
	    eval("sub REG_RDI () { 8; }") unless defined(&REG_RDI);
	    eval("sub REG_RSI () { 9; }") unless defined(&REG_RSI);
	    eval("sub REG_RBP () { 10; }") unless defined(&REG_RBP);
	    eval("sub REG_RBX () { 11; }") unless defined(&REG_RBX);
	    eval("sub REG_RDX () { 12; }") unless defined(&REG_RDX);
	    eval("sub REG_RAX () { 13; }") unless defined(&REG_RAX);
	    eval("sub REG_RCX () { 14; }") unless defined(&REG_RCX);
	    eval("sub REG_RSP () { 15; }") unless defined(&REG_RSP);
	    eval("sub REG_RIP () { 16; }") unless defined(&REG_RIP);
	    eval("sub REG_EFL () { 17; }") unless defined(&REG_EFL);
	    eval("sub REG_CSGSFS () { 18; }") unless defined(&REG_CSGSFS);
	    eval("sub REG_ERR () { 19; }") unless defined(&REG_ERR);
	    eval("sub REG_TRAPNO () { 20; }") unless defined(&REG_TRAPNO);
	    eval("sub REG_OLDMASK () { 21; }") unless defined(&REG_OLDMASK);
	    eval("sub REG_CR2 () { 22; }") unless defined(&REG_CR2);
	}
    } else {
	eval 'sub __NGREG () {19;}' unless defined(&__NGREG);
	if(defined(&__USE_MISC)) {
	    eval 'sub NGREG () { &__NGREG;}' unless defined(&NGREG);
	}
	if(defined(&__USE_GNU)) {
	    eval("sub REG_GS () { 0; }") unless defined(&REG_GS);
	    eval("sub REG_FS () { 1; }") unless defined(&REG_FS);
	    eval("sub REG_ES () { 2; }") unless defined(&REG_ES);
	    eval("sub REG_DS () { 3; }") unless defined(&REG_DS);
	    eval("sub REG_EDI () { 4; }") unless defined(&REG_EDI);
	    eval("sub REG_ESI () { 5; }") unless defined(&REG_ESI);
	    eval("sub REG_EBP () { 6; }") unless defined(&REG_EBP);
	    eval("sub REG_ESP () { 7; }") unless defined(&REG_ESP);
	    eval("sub REG_EBX () { 8; }") unless defined(&REG_EBX);
	    eval("sub REG_EDX () { 9; }") unless defined(&REG_EDX);
	    eval("sub REG_ECX () { 10; }") unless defined(&REG_ECX);
	    eval("sub REG_EAX () { 11; }") unless defined(&REG_EAX);
	    eval("sub REG_TRAPNO () { 12; }") unless defined(&REG_TRAPNO);
	    eval("sub REG_ERR () { 13; }") unless defined(&REG_ERR);
	    eval("sub REG_EIP () { 14; }") unless defined(&REG_EIP);
	    eval("sub REG_CS () { 15; }") unless defined(&REG_CS);
	    eval("sub REG_EFL () { 16; }") unless defined(&REG_EFL);
	    eval("sub REG_UESP () { 17; }") unless defined(&REG_UESP);
	    eval("sub REG_SS () { 18; }") unless defined(&REG_SS);
	}
    }
    undef(&__ctx) if defined(&__ctx);
}
1;
