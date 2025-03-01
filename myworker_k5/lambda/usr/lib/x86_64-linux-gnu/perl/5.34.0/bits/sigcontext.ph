require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_BITS_SIGCONTEXT_H)) {
    eval 'sub _BITS_SIGCONTEXT_H () {1;}' unless defined(&_BITS_SIGCONTEXT_H);
    if(!defined (&_SIGNAL_H)  && !defined (&_SYS_UCONTEXT_H)) {
	die("Never use <bits/sigcontext.h> directly; include <signal.h> instead.");
    }
    require 'bits/types.ph';
    eval 'sub FP_XSTATE_MAGIC1 () {0x46505853;}' unless defined(&FP_XSTATE_MAGIC1);
    eval 'sub FP_XSTATE_MAGIC2 () {0x46505845;}' unless defined(&FP_XSTATE_MAGIC2);
    eval 'sub FP_XSTATE_MAGIC2_SIZE () {$sizeof{ &FP_XSTATE_MAGIC2};}' unless defined(&FP_XSTATE_MAGIC2_SIZE);
    unless(defined(&__x86_64__)) {
	unless(defined(&sigcontext_struct)) {
	    eval 'sub sigcontext_struct () { &sigcontext;}' unless defined(&sigcontext_struct);
	}
	eval 'sub X86_FXSR_MAGIC () {0x;}' unless defined(&X86_FXSR_MAGIC);
    } else {
    }
}
1;
